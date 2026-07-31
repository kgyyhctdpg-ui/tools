GO ?= go
GO_BUILD_FLAGS ?= -trimpath
GO_LDFLAGS ?= -s -w
DOCX_CLI_DIST_DIR ?= bin
DOCX_CLI_PLATFORMS ?= darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64 windows/arm64
MARKDOWN_CLI_DIST_DIR ?= bin
MARKDOWN_CLI_PLATFORMS ?= darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64 windows/arm64

.PHONY: help build-docx-cli-all test-docx-cli clean-docx-cli build-markdown-cli-all test-markdown-cli clean-markdown-cli

help:
	@echo "Available targets:"
	@echo "  make build-docx-cli-all"
	@echo "  make test-docx-cli"
	@echo "  make clean-docx-cli"
	@echo "  make build-markdown-cli-all"
	@echo "  make test-markdown-cli"
	@echo "  make clean-markdown-cli"

build-docx-cli-all:
	@mkdir -p $(DOCX_CLI_DIST_DIR)
	@for platform in $(DOCX_CLI_PLATFORMS); do \
		goos=$${platform%/*}; \
		goarch=$${platform#*/}; \
		ext=""; \
		if [ "$$goos" = "windows" ]; then ext=".exe"; fi; \
		output="$(DOCX_CLI_DIST_DIR)/docx-$${goos}-$${goarch}$${ext}"; \
		echo "building $$output"; \
		GOOS=$$goos GOARCH=$$goarch $(GO) build $(GO_BUILD_FLAGS) -ldflags="$(GO_LDFLAGS)" -o "$$output" ./cmd/docx || exit 1; \
	done

test-docx-cli:
	$(GO) test ./cmd/docx

clean-docx-cli:
	rm -f $(DOCX_CLI_DIST_DIR)/docx $(DOCX_CLI_DIST_DIR)/docx-*

build-markdown-cli-all:
	@mkdir -p $(MARKDOWN_CLI_DIST_DIR)
	@for platform in $(MARKDOWN_CLI_PLATFORMS); do \
		goos=$${platform%/*}; \
		goarch=$${platform#*/}; \
		ext=""; \
		if [ "$$goos" = "windows" ]; then ext=".exe"; fi; \
		output="$(MARKDOWN_CLI_DIST_DIR)/markdown-$${goos}-$${goarch}$${ext}"; \
		echo "building $$output"; \
		GOOS=$$goos GOARCH=$$goarch $(GO) build $(GO_BUILD_FLAGS) -ldflags="$(GO_LDFLAGS)" -o "$$output" ./cmd/markdown || exit 1; \
	done

test-markdown-cli:
	$(GO) test ./markdown ./cmd/markdown

clean-markdown-cli:
	rm -f $(MARKDOWN_CLI_DIST_DIR)/markdown $(MARKDOWN_CLI_DIST_DIR)/markdown-*
