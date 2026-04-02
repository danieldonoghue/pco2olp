BINARY := pco2olp
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildTime=$(BUILD_TIME)"

PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

.PHONY: build clean test lint all release

build:
	go build $(LDFLAGS) -o $(BINARY) .

test:
	go test ./...

lint:
	go vet ./...

clean:
	rm -f $(BINARY)
	rm -rf dist/

all: $(PLATFORMS)

$(PLATFORMS):
	$(eval GOOS := $(word 1, $(subst /, ,$@)))
	$(eval GOARCH := $(word 2, $(subst /, ,$@)))
	$(eval EXT := $(if $(filter windows,$(GOOS)),.exe,))
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build $(LDFLAGS) -o dist/$(BINARY)-$(GOOS)-$(GOARCH)$(EXT) .

release: clean all
	@echo "Built binaries in dist/"
	@ls -la dist/
