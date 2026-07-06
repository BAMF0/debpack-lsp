// SPDX-License-Identifier: GPL-3.0-or-later

package server

import (
	"testing"

	"github.com/BAMF0/debpack-lsp/debpkg"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// fakeContext creates a glsp.Context whose Notify captures publishDiagnostics
// notifications into the returned slice.
func fakeContext(diags *[]protocol.Diagnostic) *glsp.Context {
	return &glsp.Context{
		Notify: func(method string, params any) {
			if method == string(protocol.ServerTextDocumentPublishDiagnostics) {
				if p, ok := params.(*protocol.PublishDiagnosticsParams); ok {
					*diags = p.Diagnostics
				}
			}
		},
	}
}

func TestIntegrationControlDiagnostics(t *testing.T) {
	s := New("test")
	uri := protocol.DocumentUri("file:///home/user/pkg/debian/control")
	text := `Source: foo
Maintainer: A <a@b.c>
Standards-Version: 3.9.8

Package: foo
Architecture: any
Description: test package
`

	var diags []protocol.Diagnostic
	ctx := fakeContext(&diags)
	s.didOpen(ctx, &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:  uri,
			Text: text,
		},
	})

	// The outdated Standards-Version should produce a diagnostic.
	found := false
	for _, d := range diags {
		if d.Code != nil {
			if code, ok := d.Code.Value.(string); ok && code == "control-standards-version-outdated" {
				found = true
			}
		}
	}
	if !found {
		t.Error("expected control-standards-version-outdated diagnostic")
	}
}

func TestIntegrationCompletion(t *testing.T) {
	s := New("test")
	uri := protocol.DocumentUri("file:///home/user/pkg/debian/control")
	text := "Source: foo\nMaintainer: A <a@b.c>\n"

	var diags []protocol.Diagnostic
	ctx := fakeContext(&diags)
	s.didOpen(ctx, &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{URI: uri, Text: text},
	})

	// Trigger completion at end of line 0 after "Source: foo" — cursor on
	// line 1 col 0 (blank line position is wrong; instead, type "B" on line 3).
	// Let's test control field-name completion by typing "Bui" on a new line.
	textWithPrefix := text + "Bui\n"
	s.didChange(ctx, &protocol.DidChangeTextDocumentParams{
		TextDocument: protocol.VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: uri},
			Version:                1,
		},
		ContentChanges: []any{
			protocol.TextDocumentContentChangeEventWhole{Text: textWithPrefix},
		},
	})

	result, err := s.completion(ctx, &protocol.CompletionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 2, Character: 3},
		},
	})
	if err != nil {
		t.Fatalf("completion error: %v", err)
	}
	items, ok := result.([]protocol.CompletionItem)
	if !ok {
		t.Fatalf("expected []CompletionItem, got %T", result)
	}
	// "Bui" should match "Build-Depends", "Build-Depends-Indep", "Build-Conflicts", etc.
	found := false
	for _, item := range items {
		if item.Label == "Build-Depends" {
			found = true
		}
	}
	if !found {
		t.Error("expected 'Build-Depends' in completion results for 'Bui' prefix")
	}
}

func TestIntegrationHoverControlField(t *testing.T) {
	s := New("test")
	uri := protocol.DocumentUri("file:///home/user/pkg/debian/control")
	text := "Source: foo\nMaintainer: A <a@b.c>\n"

	var diags []protocol.Diagnostic
	ctx := fakeContext(&diags)
	s.didOpen(ctx, &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{URI: uri, Text: text},
	})

	hover, err := s.hover(ctx, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 3},
		},
	})
	if err != nil {
		t.Fatalf("hover error: %v", err)
	}
	if hover == nil {
		t.Fatal("expected non-nil hover for 'Source' field")
	}
	mc, ok := hover.Contents.(protocol.MarkupContent)
	if !ok {
		t.Fatalf("expected MarkupContent, got %T", hover.Contents)
	}
	if mc.Value == "" {
		t.Error("expected non-empty hover content for 'Source' field")
	}
}

func TestIntegrationFoldingRange(t *testing.T) {
	s := New("test")
	uri := protocol.DocumentUri("file:///home/user/pkg/debian/control")
	text := "Source: foo\nSection: utils\n\nPackage: foo\nArchitecture: any\n"

	var diags []protocol.Diagnostic
	ctx := fakeContext(&diags)
	s.didOpen(ctx, &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{URI: uri, Text: text},
	})

	folds, err := s.foldingRange(ctx, &protocol.FoldingRangeParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
	})
	if err != nil {
		t.Fatalf("foldingRange error: %v", err)
	}
	if len(folds) != 2 {
		t.Errorf("expected 2 folds, got %d", len(folds))
	}
}

func TestIntegrationDocumentSymbol(t *testing.T) {
	s := New("test")
	uri := protocol.DocumentUri("file:///home/user/pkg/debian/control")
	text := "Source: foo\nSection: utils\n\nPackage: foo\nArchitecture: any\n"

	var diags []protocol.Diagnostic
	ctx := fakeContext(&diags)
	s.didOpen(ctx, &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{URI: uri, Text: text},
	})

	result, err := s.documentSymbol(ctx, &protocol.DocumentSymbolParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
	})
	if err != nil {
		t.Fatalf("documentSymbol error: %v", err)
	}
	syms, ok := result.([]protocol.DocumentSymbol)
	if !ok {
		t.Fatalf("expected []DocumentSymbol, got %T", result)
	}
	if len(syms) != 2 {
		t.Errorf("expected 2 symbols, got %d", len(syms))
	}
	if syms[0].Name != "Source: foo" {
		t.Errorf("sym[0].Name = %q, want %q", syms[0].Name, "Source: foo")
	}
}

func TestIntegrationDocumentLink(t *testing.T) {
	s := New("test")
	uri := protocol.DocumentUri("file:///home/user/pkg/debian/control")
	text := "Source: foo\nHomepage: https://example.com\n"

	var diags []protocol.Diagnostic
	ctx := fakeContext(&diags)
	s.didOpen(ctx, &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{URI: uri, Text: text},
	})

	links, err := s.documentLink(ctx, &protocol.DocumentLinkParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
	})
	if err != nil {
		t.Fatalf("documentLink error: %v", err)
	}
	if len(links) != 1 {
		t.Fatalf("expected 1 link, got %d", len(links))
	}
	if links[0].Target == nil || *links[0].Target != "https://example.com" {
		t.Errorf("link target = %v, want https://example.com", links[0].Target)
	}
}

func TestIntegrationCodeAction(t *testing.T) {
	s := New("test")
	uri := protocol.DocumentUri("file:///home/user/pkg/debian/control")
	text := "Source: foo  \nSection: utils\n"

	var diags []protocol.Diagnostic
	ctx := fakeContext(&diags)
	s.didOpen(ctx, &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{URI: uri, Text: text},
	})

	// Find the trailing-whitespace diagnostic.
	var twDiag *protocol.Diagnostic
	for i := range diags {
		if diags[i].Code != nil {
			if code, ok := diags[i].Code.Value.(string); ok && code == "trailing-whitespace" {
				twDiag = &diags[i]
				break
			}
		}
	}
	if twDiag == nil {
		t.Fatal("expected trailing-whitespace diagnostic")
	}

	// Request code actions for the range of that diagnostic.
	actions, err := s.codeAction(ctx, &protocol.CodeActionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
		Range:        twDiag.Range,
		Context: protocol.CodeActionContext{
			Diagnostics: diags,
		},
	})
	if err != nil {
		t.Fatalf("codeAction error: %v", err)
	}
	actionList, ok := actions.([]protocol.CodeAction)
	if !ok {
		t.Fatalf("expected []CodeAction, got %T", actions)
	}
	found := false
	for _, a := range actionList {
		if a.Title == "Remove trailing whitespace" {
			found = true
		}
	}
	if !found {
		t.Error("expected 'Remove trailing whitespace' code action")
	}
}

func TestIntegrationCodeActionOverlap(t *testing.T) {
	s := New("test")
	uri := protocol.DocumentUri("file:///home/user/pkg/debian/control")
	text := "Source: foo   \nSection: utils   \n"

	var diags []protocol.Diagnostic
	ctx := fakeContext(&diags)
	s.didOpen(ctx, &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{URI: uri, Text: text},
	})

	if len(diags) < 2 {
		t.Fatalf("expected 2+ diagnostics, got %d", len(diags))
	}

	// Request code actions for a range that starts on line 0 and ends on
	// line 1 — overlapping both diagnostics even though neither starts
	// on line 1.
	actions, err := s.codeAction(ctx, &protocol.CodeActionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 10},
			End:   protocol.Position{Line: 1, Character: 5},
		},
		Context: protocol.CodeActionContext{
			Diagnostics: diags,
		},
	})
	if err != nil {
		t.Fatalf("codeAction error: %v", err)
	}
	actionList, ok := actions.([]protocol.CodeAction)
	if !ok {
		t.Fatalf("expected []CodeAction, got %T", actions)
	}
	// The requested range overlaps both diagnostics, but dedup by code
	// limits the output to one "trailing-whitespace" quick fix.
	if len(actionList) < 1 {
		t.Errorf("expected at least 1 code action for overlapping range, got %d", len(actionList))
	}
}

func TestIntegrationPatchSnippet(t *testing.T) {
	s := New("test")
	// Simulate a client that supports snippets.
	s.snippetsSupported = true

	uri := protocol.DocumentUri("file:///home/user/pkg/debian/patches/fix.patch")
	text := "" // empty patch file — no DEP-3 header yet

	var diags []protocol.Diagnostic
	ctx := fakeContext(&diags)
	s.didOpen(ctx, &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{URI: uri, Text: text},
	})

	result, err := s.completion(ctx, &protocol.CompletionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 0},
		},
	})
	if err != nil {
		t.Fatalf("completion error: %v", err)
	}
	items, ok := result.([]protocol.CompletionItem)
	if !ok {
		t.Fatalf("expected []CompletionItem, got %T", result)
	}
	found := false
	for _, item := range items {
		if item.Label == "DEP-3 header" {
			found = true
			if item.InsertTextFormat == nil ||
				*item.InsertTextFormat != protocol.InsertTextFormatSnippet {
				t.Error("DEP-3 header snippet should have InsertTextFormat=Snippet")
			}
		}
	}
	if !found {
		t.Error("expected 'DEP-3 header' snippet in completions for empty patch file")
	}
}

func TestIntegrationRulesLint(t *testing.T) {
	s := New("test")
	uri := protocol.DocumentUri("file:///home/user/pkg/debian/rules")
	text := "#!/usr/bin/make -f\n%:\n\tdh $@\n"

	var diags []protocol.Diagnostic
	ctx := fakeContext(&diags)
	s.didOpen(ctx, &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{URI: uri, Text: text},
	})

	// A well-formed rules file should have no rules-specific diagnostics.
	for _, d := range diags {
		if d.Source != nil && *d.Source == "rules" {
			t.Errorf("unexpected rules diagnostic: %s", d.Message)
		}
	}
}

func TestIntegrationCloseClearsDiagnostics(t *testing.T) {
	s := New("test")
	uri := protocol.DocumentUri("file:///home/user/pkg/debian/control")
	text := "Source: foo  \n" // trailing whitespace

	var diags []protocol.Diagnostic
	ctx := fakeContext(&diags)
	s.didOpen(ctx, &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{URI: uri, Text: text},
	})
	if len(diags) == 0 {
		t.Fatal("expected diagnostics after didOpen")
	}

	// didClose should clear diagnostics.
	diags = nil
	s.didClose(ctx, &protocol.DidCloseTextDocumentParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
	})
	if len(diags) != 0 {
		t.Errorf("expected 0 diagnostics after didClose, got %d", len(diags))
	}
}

func TestIntegrationFileTypeUnknown(t *testing.T) {
	s := New("test")
	uri := protocol.DocumentUri("file:///home/user/pkg/debian/patches/series")
	text := "fix-crash.patch\n"

	var diags []protocol.Diagnostic
	ctx := fakeContext(&diags)
	s.didOpen(ctx, &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{URI: uri, Text: text},
	})

	// series file should be FileTypeUnknown — no DEP-3 diagnostics.
	for _, d := range diags {
		if d.Source != nil && *d.Source == "dep3" {
			t.Errorf("unexpected DEP-3 diagnostic for series file: %s", d.Message)
		}
	}
	_ = debpkg.FileTypeUnknown // ensure import is used
}
