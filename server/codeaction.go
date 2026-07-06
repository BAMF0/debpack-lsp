package server

import (
	"strings"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"github.com/BAMF0/debpack-lsp/debpkg"
)

func (s *Server) codeAction(ctx *glsp.Context, params *protocol.CodeActionParams) (any, error) {
	uri := params.TextDocument.URI
	text, ok := s.docs.get(uri)
	if !ok {
		return nil, nil
	}
	ft := debpkg.FileTypeFromURI(uri)
	lines := strings.Split(text, "\n")

	var actions []protocol.CodeAction
	seen := map[string]bool{} // de-dup by code within the requested range

	for _, diag := range params.Context.Diagnostics {
		if diag.Code == nil {
			continue
		}
		code, ok := diag.Code.Value.(string)
		if !ok || seen[code] {
			continue
		}

		// Only offer fixes for diagnostics within the requested range.
		diagLine := int(diag.Range.Start.Line)
		if diagLine < int(params.Range.Start.Line) ||
			diagLine > int(params.Range.End.Line) {
			continue
		}

		action, ok := quickFix(uri, ft, lines, diagLine, code, diag)
		if !ok {
			continue
		}
		seen[code] = true
		actions = append(actions, action)
	}

	return actions, nil
}

// quickFix builds a CodeAction for a given diagnostic code. Returns
// (action, true) if a fix is available, or (zero, false) otherwise.
func quickFix(uri string, ft debpkg.FileType, lines []string, line int, code string, diag protocol.Diagnostic) (protocol.CodeAction, bool) {
	kind := protocol.CodeActionKindQuickFix

	switch code {
	case "trailing-whitespace":
		if line >= len(lines) {
			return protocol.CodeAction{}, false
		}
		original := lines[line]
		trimmed := strings.TrimRight(original, " \t\r")
		if trimmed == original {
			return protocol.CodeAction{}, false
		}
		edit := protocol.TextEdit{
			Range: protocol.Range{
				Start: protocol.Position{Line: uint32(line), Character: uint32(len(trimmed))},
				End:   protocol.Position{Line: uint32(line), Character: uint32(len(original))},
			},
			NewText: "",
		}
		title := "Remove trailing whitespace"
		return protocol.CodeAction{
			Title:       title,
			Kind:        &kind,
			Diagnostics: []protocol.Diagnostic{diag},
			IsPreferred: boolPtr(true),
			Edit: &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{uri: {edit}},
			},
		}, true

	case "blank-line-whitespace":
		if line >= len(lines) {
			return protocol.CodeAction{}, false
		}
		original := lines[line]
		edit := protocol.TextEdit{
			Range: protocol.Range{
				Start: protocol.Position{Line: uint32(line), Character: 0},
				End:   protocol.Position{Line: uint32(line), Character: uint32(len(original))},
			},
			NewText: "",
		}
		title := "Clear whitespace-only line"
		return protocol.CodeAction{
			Title:       title,
			Kind:        &kind,
			Diagnostics: []protocol.Diagnostic{diag},
			IsPreferred: boolPtr(true),
			Edit: &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{uri: {edit}},
			},
		}, true

	case "crlf-line-ending":
		if line >= len(lines) {
			return protocol.CodeAction{}, false
		}
		original := lines[line]
		if !strings.HasSuffix(original, "\r") {
			return protocol.CodeAction{}, false
		}
		edit := protocol.TextEdit{
			Range: protocol.Range{
				Start: protocol.Position{Line: uint32(line), Character: uint32(len(original) - 1)},
				End:   protocol.Position{Line: uint32(line), Character: uint32(len(original))},
			},
			NewText: "",
		}
		title := "Convert CRLF to LF"
		return protocol.CodeAction{
			Title:       title,
			Kind:        &kind,
			Diagnostics: []protocol.Diagnostic{diag},
			Edit: &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{uri: {edit}},
			},
		}, true

	case "dep3-missing-description", "dep3-missing-origin":
		// Insert a full DEP-3 header at the top of the file.
		header := "Description: short description\n Origin: longer explanation\nOrigin: upstream, https://example.com/commit\nForwarded: no\nLast-Update: 2024-01-01\n\n"
		edit := protocol.TextEdit{
			Range: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 0},
			},
			NewText: header,
		}
		title := "Insert DEP-3 patch header"
		return protocol.CodeAction{
			Title:       title,
			Kind:        &kind,
			Diagnostics: []protocol.Diagnostic{diag},
			IsPreferred: boolPtr(true),
			Edit: &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{uri: {edit}},
			},
		}, true

	case "dep5-missing-format":
		formatLine := "Format: https://www.debian.org/doc/packaging-manuals/copyright-format/1.0/\n"
		edit := protocol.TextEdit{
			Range: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 0},
			},
			NewText: formatLine,
		}
		title := "Insert DEP-5 Format field"
		return protocol.CodeAction{
			Title:       title,
			Kind:        &kind,
			Diagnostics: []protocol.Diagnostic{diag},
			IsPreferred: boolPtr(true),
			Edit: &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{uri: {edit}},
			},
		}, true

	case "watch-missing-version":
		versionLine := "version=4\n"
		edit := protocol.TextEdit{
			Range: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 0},
			},
			NewText: versionLine,
		}
		title := "Insert watch file version declaration"
		return protocol.CodeAction{
			Title:       title,
			Kind:        &kind,
			Diagnostics: []protocol.Diagnostic{diag},
			IsPreferred: boolPtr(true),
			Edit: &protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{uri: {edit}},
			},
		}, true
	}

	return protocol.CodeAction{}, false
}

func boolPtr(b bool) *bool { return &b }
