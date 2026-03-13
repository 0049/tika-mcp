# ============================================================
#  tika-mcp — Makefile
# ============================================================

# ---------- project identity --------------------------------
BINARY      := tika-mcp
MODULE      := github.com/tika-mcp/server
VERSION     := 1.0.0
DESCRIPTION := Apache Tika MCP Server

# ---------- build inputs ------------------------------------
GO          := go
GOFLAGS     ?=
CGO_ENABLED ?= 0

# Inject version + build metadata at link time
BUILD_TIME  := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
GIT_COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
LDFLAGS     := -s -w \
               -X main.serverVersion=$(VERSION) \
               -X main.buildTime=$(BUILD_TIME) \
               -X main.gitCommit=$(GIT_COMMIT)

# ---------- output paths ------------------------------------
BUILD_DIR   := build
DIST_DIR    := dist
DEB_DIR     := $(BUILD_DIR)/deb

# ---------- cross-compile targets ---------------------------
# make release  →  builds all of these
PLATFORMS   := \
    linux/amd64 \
    linux/arm64 \
    linux/arm/v7 \
    darwin/amd64 \
    darwin/arm64 \
    windows/amd64

# ---------- Debian packaging --------------------------------
DEB_ARCH         ?= amd64# override: make deb DEB_ARCH=arm64
DEB_MAINTAINER   := Your Name <you@example.com>
DEB_HOMEPAGE     := https://github.com/your-org/tika-mcp

# ============================================================
.PHONY: all build test vet lint clean release deb deb-all \
        install uninstall docker docker-push help

# Default target
all: vet test build

# ============================================================
## build: compile for the host OS/arch
build: $(BUILD_DIR)/$(BINARY)

$(BUILD_DIR)/$(BINARY): $(wildcard *.go)
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=$(CGO_ENABLED) $(GO) build $(GOFLAGS) \
	    -ldflags "$(LDFLAGS)" \
	    -o $@ .
	@echo "→ built $@"

# ============================================================
## test: run the full test suite
test:
	$(GO) test -v -race -count=1 ./...

## test-short: fast tests only (skips race detector)
test-short:
	$(GO) test -short ./...

## cover: generate HTML coverage report
cover:
	$(GO) test -coverprofile=$(BUILD_DIR)/coverage.out ./...
	$(GO) tool cover -html=$(BUILD_DIR)/coverage.out -o $(BUILD_DIR)/coverage.html
	@echo "→ coverage report: $(BUILD_DIR)/coverage.html"

# ============================================================
## vet: run go vet
vet:
	$(GO) vet ./...

## lint: run staticcheck (install: go install honnef.co/go/tools/cmd/staticcheck@latest)
lint:
	staticcheck ./...

## fmt: format all Go sources
fmt:
	$(GO) fmt ./...

# ============================================================
## release: cross-compile for all platforms → dist/
release: vet test
	@mkdir -p $(DIST_DIR)
	@$(foreach PLATFORM,$(PLATFORMS), \
	    $(MAKE) _cross_one \
	        GOOS=$(word 1,$(subst /, ,$(PLATFORM))) \
	        GOARCH=$(word 2,$(subst /, ,$(PLATFORM))) \
	        GOARM=$(word 3,$(subst /, ,$(PLATFORM))) \
	    ;)
	@echo "→ release artefacts in $(DIST_DIR)/"

_cross_one:
	$(eval OS_ARCH := $(GOOS)-$(GOARCH)$(if $(filter v%,$(GOARM)),-$(GOARM)))
	$(eval OUT     := $(DIST_DIR)/$(BINARY)-$(VERSION)-$(OS_ARCH)$(if $(filter windows,$(GOOS)),.exe))
	@echo "  compiling $(OUT)"
	@CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) GOARM=$(subst v,,$(GOARM)) \
	    $(GO) build -ldflags "$(LDFLAGS)" -o $(OUT) .

# ============================================================
## deb: build a .deb package for DEB_ARCH (default: amd64)
#
#  Requires: dpkg-deb (apt install dpkg)
#  Usage:
#    make deb                       # amd64 .deb
#    make deb DEB_ARCH=arm64        # arm64 .deb
deb: build
	@which dpkg-deb > /dev/null || (echo "ERROR: dpkg-deb not found — apt install dpkg" && exit 1)
	@echo "→ packaging $(BINARY) $(VERSION) for $(DEB_ARCH)"

	# Wipe and recreate staging tree
	@rm -rf $(DEB_DIR)
	@mkdir -p \
	    $(DEB_DIR)/DEBIAN \
	    $(DEB_DIR)/usr/local/bin \
	    $(DEB_DIR)/etc/systemd/system \
	    $(DEB_DIR)/etc/tika-mcp \
	    $(DEB_DIR)/var/log/tika-mcp \
	    $(DEB_DIR)/usr/share/doc/$(BINARY)

	# Binary
	install -m 755 $(BUILD_DIR)/$(BINARY) $(DEB_DIR)/usr/local/bin/$(BINARY)

	# systemd units
	install -m 644 tika-mcp.service $(DEB_DIR)/etc/systemd/system/tika-mcp.service
	install -m 644 tika.service     $(DEB_DIR)/etc/systemd/system/tika.service

	# Default environment file (marked as conffile — dpkg won't overwrite on upgrade)
	install -m 640 tika-mcp.env $(DEB_DIR)/etc/tika-mcp/tika-mcp.env

	# Docs
	install -m 644 README.md $(DEB_DIR)/usr/share/doc/$(BINARY)/README.md

	# Debian control files (generated from debian/ directory)
	install -m 644 debian/control    $(DEB_DIR)/DEBIAN/control
	install -m 755 debian/postinst   $(DEB_DIR)/DEBIAN/postinst
	install -m 755 debian/prerm      $(DEB_DIR)/DEBIAN/prerm
	install -m 755 debian/postrm     $(DEB_DIR)/DEBIAN/postrm
	install -m 644 debian/conffiles  $(DEB_DIR)/DEBIAN/conffiles

	# Set installed-size (kB) in control — computed after staging tree is fully populated
	$(eval DEB_SIZE := $(shell du -sk $(DEB_DIR) 2>/dev/null | cut -f1))
	sed -i "s/^Installed-Size:.*/Installed-Size: $(DEB_SIZE)/" $(DEB_DIR)/DEBIAN/control

	@mkdir -p $(DIST_DIR)
	dpkg-deb --build --root-owner-group $(DEB_DIR) \
	    $(DIST_DIR)/$(BINARY)_$(VERSION)_$(DEB_ARCH).deb
	@echo "→ $(DIST_DIR)/$(BINARY)_$(VERSION)_$(DEB_ARCH).deb"

## deb-all: build .deb packages for amd64 and arm64
deb-all:
	$(MAKE) _deb_arch DEB_ARCH=amd64 GOOS=linux GOARCH=amd64
	$(MAKE) _deb_arch DEB_ARCH=arm64 GOOS=linux GOARCH=arm64

_deb_arch:
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) \
	    $(GO) build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY) .
	$(MAKE) deb DEB_ARCH=$(DEB_ARCH)

# ============================================================
## install: install binary + units to the local system (needs sudo)
install: build
	install -m 755 $(BUILD_DIR)/$(BINARY) /usr/local/bin/$(BINARY)
	install -m 644 tika-mcp.service /etc/systemd/system/tika-mcp.service
	install -m 644 tika.service     /etc/systemd/system/tika.service
	@[ -f /etc/tika-mcp/tika-mcp.env ] || \
	    (mkdir -p /etc/tika-mcp && \
	     install -m 640 tika-mcp.env /etc/tika-mcp/tika-mcp.env && \
	     echo "→ installed /etc/tika-mcp/tika-mcp.env")
	systemctl daemon-reload
	@echo "→ run: sudo systemctl enable --now tika-mcp"

## uninstall: remove binary + units from the local system (needs sudo)
uninstall:
	systemctl disable --now tika-mcp.service tika.service 2>/dev/null || true
	rm -f /usr/local/bin/$(BINARY)
	rm -f /etc/systemd/system/tika-mcp.service
	rm -f /etc/systemd/system/tika.service
	systemctl daemon-reload
	@echo "→ config at /etc/tika-mcp/ was preserved"

# ============================================================
## docker: build Docker image tagged as tika-mcp:VERSION and tika-mcp:latest
docker:
	docker build \
	    --build-arg VERSION=$(VERSION) \
	    -t $(BINARY):$(VERSION) \
	    -t $(BINARY):latest \
	    .
	@echo "→ docker image $(BINARY):$(VERSION)"

## docker-push: push both tags to REGISTRY (default: docker.io/yourorg)
REGISTRY ?= docker.io/yourorg
docker-push: docker
	docker tag $(BINARY):$(VERSION) $(REGISTRY)/$(BINARY):$(VERSION)
	docker tag $(BINARY):latest     $(REGISTRY)/$(BINARY):latest
	docker push $(REGISTRY)/$(BINARY):$(VERSION)
	docker push $(REGISTRY)/$(BINARY):latest

# ============================================================
## clean: remove all build artefacts
clean:
	rm -rf $(BUILD_DIR) $(DIST_DIR)

# ============================================================
## help: list available targets
help:
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@grep -E '^## ' $(MAKEFILE_LIST) \
	    | sed 's/^## /  /' \
	    | column -t -s ':'
	@echo ""
