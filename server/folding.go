package server

import (
	"strings"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"github.com/BAMF0/debpack-lsp/debpkg"
)

func (s *Server) foldingRange(ctx *glsp.Context, params *protocol.FoldingRangeParams) ([]protocol.FoldingRange, error) {
	uri := params.TextDocument.URI
	text, ok := s.docs.get(uri)
	if !ok {
		return nil, nil
	}
	ft := debpkg.FileTypeFromURI(uri)
	lines := strings.Split(text, "\n")

	switch ft {
	case debpkg.FileTypeControl, debpkg.FileTypeCopyright:
		return stanzaFolds(lines), nil
	case debpkg.FileTypeChangelog:
		return changelogFolds(lines), nil
	case debpkg.FileTypePatch:
		return patchFolds(lines), nil
	}
	return nil, nil
}

// stanzaFolds produces one folding range per blank-line-separated stanza.
func stanzaFolds(lines []string) []protocol.FoldingRange {
	var out []protocol.FoldingRange
	start := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			if start >= 0 && i-1 > start {
				out = append(out, protocol.FoldingRange{
					StartLine: uint32(start),
					EndLine:   uint32(i - 1),
				})
			}
			start = -1
			continue
		}
		if start < 0 {
			start = i
		}
	}
	if start >= 0 && len(lines)-1 > start {
		out = append(out, protocol.FoldingRange{
			StartLine: uint32(start),
			EndLine:   uint32(len(lines) - 1),
		})
	}
	return out
}

// changelogFolds produces one folding range per changelog entry. An entry
// starts at a header line matching "package (version) ..." and ends just
// before the next header or EOF.
func changelogFolds(lines []string) []protocol.FoldingRange {
	var out []protocol.FoldingRange
	start := -1
	for i, line := range lines {
		if isChangelogHeader(line) {
			if start >= 0 {
				end := trimTrailingBlanks(lines, start, i-1)
				if end > start {
					out = append(out, protocol.FoldingRange{
						StartLine: uint32(start),
						EndLine:   uint32(end),
					})
				}
			}
			start = i
		}
	}
	if start >= 0 {
		end := trimTrailingBlanks(lines, start, len(lines)-1)
		if end > start {
			out = append(out, protocol.FoldingRange{
				StartLine: uint32(start),
				EndLine:   uint32(end),
			})
		}
	}
	return out
}

// isChangelogHeader matches "packagename (version) suite; urgency=...".
func isChangelogHeader(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}
	idx := strings.Index(trimmed, " (")
	return idx > 0 && strings.Contains(trimmed, ") ")
}

// patchFolds folds the DEP-3 header block (everything before the first diff
// marker line starting with "---" or "diff ").
func patchFolds(lines []string) []protocol.FoldingRange {
	diffStart := -1
	for i, line := range lines {
		if strings.HasPrefix(line, "---") || strings.HasPrefix(line, "diff ") {
			diffStart = i
			break
		}
	}
	if diffStart <= 0 {
		return nil
	}
	end := trimTrailingBlanks(lines, 0, diffStart-1)
	if end <= 0 {
		return nil
	}
	return []protocol.FoldingRange{{
		StartLine: 0,
		EndLine:   uint32(end),
	}}
}

// trimTrailingBlanks returns the index of the last non-blank line in the
// range [start, end], or start-1 if all lines are blank.
func trimTrailingBlanks(lines []string, start, end int) int {
	for end > start && strings.TrimSpace(lines[end]) == "" {
		end--
	}
	return end
}
