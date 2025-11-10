APP_NAME := firefly
BIN_DIR := bin
GO_FILES := $(shell find . -name '*.go' -not -path "./vendor/*")

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILTAT ?= $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
LDFLAGS := -X github.com/seggev/firefly/pkg/version.Version=$(VERSION) -X github.com/seggev/firefly/pkg/version.Commit=$(COMMIT) -X github.com/seggev/firefly/pkg/version.BuiltAt=$(BUILTAT)

.PHONY: all build run test clean

all: build

build: $(GO_FILES)
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BIN_DIR)
	@go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(APP_NAME) ./cmd/firefly
	@echo "Built at $(BIN_DIR)/$(APP_NAME)"

run:
	@PORT?=8080
	@echo "Running on :$${PORT} ..."
	@go run -ldflags "$(LDFLAGS)" ./cmd/firefly

test:
	@go test ./...

clean:
	@rm -rf $(BIN_DIR)


