GOCMD = go
GOBUILD = $(GOCMD) build
GOTEST = $(GOCMD) test
GOCLEAN = $(GOCMD) clean
BINARY_NAME = mr-metrics
BUILD_DIR = bin
GO_PACKAGES = ./...

.PHONY: all build test clean lint help

all: build
	@$(MAKE) test

build:
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -v -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/app

test:
	$(GOTEST) -v -cover $(GO_PACKAGES)

clean:
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)

lint:
	golangci-lint run

help:
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)