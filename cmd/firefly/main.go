package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/shoresh319/firefly/internal/app"
	"github.com/shoresh319/firefly/pkg/version"
)

func main() {
	log.Printf("starting firefly version=%s commit=%s built_at=%s", version.Version, version.Commit, version.BuiltAt)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	application := app.New(app.Config{
		TopWordNum:      10,
		WordBankPath:    filepath.Join("internal", "assets", "words.txt"),
		ArticleListPath: filepath.Join("internal", "assets", "endg-urls.txt"),
	})

	if err := application.Run(ctx, os.Stdout); err != nil {
		log.Fatalf("firefly execution failed: %v", err)
	}
}
