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

	return &App{
		cfg:     cfg,
		fetcher: articles.NewSource(cfg.HTTPClient),
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
