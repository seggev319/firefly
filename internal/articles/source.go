package articles

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"golang.org/x/net/html"
)

// SourceConfig holds configuration for the Source.
type SourceConfig struct {
	HTTPClient           *http.Client
	RetryMax             int
	RetryWaitMin         time.Duration
	RetryWaitMax         time.Duration
	ConcurrencyPerDomain int // Maximum concurrent requests per domain (default: 3)
}

// Source fetches article content via HTTP with retry support for 429 errors.
type Source struct {
	client               *retryablehttp.Client
	domainSemaphores     map[string]chan struct{} // Semaphore per domain for concurrency control
	mu                   sync.RWMutex
	concurrencyPerDomain int
}

// NewSource constructs a Source with retryable HTTP client configured for 429 handling.
func NewSource(cfg SourceConfig) *Source {
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{
			Timeout: 15 * time.Second,
		}
	}

	if cfg.ConcurrencyPerDomain <= 0 {
		cfg.ConcurrencyPerDomain = 3 // Default: 3 concurrent requests per domain
	}

	retryClient := retryablehttp.NewClient()
	retryClient.HTTPClient = cfg.HTTPClient
	retryClient.RetryMax = cfg.RetryMax
	retryClient.RetryWaitMin = cfg.RetryWaitMin
	retryClient.RetryWaitMax = cfg.RetryWaitMax
	retryClient.CheckRetry = func(ctx context.Context, resp *http.Response, err error) (bool, error) {
		// Retry on 429 (Too Many Requests) errors
		if resp != nil && resp.StatusCode == http.StatusTooManyRequests {
			return true, nil
		}
		// Use default retry logic for other retryable errors
		return retryablehttp.DefaultRetryPolicy(ctx, resp, err)
	}
	retryClient.Backoff = func(min, max time.Duration, attemptNum int, resp *http.Response) time.Duration {
		// For 429 errors, use exponential backoff with jitter
		if resp != nil && resp.StatusCode == http.StatusTooManyRequests {
			// Check for Retry-After header (value is in seconds)
			if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
				if seconds, err := strconv.Atoi(retryAfter); err == nil {
					duration := time.Duration(seconds) * time.Second
					// Cap at max, but don't go below min
					if duration > max {
						return max
					}
					if duration < min {
						return min
					}
					return duration
				}
			}
			// Exponential backoff: 2^attemptNum seconds, capped at max
			backoff := time.Duration(1<<uint(attemptNum)) * time.Second
			if backoff > max {
				backoff = max
			}
			if backoff < min {
				backoff = min
			}
			return backoff
		}
		// Default exponential backoff for other errors
		return retryablehttp.DefaultBackoff(min, max, attemptNum, resp)
	}

	return &Source{
		client:               retryClient,
		domainSemaphores:     make(map[string]chan struct{}),
		concurrencyPerDomain: cfg.ConcurrencyPerDomain,
	}
}

// getDomainSemaphore returns a semaphore for the given domain to limit concurrent requests.
func (s *Source) getDomainSemaphore(domain string) chan struct{} {
	s.mu.RLock()
	sem, exists := s.domainSemaphores[domain]
	s.mu.RUnlock()

	if exists {
		return sem
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Double-check after acquiring write lock
	if sem, exists := s.domainSemaphores[domain]; exists {
		return sem
	}

	// Create a buffered channel as a semaphore
	// The channel capacity limits concurrent requests
	sem = make(chan struct{}, s.concurrencyPerDomain)
	// Pre-fill the semaphore with tokens to allow initial concurrent requests
	for i := 0; i < s.concurrencyPerDomain; i++ {
		sem <- struct{}{}
	}
	s.domainSemaphores[domain] = sem
	return sem
}

// extractDomain extracts the domain from a URL.
func extractDomain(rawURL string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("parse URL: %w", err)
	}
	return parsed.Hostname(), nil
}

// Fetch retrieves the textual content of the article located at url.
// It handles 429 errors with retries, using per-domain semaphores to limit
// concurrent requests while allowing multiple workers per domain.
func (s *Source) Fetch(ctx context.Context, urlStr string) (string, error) {
	domain, err := extractDomain(urlStr)
	if err != nil {
		return "", err
	}

	// Acquire semaphore slot for this domain (allows N concurrent requests)
	sem := s.getDomainSemaphore(domain)
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-sem: // Acquire semaphore
		defer func() { sem <- struct{}{} }() // Release semaphore when done
	}

	req, err := retryablehttp.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}

	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("parse HTML: %w", err)
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
