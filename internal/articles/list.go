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
func ListFromFile(ctx context.Context, filePath string) (<-chan string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open article list: %w", err)
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
