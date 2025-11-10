package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/shoresh319/firefly/pkg/version"
	"golang.org/x/net/html"
)

const topWordCount = 10

func main() {
	log.Printf("starting firefly version=%s commit=%s built_at=%s", version.Version, version.Commit, version.BuiltAt)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Load and sort the word bank
	wordBank, err := loadWordBank(ctx, "internal/assets/words.txt")
	if err != nil {
		log.Fatalf("failed to open word bank file: %v", err)
	}

	articleCh, err := loadArticleList(ctx, "internal/assets/endg-urls.txt")
	if err != nil {
		log.Fatalf("failed to open article list file: %v", err)
	}

	// Concurrent word counting across articles
	wordRegex := regexp.MustCompile(`\w+`)
	numWorkers := runtime.NumCPU()
	countsCh := make(chan map[string]int, numWorkers*2)
	var wg sync.WaitGroup

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for url := range articleCh {
				text, err := loadArticle(ctx, url)
				if err != nil {
					log.Printf("failed to load article %s: %v", url, err)
					continue
				}

				log.Printf("successfully loaded article from %s", url)
				local := make(map[string]int)
				// text := strings.ToLower(body)
				for _, token := range wordRegex.FindAllString(text, -1) {
					if validateWord(ctx, token, wordBank) {
						local[token]++
					}
				}
				if len(local) > 0 {
					select {
					case <-ctx.Done():
						return
					case countsCh <- local:
					}
				}
			}
		}()
	}

	globalCounts := make(map[string]int)
	doneMerge := make(chan struct{})
	go func() {
		for partial := range countsCh {
			for w, c := range partial {
				globalCounts[w] += c
			}
		}
		close(doneMerge)
	}()

	// Wait for workers to finish and close the reducer input
	wg.Wait()
	close(countsCh)
	<-doneMerge

	log.Printf("counted %d distinct valid words", len(globalCounts))

	// Keep only the top N words by frequency
	type kv struct {
		word  string
		count int
	}
	pairs := make([]kv, 0, len(globalCounts))
	for w, c := range globalCounts {
		pairs = append(pairs, kv{word: w, count: c})
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].count > pairs[j].count })
	if len(pairs) > topWordCount {
		pairs = pairs[:topWordCount]
	}
	topCounts := make(map[string]int, len(pairs))
	for _, p := range pairs {
		topCounts[p.word] = p.count
	}

	// Prevent unused variable warning and provide minimal visibility
	log.Printf("kept top %d words (distinct=%d)", topWordCount, len(topCounts))

	// Print the top words as pretty JSON to stdout
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(topCounts); err != nil {
		log.Fatalf("failed to encode top words as JSON: %v", err)
	}
}

func loadWordBank(ctx context.Context, filePath string) (map[string]struct{}, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	wordMap := make(map[string]struct{})

	buf := make([]byte, 4096)
	content := []byte{}
	for {
		n, err := f.Read(buf)
		content = append(content, buf[:n]...)
		if err != nil {
			break
		}
	}
	lines := string(content)
	for _, line := range strings.Split(lines, "\n") {
		w := strings.TrimSpace(line)
		if w != "" {
			wordMap[w] = struct{}{}
		}
	}

	return wordMap, nil
}

func loadArticleList(ctx context.Context, filePath string) (<-chan string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	out := make(chan string)
	go func() {
		defer close(out)
		defer f.Close()
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			default:
			}
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			out <- line
		}
		// If there's a scanning error, log it. Channel will be closed by defer.
		if err := scanner.Err(); err != nil {
			log.Printf("error reading article list from %s: %v", filePath, err)
		}
	}()
	return out, nil
}

func loadArticle(ctx context.Context, url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch article: status code %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Parse the HTML and extract the main text content
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML: %w", err)
	}
	var textBuilder strings.Builder
	var crawler func(*html.Node)
	crawler = func(n *html.Node) {
		if n.Type == html.TextNode {
			trimmed := strings.TrimSpace(n.Data)
			if trimmed != "" {
				textBuilder.WriteString(trimmed)
				textBuilder.WriteByte('\n')
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			crawler(c)
		}
	}
	crawler(doc)
	return textBuilder.String(), nil
}

func validateWord(ctx context.Context, word string, wordBank map[string]struct{}) bool {
	matched, err := regexp.MatchString(`^\w{3,}$`, word)
	if err != nil || !matched {
		return false
	}
	_, ok := wordBank[word]
	return ok
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
