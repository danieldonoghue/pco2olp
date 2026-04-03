BINARY := pco2olp
BINARY_GUI := pco2olp-gui
VERSION := $(shell git describe --tags --exact-match 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
BUILD_DATE := $(shell date -u +"%Y-%m-%d")
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildTime=$(BUILD_TIME)"

# For org-specific builds with baked-in OAuth credentials and org name:
#   make build-org PCO_CLIENT_ID=xxx PCO_CLIENT_SECRET=yyy ORG_NAME="My Church"
AUTH_PKG := github.com/danieldonoghue/pco2olp/internal/auth
ORG_LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildTime=$(BUILD_TIME) -X $(AUTH_PKG).defaultClientID=$(PCO_CLIENT_ID) -X $(AUTH_PKG).defaultClientSecret=$(PCO_CLIENT_SECRET) -X 'main.orgName=$(ORG_NAME)'"

PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

# macOS app bundle settings
BUNDLE_ID   := io.github.danieldonoghue.pco2olp
_VER_STRIPPED := $(patsubst v%,%,$(VERSION))
APP_VERSION   := $(if $(filter dev,$(_VER_STRIPPED)),0.0.0,$(_VER_STRIPPED))
ICON_PNG    := $(CURDIR)/Icon.png
SVG_ICON    := $(CURDIR)/resources/app-icon.svg

.PHONY: build build-org build-gui build-gui-org clean test lint all \
        release release-org release-gui release-gui-org convert-icon

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

build-gui:
	CGO_ENABLED=1 go build -tags gui $(LDFLAGS) -o $(BINARY_GUI) .

build-gui-org:
ifndef PCO_CLIENT_ID
	$(error PCO_CLIENT_ID is required for build-gui-org)
endif
ifndef PCO_CLIENT_SECRET
	$(error PCO_CLIENT_SECRET is required for build-gui-org)
endif
	CGO_ENABLED=1 go build -tags gui $(ORG_LDFLAGS) -o $(BINARY_GUI) .

test:
	go test ./...

lint:
	go vet ./...

clean:
	rm -f $(BINARY) $(BINARY_GUI)
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

# ── macOS GUI .app bundle (local build) ───────────────────────────────────────
# Builds a universal (arm64 + amd64) macOS .app bundle.
# Requires: fyne CLI  (go install fyne.io/fyne/v2/cmd/fyne@latest)
# Requires: brew install librsvg  (rsvg-convert, for Icon.png auto-generation)

release-gui:
	@mkdir -p dist
	@rm -rf dist/pco2olp.app dist/pco2olp-gui-*-darwin-*.zip
	@[ -f $(ICON_PNG) ] || rsvg-convert -w 1024 -h 1024 $(SVG_ICON) -o $(ICON_PNG)
	CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -tags gui $(LDFLAGS) -o dist/_gui_arm64 .
	CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -tags gui $(LDFLAGS) -o dist/_gui_amd64 .
	lipo -create dist/_gui_arm64 dist/_gui_amd64 -output dist/_gui_universal
	@rm -f dist/_gui_arm64 dist/_gui_amd64
	cd dist && fyne package --executable _gui_universal --name pco2olp --appID $(BUNDLE_ID) --appVersion $(APP_VERSION) --icon $(ICON_PNG)
	@rm -f dist/_gui_universal
	@printf '<html><body><p>Converts Planning Center Online service plans to OpenLP service files.</p><p>By <a href="https://github.com/danieldonoghue">Daniel Donoghue</a> &mdash; <a href="https://github.com/danieldonoghue/pco2olp">github.com/danieldonoghue/pco2olp</a></p><p>Build: %s &middot; %s</p></body></html>' \
		'$(COMMIT)' '$(BUILD_DATE)' > dist/pco2olp.app/Contents/Resources/Credits.html
	cd dist && zip -r pco2olp-gui-$(VERSION)-darwin-universal.zip pco2olp.app
	@echo "Built dist/pco2olp-gui-$(VERSION)-darwin-universal.zip"
	@ls -la dist/pco2olp-gui-$(VERSION)-darwin-universal.zip

release-gui-org:
ifndef PCO_CLIENT_ID
	$(error PCO_CLIENT_ID is required for release-gui-org)
endif
ifndef PCO_CLIENT_SECRET
	$(error PCO_CLIENT_SECRET is required for release-gui-org)
endif
	@mkdir -p dist
	@rm -rf dist/pco2olp.app dist/pco2olp-gui-*-darwin-*.zip
	@[ -f $(ICON_PNG) ] || rsvg-convert -w 1024 -h 1024 $(SVG_ICON) -o $(ICON_PNG)
	CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -tags gui $(ORG_LDFLAGS) -o dist/_gui_arm64 .
	CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -tags gui $(ORG_LDFLAGS) -o dist/_gui_amd64 .
	lipo -create dist/_gui_arm64 dist/_gui_amd64 -output dist/_gui_universal
	@rm -f dist/_gui_arm64 dist/_gui_amd64
	cd dist && fyne package --executable _gui_universal --name pco2olp --appID $(BUNDLE_ID) --appVersion $(APP_VERSION) --icon $(ICON_PNG)
	@rm -f dist/_gui_universal
	@printf '<html><body><p><b>Built for:</b> %s</p><p>Converts Planning Center Online service plans to OpenLP service files.</p><p>By <a href="https://github.com/danieldonoghue">Daniel Donoghue</a> &mdash; <a href="https://github.com/danieldonoghue/pco2olp">github.com/danieldonoghue/pco2olp</a></p><p>Build: %s &middot; %s</p></body></html>' \
		'$(ORG_NAME)' '$(COMMIT)' '$(BUILD_DATE)' > dist/pco2olp.app/Contents/Resources/Credits.html
	cd dist && zip -r pco2olp-gui-$(VERSION)-darwin-universal.zip pco2olp.app
	@echo "Built dist/pco2olp-gui-$(VERSION)-darwin-universal.zip"
	@ls -la dist/pco2olp-gui-$(VERSION)-darwin-universal.zip

# Convert resources/app-icon.svg → Icon.png (requires: brew install librsvg)
convert-icon:
	rsvg-convert -w 1024 -h 1024 $(SVG_ICON) -o $(ICON_PNG)
	@echo "Created Icon.png"
