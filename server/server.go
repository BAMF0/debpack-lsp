// Package server implements the LSP server for debpack-lsp.
// It wires together the glsp framework with the Debian-specific
// completion, hover and diagnostic providers.
package server

import (
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
	glspserver "github.com/tliron/glsp/server"
	"github.com/BAMF0/debpack-lsp/bugs"
	"github.com/BAMF0/debpack-lsp/debpkg"
	"github.com/BAMF0/debpack-lsp/debhelper"

	"sync"
)

const serverName = "debpack-lsp"

// Server holds all shared state for the lifetime of the LSP session.
type Server struct {
	handler  protocol.Handler
	glsp     *glspserver.Server
	docs     *documentStore
	bugs     *bugs.Store
	dh       *debhelper.Store
	pkg      string // detected source package name
	isUbuntu bool   // true when the changelog version contains "ubuntu"
	pkgMu    sync.RWMutex
}

// New creates and initialises a Server. It loads the debhelper cache and
// the lpad bug cache (if available) eagerly so that the first completion
// request is fast.
func New() *Server {
	s := &Server{
		docs: newDocumentStore(),
		bugs: bugs.NewStore(),
		dh:   debhelper.NewStore(),
	}

	// Pre-load debhelper command cache (may scrape man pages on first run).
	go s.dh.Load()

	s.handler = protocol.Handler{
		Initialize:             s.initialize,
		Initialized:            s.initialized,
		Shutdown:               s.shutdown,
		TextDocumentDidOpen:    s.didOpen,
		TextDocumentDidChange:  s.didChange,
		TextDocumentDidClose:   s.didClose,
		TextDocumentCompletion: s.completion,
		TextDocumentHover:      s.hover,
		TextDocumentFormatting: s.format,
	}

	s.glsp = glspserver.NewServer(&s.handler, serverName, false)
	return s
}

// Run starts the server over stdio (the standard LSP transport).
func (s *Server) Run() error {
	return s.glsp.RunStdio()
}

// ---------------------------------------------------------------------------
// Lifecycle handlers
// ---------------------------------------------------------------------------

func (s *Server) initialize(ctx *glsp.Context, params *protocol.InitializeParams) (any, error) {
	triggerChars := []string{
		"#", // LP: #, Closes: #
		" ", // field values after ": "
		"-", // dh_
		"_", // dh_
	}

	capabilities := s.handler.CreateServerCapabilities()
	capabilities.TextDocumentSync = protocol.TextDocumentSyncKindIncremental
	capabilities.CompletionProvider = &protocol.CompletionOptions{
		TriggerCharacters: triggerChars,
	}
	hover := true
	capabilities.HoverProvider = &hover

	return &protocol.InitializeResult{
		Capabilities: capabilities,
		ServerInfo: &protocol.InitializeResultServerInfo{
			Name:    serverName,
			Version: strPtr("dev"),
		},
	}, nil
}

func (s *Server) initialized(_ *glsp.Context, _ *protocol.InitializedParams) error {
	return nil
}

func (s *Server) shutdown(_ *glsp.Context) error {
	return nil
}

// ---------------------------------------------------------------------------
// Document sync handlers
// ---------------------------------------------------------------------------

func (s *Server) didOpen(ctx *glsp.Context, params *protocol.DidOpenTextDocumentParams) error {
	uri := params.TextDocument.URI
	text := params.TextDocument.Text
	s.docs.open(uri, text)
	s.maybeLoadBugs(uri, text)
	s.publishDiagnostics(ctx, uri, text)
	return nil
}

func (s *Server) didChange(ctx *glsp.Context, params *protocol.DidChangeTextDocumentParams) error {
	uri := params.TextDocument.URI
	s.docs.applyChanges(uri, params.ContentChanges)
	if text, ok := s.docs.get(uri); ok {
		s.publishDiagnostics(ctx, uri, text)
	}
	return nil
}

func (s *Server) didClose(ctx *glsp.Context, params *protocol.DidCloseTextDocumentParams) error {
	s.docs.close(params.TextDocument.URI)
	s.clearDiagnostics(ctx, params.TextDocument.URI)
	return nil
}

// ---------------------------------------------------------------------------
// Bug store loading
// ---------------------------------------------------------------------------

// maybeLoadBugs detects the source package from a changelog file and loads
// the lpad cache for that package. It is called whenever a new document is
// opened so that the first changelog file encountered triggers the load.
func (s *Server) maybeLoadBugs(uri, text string) {
	ft := debpkg.FileTypeFromURI(uri)
	if ft != debpkg.FileTypeChangelog {
		return
	}
	pkg := debpkg.PackageFromChangelog(text)
	if pkg == "" {
		return
	}
	s.pkgMu.Lock()
	s.pkg = pkg
	s.isUbuntu = debpkg.IsUbuntuChangelog(text)
	s.pkgMu.Unlock()
	go s.bugs.Load(pkg)
}

func strPtr(s string) *string { return &s }
