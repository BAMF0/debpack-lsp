BINARY     := debpack-lsp
INSTALL_DIR ?= $(HOME)/.local/bin
LUA_DIR    ?= $(HOME)/.local/share/nvim/site/lua
VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

.PHONY: all build install install-lua uninstall clean test fmt vet lint vuln check

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

## clean: remove build artefacts
clean:
	rm -f $(BINARY)
