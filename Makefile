BINARY := pco2olp
VERSION := $(shell git describe --tags --exact-match 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildTime=$(BUILD_TIME)"

# For org-specific builds with baked-in OAuth credentials and org name:
#   make build-org PCO_CLIENT_ID=xxx PCO_CLIENT_SECRET=yyy ORG_NAME="My Church"
AUTH_PKG := github.com/danield/pco2olp/internal/auth
ORG_LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildTime=$(BUILD_TIME) -X $(AUTH_PKG).defaultClientID=$(PCO_CLIENT_ID) -X $(AUTH_PKG).defaultClientSecret=$(PCO_CLIENT_SECRET) -X 'main.orgName=$(ORG_NAME)'"

PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

.PHONY: build build-org clean test lint all release release-org

build:
	go build $(LDFLAGS) -o $(BINARY) .

build-org:
ifndef PCO_CLIENT_ID
	$(error PCO_CLIENT_ID is required for build-org)
endif
ifndef PCO_CLIENT_SECRET
	$(error PCO_CLIENT_SECRET is required for build-org)
endif
	go build $(ORG_LDFLAGS) -o $(BINARY) .

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
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build $(ACTIVE_LDFLAGS) -o dist/$(BINARY)-$(GOOS)-$(GOARCH)$(EXT) .

release: ACTIVE_LDFLAGS = $(LDFLAGS)
release: clean all
	@echo "Built binaries in dist/"
	@ls -la dist/

release-org: ACTIVE_LDFLAGS = $(ORG_LDFLAGS)
release-org: clean all
ifndef PCO_CLIENT_ID
	$(error PCO_CLIENT_ID is required for release-org)
endif
ifndef PCO_CLIENT_SECRET
	$(error PCO_CLIENT_SECRET is required for release-org)
endif
	@echo "Built org binaries (with baked-in credentials) in dist/"
	@ls -la dist/
