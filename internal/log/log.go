// Package log provides a lightweight optional logger for debpack-lsp.
//
// Logging is disabled by default. To enable it, set the DEBPACK_LSP_LOG
// environment variable to a file path before starting Neovim:
//
//	DEBPACK_LSP_LOG=/tmp/debpack-lsp.log nvim
package log

import (
	"fmt"
	"io"
	"os"
)

var out io.Writer = io.Discard

func init() {
	path := os.Getenv("DEBPACK_LSP_LOG")
	if path == "" {
		return
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		// Can't log the error anywhere useful; silently stay discarding.
		return
	}
	out = f
}

// Logf writes a formatted message to the log file, if one is configured.
func Logf(format string, args ...any) {
	fmt.Fprintf(out, format, args...)
}
