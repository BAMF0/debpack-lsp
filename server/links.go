package server

import (
	"regexp"
	"strings"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"github.com/BAMF0/debpack-lsp/debpkg"
)

func (s *Server) documentLink(ctx *glsp.Context, params *protocol.DocumentLinkParams) ([]protocol.DocumentLink, error) {
	uri := params.TextDocument.URI
	text, ok := s.docs.get(uri)
	if !ok {
		return nil, nil
	}
	ft := debpkg.FileTypeFromURI(uri)
	lines := strings.Split(text, "\n")

	var links []protocol.DocumentLink

	// Bug references: LP: #N and Closes: #N (changelog and patch files).
	if ft == debpkg.FileTypeChangelog || ft == debpkg.FileTypePatch {
		links = append(links, bugRefLinks(lines)...)
	}

	// URL-bearing fields in control and patch files.
	if ft == debpkg.FileTypeControl || ft == debpkg.FileTypePatch ||
		ft == debpkg.FileTypeCopyright {
		links = append(links, urlFieldLinks(lines)...)
	}

	return links, nil
}

// bugRefLinkRe matches "LP: #NNN" or "Closes: #NNN".
var bugRefLinkRe = regexp.MustCompile(`(?i)(lp|closes):\s*#(\d+)`)

func bugRefLinks(lines []string) []protocol.DocumentLink {
	var out []protocol.DocumentLink
	for i, line := range lines {
		for _, m := range bugRefLinkRe.FindAllStringSubmatchIndex(line, -1) {
			kind := strings.ToLower(line[m[2]:m[3]])
			num := line[m[4]:m[5]]
			var target string
			switch kind {
			case "lp":
				target = "https://bugs.launchpad.net/bugs/" + num
			case "closes":
				target = "https://bugs.debian.org/" + num
			default:
				continue
			}
			out = append(out, protocol.DocumentLink{
				Range: protocol.Range{
					Start: protocol.Position{Line: uint32(i), Character: uint32(m[0])},
					End:   protocol.Position{Line: uint32(i), Character: uint32(m[1])},
				},
				Target:  &target,
				Tooltip: &target,
			})
		}
	}
	return out
}

// urlFieldRe matches field lines whose value looks like an http(s) URL.
var urlFieldRe = regexp.MustCompile(`(?i)^(homepage|vcs-(browser|git|hg|bzr|svn)|bug|bug-debian|bug-ubuntu|applied-upstream|origin|forwarded)\s*:\s*(https?://\S+)`)

func urlFieldLinks(lines []string) []protocol.DocumentLink {
	var out []protocol.DocumentLink
	for i, line := range lines {
		m := urlFieldRe.FindStringSubmatchIndex(line)
		if m == nil {
			continue
		}
		urlStart, urlEnd := m[6], m[7]
		target := line[urlStart:urlEnd]
		out = append(out, protocol.DocumentLink{
			Range: protocol.Range{
				Start: protocol.Position{Line: uint32(i), Character: uint32(urlStart)},
				End:   protocol.Position{Line: uint32(i), Character: uint32(urlEnd)},
			},
			Target:  &target,
			Tooltip: &target,
		})
	}
	return out
}
