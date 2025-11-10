package articles

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
)

// ListFromFile streams article URLs read from the provided file path.
// It reads all lines from the file, but respects context cancellation when sending.
func ListFromFile(ctx context.Context, filePath string) (<-chan string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open article list: %w", err)
	}

	// Use a buffered channel to prevent blocking the file reader
	out := make(chan string, 1000)
	go func() {
		defer close(out)
		defer f.Close()

		scanner := bufio.NewScanner(f)
		// Increase buffer size to handle any unusually long lines
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, 1024*1024) // 1MB max line length

		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}

			// Try to send the line, but respect context cancellation
			select {
			case <-ctx.Done():
				return
			case out <- line:
			}
		}

		if err := scanner.Err(); err != nil {
			log.Printf("error reading article list from %s: %v", filePath, err)
		}
	}()

	return out, nil
}
