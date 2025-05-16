.PHONY: all build clean link mock unit-tests docs

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

install: build link

release:
	@if [ -z "$(VERSION)" ]; then \
		echo "Error: VERSION parameter is required. Use: make release VERSION=x.y.z"; \
		exit 1; \
	fi
	@echo "Building release version '$(VERSION)'..."
	GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags="-s -w -X github.com/docker/model-cli/desktop.Version=$(VERSION)" -o dist/darwin-arm64/$(PLUGIN_NAME) .
	GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-s -w -X github.com/docker/model-cli/desktop.Version=$(VERSION)" -o dist/windows-amd64/$(PLUGIN_NAME).exe .
	GOOS=windows GOARCH=arm64 go build -trimpath -ldflags="-s -w -X github.com/docker/model-cli/desktop.Version=$(VERSION)" -o dist/windows-arm64/$(PLUGIN_NAME).exe .
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w -X github.com/docker/model-cli/desktop.Version=$(VERSION)" -o dist/linux-amd64/$(PLUGIN_NAME) .
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="-s -w -X github.com/docker/model-cli/desktop.Version=$(VERSION)" -o dist/linux-arm64/$(PLUGIN_NAME) .
	@echo "Release build complete: $(PLUGIN_NAME) version '$(VERSION)'"

ce-release:
	@if [ -z "$(VERSION)" ]; then \
		echo "Error: VERSION parameter is required. Use: make release VERSION=x.y.z"; \
		exit 1; \
	fi
	@if [ "$(uname -s)" != "Linux" ]; then \
		echo "Warning: This release target is designed for Linux"; \
	fi
	@echo "Building local release version '$(VERSION)'..."
	CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w -X github.com/docker/model-cli/desktop.Version=$(VERSION)" -o dist/$(PLUGIN_NAME) .
	@echo "Local release build complete: $(PLUGIN_NAME) version '$(VERSION)'"

mock:
	@echo "Generating mocks..."
	@mkdir -p mocks
	@go generate ./...
	@echo "Mocks generated!"

unit-tests:
	@echo "Running unit tests..."
	@go test -v ./...
	@echo "Unit tests completed!"

clean:
	@echo "Cleaning up..."
	@rm -f $(BINARY_NAME)
	@echo "Cleaned!"

docs:
	$(eval $@_TMP_OUT := $(shell mktemp -d -t model-cli-output.XXXXXXXXXX))
	docker buildx bake --set "*.output=type=local,dest=$($@_TMP_OUT)" update-docs
	rm -rf ./docs/reference/*
	cp -R "$($@_TMP_OUT)"/* ./docs/reference/
	rm -rf "$($@_TMP_OUT)"/*
