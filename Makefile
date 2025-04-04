.PHONY: all build clean link

BINARY_NAME=model-cli

PLUGIN_DIR=$(HOME)/.docker/cli-plugins
PLUGIN_NAME=docker-model

VERSION ?=

all: build

build:
	@echo "Building $(BINARY_NAME)..."
	go build -ldflags="-s -w" -o $(BINARY_NAME) .

link:
	@if [ ! -f $(BINARY_NAME) ]; then \
		echo "Binary not found, building first..."; \
		$(MAKE) build; \
	else \
		echo "Using existing binary $(BINARY_NAME)"; \
	fi
	@echo "Linking $(BINARY_NAME) to Docker CLI plugins directory..."
	@mkdir -p $(PLUGIN_DIR)
	@ln -sf $(shell pwd)/$(BINARY_NAME) $(PLUGIN_DIR)/$(PLUGIN_NAME)
	@echo "Link created: $(PLUGIN_DIR)/$(PLUGIN_NAME)"

release:
	@if [ -z "$(VERSION)" ]; then \
		echo "Error: VERSION parameter is required. Use: make release VERSION=x.y.z"; \
		exit 1; \
	fi
	@echo "Building release version '$(VERSION)'..."
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w -X github.com/docker/model-cli/commands.Version=$(VERSION)" -o dist/darwin-arm64/$(PLUGIN_NAME) .
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w -X github.com/docker/model-cli/commands.Version=$(VERSION)" -o dist/windows-amd64/$(PLUGIN_NAME) .
	@echo "Release build complete: $(PLUGIN_NAME) version '$(VERSION)'"

clean:
	@echo "Cleaning up..."
	@rm -f $(BINARY_NAME)
	@echo "Cleaned!"
