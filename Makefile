BINARY     := debpack-lsp
INSTALL_DIR ?= $(HOME)/.local/bin
VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

.PHONY: all build install uninstall clean test

all: build

## build: compile the LSP server binary
build:
	go build -ldflags "-X main.version=$(VERSION)" -o $(BINARY) .

## install: build and copy the binary to INSTALL_DIR (default: ~/.local/bin)
install: build
	install -Dm755 $(BINARY) $(INSTALL_DIR)/$(BINARY)
	@echo "Installed $(BINARY) to $(INSTALL_DIR)/$(BINARY)"

## uninstall: remove the binary from INSTALL_DIR
uninstall:
	rm -f $(INSTALL_DIR)/$(BINARY)
	@echo "Removed $(INSTALL_DIR)/$(BINARY)"

## test: run all Go tests
test:
	go test ./...

## clean: remove build artefacts
clean:
	rm -f $(BINARY)
