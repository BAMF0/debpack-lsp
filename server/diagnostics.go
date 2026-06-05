package server

import (
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"github.com/yourusername/debpack-lsp/debpkg"
)

// publishDiagnostics lints the document at uri and pushes the results to the
// client via a textDocument/publishDiagnostics notification.
func (s *Server) publishDiagnostics(ctx *glsp.Context, uri, text string) {
	ft := debpkg.FileTypeFromURI(uri)

	s.pkgMu.RLock()
	lctx := debpkg.LintContext{IsUbuntu: s.isUbuntu}
	s.pkgMu.RUnlock()

	raw := debpkg.Lint(text, ft, lctx)

	source := "debpack-lsp"
	lspDiags := make([]protocol.Diagnostic, 0, len(raw))
	for _, d := range raw {
		d := d
		sev := protocol.DiagnosticSeverity(d.Severity)
		lspDiags = append(lspDiags, protocol.Diagnostic{
			Range: protocol.Range{
				Start: protocol.Position{
					Line:      uint32(d.Line),
					Character: uint32(d.Col),
				},
				End: protocol.Position{
					Line:      uint32(d.EndLine),
					Character: uint32(d.EndCol),
				},
			},
			Severity: &sev,
			Source:   &source,
			Message:  d.Message,
		})
	}

	ctx.Notify(
		string(protocol.ServerTextDocumentPublishDiagnostics),
		&protocol.PublishDiagnosticsParams{
			URI:         uri,
			Diagnostics: lspDiags,
		},
	)
}

// clearDiagnostics pushes an empty diagnostics list for uri, erasing any
// squiggles the client is currently showing for that file.
func (s *Server) clearDiagnostics(ctx *glsp.Context, uri string) {
	ctx.Notify(
		string(protocol.ServerTextDocumentPublishDiagnostics),
		&protocol.PublishDiagnosticsParams{
			URI:         uri,
			Diagnostics: []protocol.Diagnostic{},
		},
	)
}
