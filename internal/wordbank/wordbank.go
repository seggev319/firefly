package wordbank

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// Validator checks whether a token is considered a valid word and exists in the
// previously loaded word bank.
type Validator struct {
	words map[string]struct{}

	wordMatcher *regexp.Regexp
}

// Load reads the word bank from the supplied file path and returns it as a set.
func Load(ctx context.Context, filePath string) (map[string]struct{}, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open word bank: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 4096), 1024*1024)

	words := make(map[string]struct{})
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		w := strings.TrimSpace(scanner.Text())
		if w == "" {
			continue
		}
		words[w] = struct{}{}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan word bank: %w", err)
	}

	return words, nil
}

// NewValidator constructs a validator for the supplied in-memory word bank.
func NewValidator(words map[string]struct{}) *Validator {
	return &Validator{
		words:       words,
		wordMatcher: regexp.MustCompile(`^\w{3,}$`),
	}
}

// Validate returns true when the provided token matches the configured word
// pattern and exists in the word bank.
func (v *Validator) Validate(word string) bool {
	if !v.wordMatcher.MatchString(word) {
		return false
	}

	_, ok := v.words[word]
	return ok
}
