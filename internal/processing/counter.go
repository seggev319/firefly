package processing

import (
	"context"
	"log"
	"regexp"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
)

// ArticleFetcher returns the textual content for a given article URL.
type ArticleFetcher interface {
	Fetch(ctx context.Context, url string) (string, error)
}

// WordValidator determines if a token should be counted.
type WordValidator interface {
	Validate(word string) bool
}

// Counter orchestrates concurrent word counting for a series of articles.
type Counter struct {
	fetcher   ArticleFetcher
	validator WordValidator
	wordRegex *regexp.Regexp
	workers   int
}

// Option configures a Counter.
type Option func(*Counter)

// WithWorkerCount overrides the default worker count.
func WithWorkerCount(workers int) Option {
	return func(c *Counter) {
		if workers > 0 {
			c.workers = workers
		}
	}
}

// WithWordRegex overrides the default token extraction expression.
func WithWordRegex(expr *regexp.Regexp) Option {
	return func(c *Counter) {
		if expr != nil {
			c.wordRegex = expr
		}
	}
}

// NewCounter constructs a Counter with optional configuration.
func NewCounter(fetcher ArticleFetcher, validator WordValidator, opts ...Option) *Counter {
	counter := &Counter{
		fetcher:   fetcher,
		validator: validator,
		wordRegex: regexp.MustCompile(`\w+`),
		workers:   runtime.NumCPU(),
	}

	for _, opt := range opts {
		opt(counter)
	}

	return counter
}

// CountTopWords loads articles from the provided URL channel and returns a map
// containing the topN tokens by frequency.
func (c *Counter) CountTopWords(ctx context.Context, urlCh <-chan string, topN int) (map[string]int, error) {
	countsCh := make(chan map[string]int, c.workers*2)
	var wg sync.WaitGroup
	var successes, failures int64

	for i := 0; i < c.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case url, ok := <-urlCh:
					if !ok {
						return
					}
					if c.processURL(ctx, url, countsCh) {
						atomic.AddInt64(&successes, 1)
					} else {
						atomic.AddInt64(&failures, 1)
					}
				}
			}
		}()
	}

	globalCounts := make(map[string]int)
	doneMerge := make(chan struct{})
	go func() {
		for partial := range countsCh {
			for token, count := range partial {
				globalCounts[token] += count
			}
		}
		close(doneMerge)
	}()

	wg.Wait()
	close(countsCh)
	<-doneMerge

	log.Printf("processed articles: %d successes, %d failures", atomic.LoadInt64(&successes), atomic.LoadInt64(&failures))
	log.Printf("counted %d distinct valid words", len(globalCounts))

	topCounts := pickTop(globalCounts, topN)
	log.Printf("kept top %d words (distinct=%d)", topN, len(topCounts))

	return topCounts, nil
}

func (c *Counter) processURL(ctx context.Context, url string, countsCh chan<- map[string]int) bool {
	text, err := c.fetcher.Fetch(ctx, url)
	if err != nil {
		log.Printf("failed to load article %s: %v", url, err)
		return false
	}

	local := make(map[string]int)
	for _, token := range c.wordRegex.FindAllString(text, -1) {
		if c.validator.Validate(token) {
			local[token]++
		}
	}

	if len(local) == 0 {
		return true
	}

	select {
	case <-ctx.Done():
		return true
	case countsCh <- local:
		return true
	}
}

func pickTop(globalCounts map[string]int, topN int) map[string]int {
	if topN <= 0 || len(globalCounts) == 0 {
		return map[string]int{}
	}

	type kv struct {
		word  string
		count int
	}

	pairs := make([]kv, 0, len(globalCounts))
	for word, count := range globalCounts {
		pairs = append(pairs, kv{word: word, count: count})
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].count > pairs[j].count
	})

	if len(pairs) > topN {
		pairs = pairs[:topN]
	}

	topCounts := make(map[string]int, len(pairs))
	for _, pair := range pairs {
		topCounts[pair.word] = pair.count
	}

	return topCounts
}
