package server

import (
	"strings"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"github.com/BAMF0/debpack-lsp/debpkg"
)

func (s *Server) documentSymbol(ctx *glsp.Context, params *protocol.DocumentSymbolParams) (any, error) {
	uri := params.TextDocument.URI
	text, ok := s.docs.get(uri)
	if !ok {
		return nil, nil
	}
	ft := debpkg.FileTypeFromURI(uri)
	lines := strings.Split(text, "\n")

	switch ft {
	case debpkg.FileTypeControl:
		return controlSymbols(lines), nil
	case debpkg.FileTypeCopyright:
		return copyrightSymbols(lines), nil
	case debpkg.FileTypeChangelog:
		return changelogSymbols(lines), nil
	case debpkg.FileTypePatch:
		return patchSymbols(lines), nil
	}
	return nil, nil
}

// stanzaBounds returns the start and end (inclusive) line indices of each
// blank-line-separated stanza.
func stanzaBounds(lines []string) [][2]int {
	var out [][2]int
	start := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			if start >= 0 {
				out = append(out, [2]int{start, i - 1})
			}
			start = -1
			continue
		}
		if start < 0 {
			start = i
		}
	}
	if start >= 0 {
		out = append(out, [2]int{start, len(lines) - 1})
	}
	return out
}

func controlSymbols(lines []string) []protocol.DocumentSymbol {
	var out []protocol.DocumentSymbol
	for _, b := range stanzaBounds(lines) {
		firstLine := strings.TrimSpace(lines[b[0]])
		name, _, _ := strings.Cut(firstLine, ":")
		name = strings.TrimSpace(name)
		val := strings.TrimSpace(firstLine)
		if idx := strings.Index(val, ":"); idx >= 0 {
			val = strings.TrimSpace(val[idx+1:])
		}
		var detail *string
		if name == "Source" || name == "Package" {
			d := "Package: " + val
			detail = &d
		}
		kind := protocol.SymbolKindModule
		out = append(out, protocol.DocumentSymbol{
			Name:          name + ": " + val,
			Detail:        detail,
			Kind:          kind,
			Range:         stanzaRange(b, lines),
			SelectionRange: stanzaRange(b, lines),
		})
	}
	return out
}

func copyrightSymbols(lines []string) []protocol.DocumentSymbol {
	var out []protocol.DocumentSymbol
	for _, b := range stanzaBounds(lines) {
		firstLine := strings.TrimSpace(lines[b[0]])
		name, val, _ := strings.Cut(firstLine, ":")
		name = strings.TrimSpace(name)
		val = strings.TrimSpace(val)
		kind := protocol.SymbolKindFile
		out = append(out, protocol.DocumentSymbol{
			Name:           name + ": " + val,
			Kind:           kind,
			Range:          stanzaRange(b, lines),
			SelectionRange: stanzaRange(b, lines),
		})
	}
	return out
}

func changelogSymbols(lines []string) []protocol.DocumentSymbol {
	var out []protocol.DocumentSymbol
	start := -1
	for i, line := range lines {
		if isChangelogHeader(line) {
			if start >= 0 {
				out = append(out, changelogSymbol([2]int{start, i - 1}, lines))
			}
			start = i
		}
	}
	if start >= 0 {
		out = append(out, changelogSymbol([2]int{start, len(lines) - 1}, lines))
	}
	return out
}

func changelogSymbol(b [2]int, lines []string) protocol.DocumentSymbol {
	headerLine := lines[b[0]]
	trimmed := strings.TrimSpace(headerLine)
	detail := trimmed
	endCol := 0
	if b[1] < len(lines) {
		endCol = len(lines[b[1]])
	}
	kind := protocol.SymbolKindEvent
	return protocol.DocumentSymbol{
		Name:  trimmed,
		Detail: &detail,
		Kind:  kind,
		Range: protocol.Range{
			Start: protocol.Position{Line: uint32(b[0]), Character: 0},
			End:   protocol.Position{Line: uint32(b[1]), Character: uint32(endCol)},
		},
		SelectionRange: protocol.Range{
			Start: protocol.Position{Line: uint32(b[0]), Character: 0},
			End:   protocol.Position{Line: uint32(b[0]), Character: uint32(len(trimmed))},
		},
	}
}

func patchSymbols(lines []string) []protocol.DocumentSymbol {
	var out []protocol.DocumentSymbol
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "---") || strings.HasPrefix(trimmed, "diff ") {
			break
		}
		name, val, found := strings.Cut(trimmed, ":")
		if !found {
			continue
		}
		name = strings.TrimSpace(name)
		val = strings.TrimSpace(val)
		kind := protocol.SymbolKindField
		out = append(out, protocol.DocumentSymbol{
			Name:            name + ": " + val,
			Kind:            kind,
			Range:           protocol.Range{
				Start: protocol.Position{Line: uint32(i), Character: 0},
				End:   protocol.Position{Line: uint32(i), Character: uint32(len(line))},
			},
			SelectionRange: protocol.Range{
				Start: protocol.Position{Line: uint32(i), Character: 0},
				End:   protocol.Position{Line: uint32(i), Character: uint32(len(name) + 1)},
			},
		})
	}
	return out
}

func stanzaRange(b [2]int, lines []string) protocol.Range {
	endCol := 0
	if b[1] < len(lines) {
		endCol = len(lines[b[1]])
	}
	return protocol.Range{
		Start: protocol.Position{Line: uint32(b[0]), Character: 0},
		End:   protocol.Position{Line: uint32(b[1]), Character: uint32(endCol)},
	}
}
