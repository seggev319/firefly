## Firefly

A Go application that processes articles from URLs, extracts valid words, and outputs the top N most frequent words as JSON.

**Quick start**

1. Build:
   ```bash
   make build
   ```
2. Run:
   ```bash
   ./bin/firefly
   ```
   Or:
   ```bash
   make run
   ```
   The application will process articles and output JSON to stdout.
3. Test:
   ```bash
   make test
   ```

**Configuration**

The application can be configured via `app.Config` in `cmd/firefly/main.go`:
- **TopWordNum**: Number of top words to return (default: 10)
- **WordBankPath**: Path to the word bank file
- **ArticleListPath**: Path to the article URL list file
- **WorkerCount**: Number of worker goroutines (0 = use default: runtime.NumCPU())
- **RetryMax**: Maximum number of HTTP retries (default: 3)
- **RetryWaitMin**: Minimum wait time between retries (default: 1s)
- **RetryWaitMax**: Maximum wait time between retries (default: 5s)
- **ConcurrencyPerDomain**: Maximum concurrent requests per domain (default: 3)

**Features**

- Concurrent article processing with configurable worker count
- Per-domain rate limiting to prevent overwhelming servers
- Automatic retry with exponential backoff for 429 (Too Many Requests) errors
- Respects `Retry-After` headers from servers
- HTML parsing to extract text content

**Version metadata**

Injected at build time (via `-ldflags`):
- `pkg/version.Version`
- `pkg/version.Commit`
- `pkg/version.BuiltAt`

**Docker**

Build and run:
```bash
docker build -t firefly:latest .
docker run --rm firefly:latest
```


