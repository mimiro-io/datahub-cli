SHELL := /bin/bash
EXECUTABLE=datahub-cli
WINDOWS=$(EXECUTABLE)_windows_amd64.exe
LINUX=$(EXECUTABLE)_linux_amd64
DARWIN=$(EXECUTABLE)_darwin_amd64
VERSION=$(shell git describe --tags --always --long --dirty)

#.PHONY: all test clean
.PHONY: all clean

#all: test build ## Build and run tests
all: crossplatform test ## Build and run tests

#test: ## Run unit tests
#	./scripts/test_unit.sh

crossplatform: windows linux darwin ## Build binaries
	@echo version: $(VERSION)

windows: $(WINDOWS) ## Build for Windows

linux: $(LINUX) ## Build for Linux

darwin: $(DARWIN) ## Build for Darwin (macOS)

$(WINDOWS):
	env GOOS=windows GOARCH=amd64 go build -v -o bin/$(WINDOWS) -ldflags="-s -w -X main.version=$(VERSION)"  ./cmd/cli/main.go

$(LINUX):
	env GOOS=linux GOARCH=amd64 go build -v -o bin/$(LINUX) -ldflags="-s -w -X main.version=$(VERSION)"  ./cmd/cli/main.go

$(DARWIN):
	env GOOS=darwin GOARCH=amd64 go build -v -o bin/$(DARWIN) -ldflags="-s -w -X main.version=$(VERSION)"  ./cmd/cli/main.go

clean: ## Remove previous build
	rm -f $(WINDOWS) $(LINUX) $(DARWIN)

help: ## Display available commands
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

run:
	go run cmd/cli/main.go

bindata:
	GOBIN="$$PWD/bin" go install github.com/go-bindata/go-bindata/...
	./bin/go-bindata -pkg assets -o internal/assets/cow.go resources

build: bindata
	go build cmd/cli/main.go

test:
	go vet ./...
	go test ./... -v

license:
	go get -u github.com/google/addlicense; addlicense -c "MIMIRO AS" $(shell find . -iname "*.go")

mim:    bindata
	go build -o bin/mim ./cmd/cli/main.go


