# OVAV Go Runtime — Build System
# CAPA 9.5: Cross-compile para linux/darwin/windows amd64/arm64.
# Stack definitivo: Go 1.24+, stdlib cpanel/ovav; Bubble Tea para Cockpit.

# ── Build metadata ────────────────────────────────────────────────────────────
VERSION     ?= 0.0.0-dev
BUILD       ?= CAPA 9 — Go Runtime
GIT_BRANCH  := $(shell git branch --show-current 2>/dev/null || echo "unknown")
GIT_SHA     := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME  := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS     := -s -w \
	-X main.Version=$(VERSION) \
	-X main.Build=$(BUILD) \
	-X main.GitBranch=$(GIT_BRANCH) \
	-X main.GitSHA=$(GIT_SHA) \
	-X main.BuildTime=$(BUILD_TIME)

# ── Targets ───────────────────────────────────────────────────────────────────
GOOS        ?= $(shell go env GOOS)
GOARCH      ?= $(shell go env GOARCH)
OUTPUT_DIR  := build
BINARY_NAME := ovav

# Default target — builds ovav CLI + cockpit TUI for current platform
.PHONY: all
all: clean build-current build-cockpit
	@echo ""
	@echo "✅ Build complete: ovav + cockpit TUI. Binaries in $(OUTPUT_DIR)/"
	@echo "   Run 'make install' to install to ~/.local/bin/"

# ── Build ─────────────────────────────────────────────────────────────────────

.PHONY: build
build: build-linux build-darwin build-windows
	@echo "✅ Build complete. Binaries in $(OUTPUT_DIR)/"

.PHONY: build-current
build-current:
	@echo "🔨 Building for $(GOOS)/$(GOARCH)..."
	@mkdir -p $(OUTPUT_DIR)
	CGO_ENABLED=0 go build \
		-ldflags "$(LDFLAGS)" \
		-o $(OUTPUT_DIR)/$(BINARY_NAME) \
		./cmd/ovav/
	@echo "✅ $(OUTPUT_DIR)/$(BINARY_NAME)"

.PHONY: build-linux
build-linux:
	@echo "🔨 Building linux/amd64..."
	@mkdir -p $(OUTPUT_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
		-ldflags "$(LDFLAGS)" \
		-o $(OUTPUT_DIR)/$(BINARY_NAME)-linux-amd64 \
		./cmd/ovav/
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build \
		-ldflags "$(LDFLAGS)" \
		-o $(OUTPUT_DIR)/$(BINARY_NAME)-linux-arm64 \
		./cmd/ovav/
	@echo "✅ Linux binaries built"

.PHONY: build-darwin
build-darwin:
	@echo "🔨 Building darwin/amd64 + arm64..."
	@mkdir -p $(OUTPUT_DIR)
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build \
		-ldflags "$(LDFLAGS)" \
		-o $(OUTPUT_DIR)/$(BINARY_NAME)-darwin-amd64 \
		./cmd/ovav/
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build \
		-ldflags "$(LDFLAGS)" \
		-o $(OUTPUT_DIR)/$(BINARY_NAME)-darwin-arm64 \
		./cmd/ovav/
	@echo "✅ macOS binaries built"

.PHONY: build-windows
build-windows:
	@echo "🔨 Building windows/amd64..."
	@mkdir -p $(OUTPUT_DIR)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build \
		-ldflags "$(LDFLAGS)" \
		-o $(OUTPUT_DIR)/$(BINARY_NAME)-windows-amd64.exe \
		./cmd/ovav/
	@echo "✅ Windows binary built"

# ── cPanel ─────────────────────────────────────────────────────────────────────

CPANEL_BINARY := cpanel
CPANEL_LDFLAGS := -s -w \
	-X 'main.Version=$(VERSION)' \
	-X 'main.Build=cPanel v5 - Go Runtime' \
	-X 'main.GitBranch=$(GIT_BRANCH)' \
	-X 'main.GitSHA=$(GIT_SHA)' \
	-X 'main.BuildTime=$(BUILD_TIME)'

.PHONY: build-cpanel
build-cpanel:
	@echo "🔨 Building cPanel v5.0 for $(GOOS)/$(GOARCH)..."
	@mkdir -p $(OUTPUT_DIR)
	CGO_ENABLED=0 go build \
		-ldflags "$(CPANEL_LDFLAGS)" \
		-o $(OUTPUT_DIR)/$(CPANEL_BINARY) \
		./cmd/cpanel/
	@echo "✅ $(OUTPUT_DIR)/$(CPANEL_BINARY)"

.PHONY: build-cpanel-all
build-cpanel-all:
	@echo "🔨 Building cPanel for all platforms..."
	@mkdir -p $(OUTPUT_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "$(CPANEL_LDFLAGS)" -o $(OUTPUT_DIR)/cpanel-linux-amd64 ./cmd/cpanel/
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags "$(CPANEL_LDFLAGS)" -o $(OUTPUT_DIR)/cpanel-linux-arm64 ./cmd/cpanel/
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags "$(CPANEL_LDFLAGS)" -o $(OUTPUT_DIR)/cpanel-darwin-amd64 ./cmd/cpanel/
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags "$(CPANEL_LDFLAGS)" -o $(OUTPUT_DIR)/cpanel-darwin-arm64 ./cmd/cpanel/
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "$(CPANEL_LDFLAGS)" -o $(OUTPUT_DIR)/cpanel-windows-amd64.exe ./cmd/cpanel/
	@echo "✅ cPanel cross-compiled for 5 targets"

.PHONY: build-all
build-all: build build-cpanel build-cockpit
	@echo "✅ All binaries built. OVAV + cPanel + Cockpit in $(OUTPUT_DIR)/"

# ── Tailor ─────────────────────────────────────────────────────────────────────

TAILOR_BINARY := tailor
TAILOR_LDFLAGS := -s -w \
	-X 'main.Version=$(VERSION)' \
	-X 'main.Build=Tailor v1.0 - Go' \
	-X 'main.GitSHA=$(GIT_SHA)' \
	-X 'main.BuildTime=$(BUILD_TIME)'

.PHONY: build-tailor
build-tailor:
	@echo "🔨 Building Tailor for $(GOOS)/$(GOARCH)..."
	@mkdir -p $(OUTPUT_DIR)
	CGO_ENABLED=0 go build \
		-ldflags "$(TAILOR_LDFLAGS)" \
		-o $(OUTPUT_DIR)/$(TAILOR_BINARY) \
		./cmd/tailor/
	@echo "✅ $(OUTPUT_DIR)/$(TAILOR_BINARY)"

.PHONY: build-tailor-all
build-tailor-all:
	@echo "🔨 Building Tailor for all platforms..."
	@mkdir -p $(OUTPUT_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "$(TAILOR_LDFLAGS)" -o $(OUTPUT_DIR)/tailor-linux-amd64 ./cmd/tailor/
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags "$(TAILOR_LDFLAGS)" -o $(OUTPUT_DIR)/tailor-linux-arm64 ./cmd/tailor/
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags "$(TAILOR_LDFLAGS)" -o $(OUTPUT_DIR)/tailor-darwin-amd64 ./cmd/tailor/
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags "$(TAILOR_LDFLAGS)" -o $(OUTPUT_DIR)/tailor-darwin-arm64 ./cmd/tailor/
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "$(TAILOR_LDFLAGS)" -o $(OUTPUT_DIR)/tailor-windows-amd64.exe ./cmd/tailor/
	@echo "✅ Tailor cross-compiled for 5 targets"

# ── Cockpit TUI (Bubble Tea) ─────────────────────────────────────────────────

COCKPIT_BINARY := cockpit
COCKPIT_LDFLAGS := -s -w \
	-X 'main.Version=$(VERSION)' \
	-X 'main.Build=Cockpit v1.0 - Go TUI' \
	-X 'main.GitSHA=$(GIT_SHA)' \
	-X 'main.BuildTime=$(BUILD_TIME)'

.PHONY: build-cockpit
build-cockpit:
	@echo "🔨 Building Cockpit TUI v1.0 for $(GOOS)/$(GOARCH)..."
	@mkdir -p $(OUTPUT_DIR)
	CGO_ENABLED=0 go build \
		-ldflags "$(COCKPIT_LDFLAGS)" \
		-o $(OUTPUT_DIR)/$(COCKPIT_BINARY) \
		./cmd/cockpit/
	@echo "✅ $(OUTPUT_DIR)/$(COCKPIT_BINARY)"

.PHONY: build-cockpit-all
build-cockpit-all:
	@echo "🔨 Building Cockpit for all platforms..."
	@mkdir -p $(OUTPUT_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "$(COCKPIT_LDFLAGS)" -o $(OUTPUT_DIR)/cockpit-linux-amd64 ./cmd/cockpit/
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags "$(COCKPIT_LDFLAGS)" -o $(OUTPUT_DIR)/cockpit-linux-arm64 ./cmd/cockpit/
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags "$(COCKPIT_LDFLAGS)" -o $(OUTPUT_DIR)/cockpit-darwin-amd64 ./cmd/cockpit/
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags "$(COCKPIT_LDFLAGS)" -o $(OUTPUT_DIR)/cockpit-darwin-arm64 ./cmd/cockpit/
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "$(COCKPIT_LDFLAGS)" -o $(OUTPUT_DIR)/cockpit-windows-amd64.exe ./cmd/cockpit/
	@echo "✅ Cockpit cross-compiled for 5 targets"

# ── Quality ───────────────────────────────────────────────────────────────────

.PHONY: test
test:
	@echo "🧪 Running tests..."
	go test ./... -v -count=1 -timeout 30s

.PHONY: test-cover
test-cover:
	@echo "🧪 Running tests with coverage..."
	go test ./... -cover -coverprofile=coverage.out
	go tool cover -func=coverage.out

.PHONY: vet
vet:
	@echo "🔍 Running go vet..."
	go vet ./...

.PHONY: fmt
fmt:
	@echo "📝 Formatting..."
	go fmt ./...

.PHONY: lint
lint: fmt vet
	@echo "✅ Lint complete"

# ── Clean ─────────────────────────────────────────────────────────────────────

.PHONY: clean
clean:
	@rm -rf $(OUTPUT_DIR) coverage.out
	@echo "🧹 Cleaned"

# ── Install (dev) ─────────────────────────────────────────────────────────────

.PHONY: install
install: build-current build-cockpit
	@mkdir -p $$HOME/.local/bin
	@cp $(OUTPUT_DIR)/$(BINARY_NAME) $$HOME/.local/bin/ovav 2>/dev/null || \
		echo "⚠ Could not install ovav. Copy manually: cp $(OUTPUT_DIR)/$(BINARY_NAME) $$HOME/.local/bin/ovav"
	@cp $(OUTPUT_DIR)/$(COCKPIT_BINARY) $$HOME/.local/bin/ovav-cockpit 2>/dev/null || \
		echo "⚠ Could not install cockpit. Copy manually: cp $(OUTPUT_DIR)/$(COCKPIT_BINARY) $$HOME/.local/bin/ovav-cockpit"
	@echo "✅ Installed ovav + ovav-cockpit to ~/.local/bin/"
	@echo "   Run 'ovav' to launch Cockpit TUI, 'ovav --cli' for flat CLI."

# ── Release (checksums + signing placeholder) ─────────────────────────────────

.PHONY: release
release: clean build
	@echo "📦 Generating checksums..."
	@cd $(OUTPUT_DIR) && sha256sum * > SHA256SUMS
	@echo "✅ Release ready. Binaries in $(OUTPUT_DIR)/"
	@echo ""
	@echo "Binaries:"
	@ls -lh $(OUTPUT_DIR)/$(BINARY_NAME)-*

# ── Help ──────────────────────────────────────────────────────────────────────

.PHONY: help
help:
	@echo "OVAV Go Runtime — Build System"
	@echo ""
	@echo "Targets:"
	@echo "  make build          Cross-compile OVAV for all platforms"
	@echo "  make build-current  Build OVAV for current platform only"
	@echo "  make build-cpanel   Build cPanel for current platform"
	@echo "  make build-cpanel-all   Cross-compile cPanel for 5 targets"
	@echo "  make build-cockpit  Build Cockpit for current platform"
	@echo "  make build-cockpit-all  Cross-compile Cockpit for 5 targets"
	@echo "  make build-tailor   Build Tailor for current platform"
	@echo "  make build-tailor-all   Cross-compile Tailor for 5 targets"
	@echo "  make build-all      Build all 3 binaries"
	@echo "  make test           Run all tests"
	@echo "  make vet            Run go vet"
	@echo "  make fmt            Format code"
	@echo "  make lint           Format + vet"
	@echo "  make clean          Remove build artifacts"
	@echo "  make install        Install binary (requires sudo or ~/.local/bin)"
	@echo "  make release        Build + generate checksums"
	@echo ""
	@echo "Variables:"
	@echo "  VERSION=$(VERSION)"
	@echo "  GOOS=$(GOOS) GOARCH=$(GOARCH)"
