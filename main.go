package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/BAMF0/debpack-lsp/server"
)

var version = "dev"

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("debpack-lsp %s\n", version)
		os.Exit(0)
	}

	srv := server.New()
	if err := srv.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "debpack-lsp: %v\n", err)
		os.Exit(1)
	}
}
