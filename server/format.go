// SPDX-License-Identifier: GPL-3.0-or-later

package server

import (
	"strings"

	"github.com/BAMF0/debpack-lsp/debpkg"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func (s *Server) format(_ *glsp.Context, params *protocol.DocumentFormattingParams) ([]protocol.TextEdit, error) {
	uri := params.TextDocument.URI
	text, ok := s.docs.get(uri)
	if !ok {
		return nil, nil
	}

	var formatted string
	switch debpkg.FileTypeFromURI(uri) {
	case debpkg.FileTypeChangelog:
		formatted = debpkg.FormatChangelog(text)
	default:
		return nil, nil
	}

	if formatted == text {
		return nil, nil
	}

	// Return a single whole-document TextEdit replacing everything from the
	// first character to the last character of the original text.
	origLines := strings.Split(text, "\n")
	lastLine := len(origLines) - 1
	lastChar := len(origLines[lastLine])

	return []protocol.TextEdit{{
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: uint32(lastLine), Character: uint32(lastChar)},
		},
		NewText: formatted,
	}}, nil
}
