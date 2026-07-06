package server

import (
	"testing"

	protocol "github.com/tliron/glsp/protocol_3_16"
	"github.com/BAMF0/debpack-lsp/debpkg"
)

func TestStanzaFolds(t *testing.T) {
	lines := []string{
		"Source: foo",
		"Section: utils",
		"",
		"Package: foo",
		"Architecture: any",
		"",
		"Package: foo-doc",
		"Architecture: all",
	}
	folds := stanzaFolds(lines)
	if len(folds) != 3 {
		t.Fatalf("expected 3 folds, got %d", len(folds))
	}
	// First stanza: lines 0-1
	if folds[0].StartLine != 0 || folds[0].EndLine != 1 {
		t.Errorf("fold[0] = (%d, %d), want (0, 1)", folds[0].StartLine, folds[0].EndLine)
	}
	// Second stanza: lines 3-4
	if folds[1].StartLine != 3 || folds[1].EndLine != 4 {
		t.Errorf("fold[1] = (%d, %d), want (3, 4)", folds[1].StartLine, folds[1].EndLine)
	}
	// Third stanza: lines 6-7
	if folds[2].StartLine != 6 || folds[2].EndLine != 7 {
		t.Errorf("fold[2] = (%d, %d), want (6, 7)", folds[2].StartLine, folds[2].EndLine)
	}
}

func TestChangelogFolds(t *testing.T) {
	lines := []string{
		"foo (1.0-1) unstable; urgency=medium",
		"",
		"  * Fix bug",
		"",
		" -- Maint <m@example.com>  Mon, 01 Jan 2024 00:00:00 +0000",
		"",
		"foo (0.9-1) unstable; urgency=low",
		"",
		"  * Initial release",
		"",
		" -- Maint <m@example.com>  Mon, 01 Dec 2023 00:00:00 +0000",
	}
	folds := changelogFolds(lines)
	if len(folds) != 2 {
		t.Fatalf("expected 2 folds, got %d", len(folds))
	}
	if folds[0].StartLine != 0 || folds[0].EndLine != 4 {
		t.Errorf("fold[0] = (%d, %d), want (0, 4)", folds[0].StartLine, folds[0].EndLine)
	}
	if folds[1].StartLine != 6 || folds[1].EndLine != 10 {
		t.Errorf("fold[1] = (%d, %d), want (6, 10)", folds[1].StartLine, folds[1].EndLine)
	}
}

func TestPatchFolds(t *testing.T) {
	lines := []string{
		"Description: fix crash",
		"Origin: upstream, https://example.com",
		"",
		"--- a/foo.c",
		"+++ b/foo.c",
		"@@ -1,3 +1,3 @@",
	}
	folds := patchFolds(lines)
	if len(folds) != 1 {
		t.Fatalf("expected 1 fold, got %d", len(folds))
	}
	if folds[0].StartLine != 0 || folds[0].EndLine != 1 {
		t.Errorf("fold = (%d, %d), want (0, 1)", folds[0].StartLine, folds[0].EndLine)
	}
}

func TestBugRefLinks(t *testing.T) {
	lines := []string{
		"  * Fix crash (LP: #12345)",
		"  * Closes: #67890",
		"  * No bug here",
	}
	links := bugRefLinks(lines)
	if len(links) != 2 {
		t.Fatalf("expected 2 links, got %d", len(links))
	}
	// First link: LP: #12345
	if links[0].Target == nil || *links[0].Target != "https://bugs.launchpad.net/bugs/12345" {
		t.Errorf("link[0] target = %v, want launchpad URL", links[0].Target)
	}
	// Second link: Closes: #67890
	if links[1].Target == nil || *links[1].Target != "https://bugs.debian.org/67890" {
		t.Errorf("link[1] target = %v, want debian BTS URL", links[1].Target)
	}
}

func TestURLFieldLinks(t *testing.T) {
	lines := []string{
		"Homepage: https://example.com",
		"Vcs-Browser: https://salsa.debian.org/foo/foo",
		"Section: utils",
		"Bug-Debian: https://bugs.debian.org/12345",
	}
	links := urlFieldLinks(lines)
	if len(links) != 3 {
		t.Fatalf("expected 3 links, got %d", len(links))
	}
	if links[0].Target == nil || *links[0].Target != "https://example.com" {
		t.Errorf("link[0] target = %v", links[0].Target)
	}
	if links[1].Target == nil || *links[1].Target != "https://salsa.debian.org/foo/foo" {
		t.Errorf("link[1] target = %v", links[1].Target)
	}
}

func TestControlSymbols(t *testing.T) {
	lines := []string{
		"Source: foo",
		"Section: utils",
		"",
		"Package: foo",
		"Architecture: any",
	}
	syms := controlSymbols(lines)
	if len(syms) != 2 {
		t.Fatalf("expected 2 symbols, got %d", len(syms))
	}
	if syms[0].Name != "Source: foo" {
		t.Errorf("sym[0] name = %q, want %q", syms[0].Name, "Source: foo")
	}
	if syms[1].Name != "Package: foo" {
		t.Errorf("sym[1] name = %q, want %q", syms[1].Name, "Package: foo")
	}
}

func TestQuickFixTrailingWhitespace(t *testing.T) {
	lines := []string{"foo   ", "bar"}
	diag := protocol.Diagnostic{
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 3},
			End:   protocol.Position{Line: 0, Character: 6},
		},
	}
	code := protocol.IntegerOrString{Value: "trailing-whitespace"}
	diag.Code = &code

	action, ok := quickFix("file:///test", debpkg.FileTypeControl, lines, 0, "trailing-whitespace", diag)
	if !ok {
		t.Fatal("expected a quick-fix")
	}
	if action.Title != "Remove trailing whitespace" {
		t.Errorf("title = %q", action.Title)
	}
	edits := action.Edit.Changes["file:///test"]
	if len(edits) != 1 {
		t.Fatalf("expected 1 edit, got %d", len(edits))
	}
	if edits[0].NewText != "" {
		t.Errorf("NewText = %q, want empty", edits[0].NewText)
	}
	if edits[0].Range.Start.Character != 3 || edits[0].Range.End.Character != 6 {
		t.Errorf("edit range = (%d, %d), want (3, 6)", edits[0].Range.Start.Character, edits[0].Range.End.Character)
	}
}

func TestQuickFixUnknownCode(t *testing.T) {
	lines := []string{"foo"}
	diag := protocol.Diagnostic{}
	_, ok := quickFix("file:///test", debpkg.FileTypeControl, lines, 0, "unknown-code", diag)
	if ok {
		t.Error("expected no quick-fix for unknown code")
	}
}
