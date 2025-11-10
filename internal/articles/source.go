package articles

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"golang.org/x/net/html"
)

// Source fetches article content via HTTP.
type Source struct {
	client *http.Client
}

// NewSource constructs a Source. When client is nil the default HTTP client is
// used.
func NewSource(client *http.Client) *Source {
	if client == nil {
		client = http.DefaultClient
	}
	return &Source{client: client}
}

// Fetch retrieves the textual content of the article located at url.
func (s *Source) Fetch(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
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
