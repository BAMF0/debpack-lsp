BINARY     := debpack-lsp
INSTALL_DIR ?= $(HOME)/.local/bin
LUA_DIR    ?= $(HOME)/.local/share/nvim/site/lua
VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

.PHONY: all build install install-lua uninstall clean test fmt vet lint vuln check vscode-cross-compile vscode-package vscode-clean

all: build

## build: compile the LSP server binary
build:
	go build -ldflags "-X main.version=$(VERSION)" -o $(BINARY) .

## install: build and copy the binary to INSTALL_DIR (default: ~/.local/bin)
install: build install-lua
	install -Dm755 $(BINARY) $(INSTALL_DIR)/$(BINARY)
	@echo "Installed $(BINARY) to $(INSTALL_DIR)/$(BINARY)"

## install-lua: copy the Neovim companion plugin to LUA_DIR
install-lua:
	install -Dm644 lua/debpack-lsp.lua $(LUA_DIR)/debpack-lsp.lua
	@echo "Installed Neovim plugin to $(LUA_DIR)/debpack-lsp.lua"

## uninstall: remove the binary and Lua plugin
uninstall:
	rm -f $(INSTALL_DIR)/$(BINARY)
	rm -f $(LUA_DIR)/debpack-lsp.lua
	@echo "Removed $(BINARY) and Lua plugin"

## test: run all Go tests with race detection
test:
	go test -race ./...

## fmt: format all Go source files
fmt:
	gofmt -s -w .

## vet: run go vet
vet:
	go vet ./...

## lint: run golangci-lint (if installed)
lint:
	@command -v golangci-lint >/dev/null 2>&1 && golangci-lint run ./... || echo "golangci-lint not installed; skipping"

## vuln: run govulncheck (if installed)
vuln:
	@command -v govulncheck >/dev/null 2>&1 && govulncheck ./... || echo "govulncheck not installed; skipping"

## check: composite target — fmt + vet + test
check: vet test
	@echo "All checks passed"

## vscode-cross-compile: build platform-specific binaries for the VS Code extension
vscode-cross-compile:
	@mkdir -p vscode-ext/bin/linux-x64 vscode-ext/bin/linux-arm64 vscode-ext/bin/darwin-x64 vscode-ext/bin/darwin-arm64
	GOOS=linux  GOARCH=amd64 go build -ldflags "-X main.version=$(VERSION)" -o vscode-ext/bin/linux-x64/$(BINARY) .
	GOOS=linux  GOARCH=arm64 go build -ldflags "-X main.version=$(VERSION)" -o vscode-ext/bin/linux-arm64/$(BINARY) .
	GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.version=$(VERSION)" -o vscode-ext/bin/darwin-x64/$(BINARY) .
	GOOS=darwin GOARCH=arm64 go build -ldflags "-X main.version=$(VERSION)" -o vscode-ext/bin/darwin-arm64/$(BINARY) .
	@echo "Cross-compiled binaries for linux-x64, linux-arm64, darwin-x64, darwin-arm64"

## vscode-package: build the VS Code .vsix extension package
vscode-package: vscode-cross-compile
	cp LICENSE vscode-ext/LICENSE
	cd vscode-ext && npm install && npm run build && npx vsce package --no-dependencies
	@echo "VS Code extension packaged. Install with: code --install-extension vscode-ext/*.vsix"

## vscode-clean: remove VS Code extension build artefacts
vscode-clean:
	rm -rf vscode-ext/bin vscode-ext/node_modules vscode-ext/out vscode-ext/*.vsix

## clean: remove build artefacts
clean:
	rm -f $(BINARY)
	rm -rf vscode-ext/bin vscode-ext/node_modules vscode-ext/out vscode-ext/*.vsix
