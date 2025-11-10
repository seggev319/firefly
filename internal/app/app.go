package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/shoresh319/firefly/internal/articles"
	"github.com/shoresh319/firefly/internal/processing"
	"github.com/shoresh319/firefly/internal/wordbank"
)

// Config encapsulates runtime configuration for the application.
type Config struct {
	WordBankPath    string
	ArticleListPath string
	TopWordNum      int
	HTTPClient      *http.Client
	WorkerCount     int
	// Retry configuration for HTTP requests
	RetryMax     int           // Maximum number of retries (default: 3)
	RetryWaitMin time.Duration // Minimum wait time between retries (default: 1s)
	RetryWaitMax time.Duration // Maximum wait time between retries (default: 5s)
	// Concurrency configuration
	ConcurrencyPerDomain int // Maximum concurrent requests per domain (default: 3)
}

// App glues together input sources, processors and outputs.
type App struct {
	cfg     Config
	fetcher *articles.Source
}

// New constructs a new App with the provided configuration.
func New(cfg Config) *App {
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{
			Timeout: 15 * time.Second,
		}
	}

	// Set default retry configuration if not provided
	if cfg.RetryMax == 0 {
		cfg.RetryMax = 3
	}
	if cfg.RetryWaitMin == 0 {
		cfg.RetryWaitMin = 1 * time.Second
	}
	if cfg.RetryWaitMax == 0 {
		cfg.RetryWaitMax = 5 * time.Second
	}
	if cfg.ConcurrencyPerDomain == 0 {
		cfg.ConcurrencyPerDomain = 3
	}

	return &App{
		cfg: cfg,
		fetcher: articles.NewSource(articles.SourceConfig{
			HTTPClient:           cfg.HTTPClient,
			RetryMax:             cfg.RetryMax,
			RetryWaitMin:         cfg.RetryWaitMin,
			RetryWaitMax:         cfg.RetryWaitMax,
			ConcurrencyPerDomain: cfg.ConcurrencyPerDomain,
		}),
	}
}

// Run executes the application and writes the resulting JSON payload to out.
func (a *App) Run(ctx context.Context, out io.Writer) error {
	wordBank, err := wordbank.Load(ctx, a.cfg.WordBankPath)
	if err != nil {
		return fmt.Errorf("load word bank from %s: %w", a.cfg.WordBankPath, err)
	}

	urlCh, err := articles.ListFromFile(ctx, a.cfg.ArticleListPath)
	if err != nil {
		return fmt.Errorf("load article list from %s: %w", a.cfg.ArticleListPath, err)
	}

	validator := wordbank.NewValidator(wordBank)
	options := []processing.Option{}
	if a.cfg.WorkerCount > 0 {
		options = append(options, processing.WithWorkerCount(a.cfg.WorkerCount))
	}
	counter := processing.NewCounter(a.fetcher, validator, options...)

	topCounts, err := counter.CountTopWords(ctx, urlCh, a.cfg.TopWordNum)
	if err != nil {
		return fmt.Errorf("count top words: %w", err)
	}

	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(topCounts); err != nil {
		return fmt.Errorf("encode result: %w", err)
	}

	return nil
}
