// SPDX-License-Identifier: GPL-3.0-or-later

package server

import (
	"testing"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestSplitLinesDocuments(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  []string
	}{
		{"empty", "", []string{""}},
		{"single line no newline", "foo", []string{"foo"}},
		// splitLines always appends a trailing element after the last \n.
		{"single line with newline", "foo\n", []string{"foo\n", ""}},
		{"two lines", "foo\nbar\n", []string{"foo\n", "bar\n", ""}},
		{"two lines no trailing newline", "foo\nbar", []string{"foo\n", "bar"}},
		{"CRLF", "foo\r\nbar\r\n", []string{"foo\r\n", "bar\r\n", ""}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := splitLines(tc.input)
			if len(got) != len(tc.want) {
				t.Fatalf("splitLines(%q) = %v (len %d), want %v (len %d)",
					tc.input, got, len(got), tc.want, len(tc.want))
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("  line[%d] = %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestStripTrailingNewline(t *testing.T) {
	cases := []struct{ in, want string }{
		{"foo\n", "foo"},
		{"foo\r\n", "foo"},
		{"foo", "foo"},
		{"foo\r", "foo"},
		{"", ""},
		{"\n", ""},
	}
	for _, tc := range cases {
		if got := stripTrailingNewline(tc.in); got != tc.want {
			t.Errorf("stripTrailingNewline(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestLineColToOffset(t *testing.T) {
	lines := splitLines("hello\nworld\n")
	cases := []struct {
		line, col, want int
	}{
		{0, 0, 0},   // start of line 0
		{0, 3, 3},   // col 3 of line 0
		{0, 5, 5},   // end of "hello"
		{1, 0, 6},   // start of line 1 (after "hello\n")
		{1, 3, 9},   // col 3 of line 1
		{2, 0, 12},  // past EOF (line 2 doesn't exist)
		{99, 0, 12}, // far beyond
	}
	for _, tc := range cases {
		got := lineColToOffset(lines, tc.line, tc.col)
		if got != tc.want {
			t.Errorf("lineColToOffset(line=%d, col=%d) = %d, want %d", tc.line, tc.col, got, tc.want)
		}
	}
}

func TestLineColToOffsetMultiByte(t *testing.T) {
	// "café\nbar\n" — 'é' is 2 bytes, so line 0 is 5 bytes + \n = 6 bytes.
	lines := splitLines("café\nbar\n")
	// Line 0, col 3 -> byte offset 3 (start of 'é')
	if got := lineColToOffset(lines, 0, 3); got != 3 {
		t.Errorf("lineColToOffset(0,3) = %d, want 3", got)
	}
	// Line 0, col 4 -> byte offset 5 (past 'é')
	if got := lineColToOffset(lines, 0, 4); got != 5 {
		t.Errorf("lineColToOffset(0,4) = %d, want 5", got)
	}
	// Line 1, col 0 -> byte offset 6 (after "café\n")
	if got := lineColToOffset(lines, 1, 0); got != 6 {
		t.Errorf("lineColToOffset(1,0) = %d, want 6", got)
	}
}

func TestApplyRangeChangeSingleLine(t *testing.T) {
	text := "hello world\n"
	r := protocol.Range{
		Start: protocol.Position{Line: 0, Character: 6},
		End:   protocol.Position{Line: 0, Character: 11},
	}
	got := applyRangeChange(text, r, "there")
	want := "hello there\n"
	if got != want {
		t.Errorf("applyRangeChange = %q, want %q", got, want)
	}
}

func TestApplyRangeChangeInsertAtStart(t *testing.T) {
	text := "world\n"
	r := protocol.Range{
		Start: protocol.Position{Line: 0, Character: 0},
		End:   protocol.Position{Line: 0, Character: 0},
	}
	got := applyRangeChange(text, r, "hello ")
	want := "hello world\n"
	if got != want {
		t.Errorf("applyRangeChange = %q, want %q", got, want)
	}
}

func TestApplyRangeChangeMultiLine(t *testing.T) {
	text := "line1\nline2\nline3\n"
	r := protocol.Range{
		Start: protocol.Position{Line: 0, Character: 0},
		End:   protocol.Position{Line: 1, Character: 5},
	}
	got := applyRangeChange(text, r, "replaced")
	want := "replaced\nline3\n"
	if got != want {
		t.Errorf("applyRangeChange = %q, want %q", got, want)
	}
}

func TestApplyRangeChangeMultiByte(t *testing.T) {
	// "café world" — 'é' is 2 bytes. "world" starts at col 5 (byte 6).
	text := "café world\n"
	r := protocol.Range{
		Start: protocol.Position{Line: 0, Character: 5},  // 'w'
		End:   protocol.Position{Line: 0, Character: 10}, // past 'd'
	}
	got := applyRangeChange(text, r, "there")
	want := "café there\n"
	if got != want {
		t.Errorf("applyRangeChange = %q, want %q", got, want)
	}
}

func TestApplyRangeChangeCRLF(t *testing.T) {
	text := "foo\r\nbar\r\n"
	// Replace "bar" on line 1.
	r := protocol.Range{
		Start: protocol.Position{Line: 1, Character: 0},
		End:   protocol.Position{Line: 1, Character: 3},
	}
	got := applyRangeChange(text, r, "baz")
	want := "foo\r\nbaz\r\n"
	if got != want {
		t.Errorf("applyRangeChange = %q, want %q", got, want)
	}
}

func TestDocumentStoreOpenGetClose(t *testing.T) {
	ds := newDocumentStore()
	uri := protocol.DocumentUri("file:///test")

	// Get before open.
	if _, ok := ds.get(uri); ok {
		t.Error("expected ok=false before open")
	}

	ds.open(uri, "hello\n")
	text, ok := ds.get(uri)
	if !ok || text != "hello\n" {
		t.Errorf("after open, get = (%q, %v), want (%q, true)", text, ok, "hello\n")
	}

	ds.close(uri)
	if _, ok := ds.get(uri); ok {
		t.Error("expected ok=false after close")
	}
}

func TestDocumentStoreApplyChangesFull(t *testing.T) {
	ds := newDocumentStore()
	uri := protocol.DocumentUri("file:///test")
	ds.open(uri, "old\n")

	ds.applyChanges(uri, []any{
		protocol.TextDocumentContentChangeEventWhole{Text: "new\n"},
	})
	text, _ := ds.get(uri)
	if text != "new\n" {
		t.Errorf("after full change, text = %q, want %q", text, "new\n")
	}
}

func TestDocumentStoreApplyChangesIncremental(t *testing.T) {
	ds := newDocumentStore()
	uri := protocol.DocumentUri("file:///test")
	ds.open(uri, "hello world\n")

	ds.applyChanges(uri, []any{
		protocol.TextDocumentContentChangeEvent{
			Range: &protocol.Range{
				Start: protocol.Position{Line: 0, Character: 6},
				End:   protocol.Position{Line: 0, Character: 11},
			},
			Text: "there",
		},
	})
	text, _ := ds.get(uri)
	if text != "hello there\n" {
		t.Errorf("after incremental change, text = %q, want %q", text, "hello there\n")
	}
}

func TestDocumentStoreApplyChangesUnknownURI(t *testing.T) {
	ds := newDocumentStore()
	// Should not panic on unknown URI.
	ds.applyChanges("file:///unknown", []any{
		protocol.TextDocumentContentChangeEventWhole{Text: "noop"},
	})
}
