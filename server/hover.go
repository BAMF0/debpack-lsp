package server

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"github.com/BAMF0/debpack-lsp/bugs"
	"github.com/BAMF0/debpack-lsp/debpkg"
	"github.com/BAMF0/debpack-lsp/debhelper"
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
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// changelog hover: LP: #NNNNNN
// ---------------------------------------------------------------------------

func (s *Server) hoverBugRef(text, line string, offset int) (*protocol.Hover, error) {
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
func lineColToByteOffset(text string, line, col int) int {
	off := 0
	cur := 0
	for _, ch := range text {
		if cur == line {
			break
		}
		if ch == '\n' {
			cur++
		}
		off++
	}
	return off + col
}

// wordAtCol extracts the identifier-like word (letters, digits, _, -) under col.
func wordAtCol(line string, col int) string {
	if col > len(line) {
		col = len(line)
	}
	isWord := func(b byte) bool {
		return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') ||
			(b >= '0' && b <= '9') || b == '_' || b == '-'
	}
	start := col
	for start > 0 && isWord(line[start-1]) {
		start--
	}
	end := col
	for end < len(line) && isWord(line[end]) {
		end++
	}
	return line[start:end]
}
