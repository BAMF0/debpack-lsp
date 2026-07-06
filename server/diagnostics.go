// SPDX-License-Identifier: GPL-3.0-or-later

package server

import (
	"github.com/BAMF0/debpack-lsp/debpkg"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// publishDiagnostics lints the document at uri and pushes the results to the
// client via a textDocument/publishDiagnostics notification.
func (s *Server) publishDiagnostics(ctx *glsp.Context, uri, text string) {
	ft := debpkg.FileTypeFromURI(uri)

	s.pkgMu.RLock()
	lctx := debpkg.LintContext{IsUbuntu: s.isUbuntu}
	s.pkgMu.RUnlock()

	raw := debpkg.Lint(text, ft, lctx)

	// Cross-file checks (debhelper-compat vs debian/compat, quilt series, …).
	raw = append(raw, s.crossFileDiagnostics(ft, text)...)

	defaultSource := "debpack-lsp"
	lspDiags := make([]protocol.Diagnostic, 0, len(raw))
	for _, d := range raw {
		d := d
		sev := protocol.DiagnosticSeverity(d.Severity)
		src := d.Source
		if src == "" {
			src = defaultSource
		}
		diag := protocol.Diagnostic{
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
			Source:   &src,
			Message:  d.Message,
		}
		if d.Code != "" {
			code := protocol.IntegerOrString{Value: d.Code}
			diag.Code = &code
		}
		lspDiags = append(lspDiags, diag)
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
