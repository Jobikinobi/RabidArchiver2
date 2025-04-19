.PHONY: build test clean lint run macos

# Build variables
BINARY_NAME=archiver
BUILD_DIR=./bin
MAIN_PATH=./cmd/archiver

build:
	go build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)

macos: 
	GOOS=darwin GOARCH=arm64 go build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)

test:
	go test -v ./...

lint:
	go vet ./...
	@command -v staticcheck >/dev/null 2>&1 || go install honnef.co/go/tools/cmd/staticcheck@latest
	staticcheck ./...
	@command -v greptile >/dev/null 2>&1 || echo "Greptile not installed. Run: go install github.com/greptile/greptile/cmd/greptile@latest"
	@command -v greptile >/dev/null 2>&1 && greptile lint ./... || echo "Skipping greptile lint"

clean:
	rm -rf $(BUILD_DIR)
	rm -rf index/
	go clean -cache -testcache

run:
	$(BUILD_DIR)/$(BINARY_NAME) $(ARGS)

deps:
	go mod tidy
	go mod vendor

# Create directories
init:
	mkdir -p $(BUILD_DIR)
	mkdir -p index

.DEFAULT_GOAL := build 