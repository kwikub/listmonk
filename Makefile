BINARY=listmonk
VERSION=$(shell git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
BUILD_DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

LDFLAGS=-ldflags "-X 'main.buildVersion=$(VERSION)' -X 'main.buildDate=$(BUILD_DATE)' -X 'main.buildCommit=$(COMMIT)'"

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Frontend
YARN=yarn
FRONTEND_DIR=frontend

# Default config file path (override with: make run CONFIG=path/to/config.toml)
CONFIG=config.toml

.PHONY: all build build-backend build-frontend clean test deps run dev

## Default target: build everything
all: deps build

## Build both backend and frontend
build: build-backend build-frontend

## Build only the Go backend binary
build-backend:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY) ./cmd/

## Build the Vue.js frontend
build-frontend:
	cd $(FRONTEND_DIR) && $(YARN) install && $(YARN) build

## Run Go tests
test:
	$(GOTEST) -v ./...

## Download Go module dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

## Clean build artifacts
clean:
	$(GOCLEAN)
	rm -f $(BINARY)
	rm -rf $(FRONTEND_DIR)/dist

## Run the application (requires a config file)
run: build-backend
	./$(BINARY) --config $(CONFIG)

## Start backend in dev/watch mode (requires air: github.com/cosmtrek/air)
dev-backend:
	air -c .air.toml

## Start frontend dev server
dev-frontend:
	cd $(FRONTEND_DIR) && $(YARN) dev

## Build a Docker image
docker:
	docker build -t listmonk:$(VERSION) .

## Generate a sample config file
config:
	./$(BINARY) --new-config

## Run database migrations
migrate: build-backend
	./$(BINARY) --config $(CONFIG) --upgrade

## Install the binary to $GOPATH/bin
install:
	$(GOBUILD) $(LDFLAGS) -o $(GOPATH)/bin/$(BINARY) ./cmd/

## Print version info
version:
	@echo "Version:    $(VERSION)"
	@echo "Commit:     $(COMMIT)"
	@echo "Build date: $(BUILD_DATE)"

help:
	@echo "Available targets:"
	@grep -E '^##' Makefile | sed 's/## /  /'
