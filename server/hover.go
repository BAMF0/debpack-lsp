// SPDX-License-Identifier: GPL-3.0-or-later

package server

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/BAMF0/debpack-lsp/bugs"
	"github.com/BAMF0/debpack-lsp/debhelper"
	"github.com/BAMF0/debpack-lsp/debpkg"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func (s *Server) hover(ctx *glsp.Context, params *protocol.HoverParams) (*protocol.Hover, error) {
	uri := params.TextDocument.URI
	text, ok := s.docs.get(uri)
	if !ok {
		return nil, nil
	}

	ft := debpkg.FileTypeFromURI(uri)
	line := int(params.Position.Line)
	col := int(params.Position.Character)
	fullLine := fullLineAt(text, line)
	offset := lineColToByteOffset(text, line, col)

	switch ft {
	case debpkg.FileTypeChangelog:
		return s.hoverBugRef(text, fullLine, offset)
	case debpkg.FileTypeControl:
		return s.hoverControlField(fullLine)
	case debpkg.FileTypeRules:
		return s.hoverDhCommand(fullLine, col)
	case debpkg.FileTypeCopyright:
		return s.hoverCopyrightField(fullLine)
	case debpkg.FileTypePatch:
		return s.hoverPatchField(fullLine)
	case debpkg.FileTypeWatch:
		return s.hoverWatchOption(fullLine, col)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// changelog hover: LP: #NNNNNN / Closes: #NNNNNN
// ---------------------------------------------------------------------------

func (s *Server) hoverBugRef(text, line string, offset int) (*protocol.Hover, error) {
	// Distinguish Launchpad ("LP: #") from Debian BTS ("Closes: #") refs.
	// The Debian BTS backend is not yet implemented, so "Closes: #N" gets a
	// dedicated message rather than a (wrong) Launchpad lookup.
	if _, idStr := debpkg.ClosesRefAtOffset(text, offset); idStr != "" {
		id, err := strconv.Atoi(idStr)
		if err != nil {
			return nil, nil
		}
		md := fmt.Sprintf(
			"**Closes: #%d** — Debian BTS support is not yet implemented.\n\n"+
				"See https://bugs.debian.org/%d", id, id)
		return markdownHover(md), nil
	}

	idStr := debpkg.BugNumberAtOffset(text, offset)
	if idStr == "" {
		return nil, nil
	}
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return nil, nil
	}
	bug := s.bugs.ByID(id)
	if bug == nil {
		md := fmt.Sprintf("**LP: #%d** — not found in local cache.\n\nRun `lpad sync` to refresh.", id)
		return markdownHover(md), nil
	}
	return markdownHover(bugMarkdown(*bug)), nil
}

// ---------------------------------------------------------------------------
// control hover: field names
// ---------------------------------------------------------------------------

func (s *Server) hoverControlField(line string) (*protocol.Hover, error) {
	name := debpkg.FieldNameFromLine(line)
	if name == "" {
		return nil, nil
	}
	f := debpkg.LookupField(name)
	if f == nil {
		return nil, nil
	}
	md := fmt.Sprintf("**%s**\n\n%s", f.Name, f.Description)
	if len(f.Values) > 0 {
		md += "\n\n**Known values:** " + strings.Join(f.Values, ", ")
	}
	return markdownHover(md), nil
}

// ---------------------------------------------------------------------------
// rules hover: dh_* commands
// ---------------------------------------------------------------------------

func (s *Server) hoverDhCommand(line string, col int) (*protocol.Hover, error) {
	word := wordAtCol(line, col)
	if !strings.HasPrefix(word, "dh_") {
		return nil, nil
	}
	cmd := s.dh.ByName(word)
	if cmd == nil {
		return nil, nil
	}
	return markdownHover(dhMarkdown(*cmd)), nil
}

// ---------------------------------------------------------------------------
// copyright hover: DEP-5 fields
// ---------------------------------------------------------------------------

func (s *Server) hoverCopyrightField(line string) (*protocol.Hover, error) {
	name := debpkg.FieldNameFromLine(line)
	if name == "" {
		return nil, nil
	}
	f := debpkg.LookupCopyrightField(name)
	if f == nil {
		return nil, nil
	}
	md := fmt.Sprintf("**%s** (DEP-5)\n\n%s", f.Name, f.Description)
	if len(f.Values) > 0 {
		md += "\n\n**Known values:** " + strings.Join(f.Values, ", ")
	}
	return markdownHover(md), nil
}

// ---------------------------------------------------------------------------
// patch hover: DEP-3 fields
// ---------------------------------------------------------------------------

func (s *Server) hoverPatchField(line string) (*protocol.Hover, error) {
	name := debpkg.FieldNameFromLine(line)
	if name == "" {
		return nil, nil
	}
	f := debpkg.LookupPatchField(name)
	if f == nil {
		return nil, nil
	}
	md := fmt.Sprintf("**%s** (DEP-3)\n\n%s", f.Name, f.Description)
	if len(f.Values) > 0 {
		md += "\n\n**Known values:** " + strings.Join(f.Values, ", ")
	}
	return markdownHover(md), nil
}

// ---------------------------------------------------------------------------
// watch hover: debian/watch options
// ---------------------------------------------------------------------------

func (s *Server) hoverWatchOption(line string, col int) (*protocol.Hover, error) {
	word := wordAtCol(line, col)
	if word == "" {
		return nil, nil
	}
	lower := strings.ToLower(word)

	// "version" keyword
	if lower == "version" {
		return markdownHover("**version**\n\nWatch file version declaration. Valid values are `4` and `5`."), nil
	}

	// Check KnownWatchOptions
	for _, opt := range debpkg.KnownWatchOptions {
		if strings.ToLower(opt.Name) == lower {
			return markdownHover(fmt.Sprintf("**%s**\n\n%s", opt.Name, opt.Description)), nil
		}
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// Markdown formatters
// ---------------------------------------------------------------------------

func bugMarkdown(bug bugs.Bug) string {
	return fmt.Sprintf(
		"**LP: #%d** — %s\n\n%s · %s",
		bug.ID, bug.Title, bug.Status, bug.Importance,
	)
}

func dhMarkdown(cmd debhelper.Command) string {
	md := fmt.Sprintf("**%s**\n\n%s", cmd.Name, cmd.Synopsis)
	if cmd.Description != "" {
		md += "\n\n" + cmd.Description
	}
	if len(cmd.Flags) > 0 {
		md += "\n\n**Flags:**\n"
		for _, f := range cmd.Flags {
			md += fmt.Sprintf("- `%s`\n", f)
		}
	}
	return md
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func markdownHover(md string) *protocol.Hover {
	return &protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: md,
		},
	}
}

// lineColToByteOffset converts a line/col (0-indexed) to a byte offset in text.
// col is a character (rune) count as reported by the LSP client; it is
// converted to a byte offset within the target line.
func lineColToByteOffset(text string, line, col int) int {
	// Advance to the start of the target line by counting newlines.
	lineStart := 0
	cur := 0
	for i := 0; i < len(text); i++ {
		if cur == line {
			break
		}
		if text[i] == '\n' {
			cur++
			lineStart = i + 1
		}
	}
	if cur < line {
		return len(text) // line beyond EOF
	}
	// Find the end of the target line.
	lineEnd := len(text)
	for i := lineStart; i < len(text); i++ {
		if text[i] == '\n' {
			lineEnd = i
			break
		}
	}
	return lineStart + colToByteOffset(text[lineStart:lineEnd], col)
}

// wordAtCol extracts the identifier-like word (letters, digits, _, -) under col.
// col is a character (rune) count; it is converted to a byte offset before
// scanning word boundaries.
func wordAtCol(line string, col int) string {
	byteCol := colToByteOffset(line, col)
	if byteCol > len(line) {
		byteCol = len(line)
	}
	isWord := func(b byte) bool {
		return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') ||
			(b >= '0' && b <= '9') || b == '_' || b == '-'
	}
	start := byteCol
	for start > 0 && isWord(line[start-1]) {
		start--
	}
	end := byteCol
	for end < len(line) && isWord(line[end]) {
		end++
	}
	return line[start:end]
}
