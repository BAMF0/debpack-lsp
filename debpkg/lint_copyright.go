package debpkg

import (
	"fmt"
	"strings"
)

// ---------------------------------------------------------------------------
// Canonical DEP-5 Format URIs
// ---------------------------------------------------------------------------

const (
	dep5CanonicalURI = "https://www.debian.org/doc/packaging-manuals/copyright-format/1.0/"
	dep5LegacyURI    = "http://www.debian.org/doc/packaging-manuals/copyright-format/1.0/"
)

// ---------------------------------------------------------------------------
// Entry point
// ---------------------------------------------------------------------------

func lintCopyright(text string) []Diag {
	lines := splitLines(text)
	stanzas := parseCopyrightStanzas(lines)

	if len(stanzas) == 0 {
		return nil
	}

	var diags []Diag
	hasFormatField := false
	hasCatchAll := false

	for i, s := range stanzas {
		if i == 0 {
			// Header stanza: must have Format field.
			if fmtVal, ok := s.fields["format"]; ok {
				hasFormatField = true
				if fmtVal != dep5CanonicalURI && fmtVal != dep5LegacyURI {
					ln := s.fieldLine["format"]
					diags = append(diags, Diag{
						Line: ln, Col: 0, EndLine: ln, EndCol: 0,
						Severity: SeverityWarning,
						Message: fmt.Sprintf(
							"Format should be the canonical DEP-5 URI %q", dep5CanonicalURI,
						),
					})
				}
			} else {
				diags = append(diags, Diag{
					Line: s.start, Col: 0, EndLine: s.start, EndCol: 0,
					Severity: SeverityError,
					Message:  "copyright header stanza is missing the mandatory \"Format:\" field",
				})
			}
			continue
		}

		// Files stanza: must have Copyright and License.
		if filesVal, ok := s.fields["files"]; ok {
			if strings.TrimSpace(filesVal) == "*" {
				hasCatchAll = true
			}
			if _, ok := s.fields["copyright"]; !ok {
				diags = append(diags, Diag{
					Line: s.start, Col: 0, EndLine: s.start, EndCol: 0,
					Severity: SeverityError,
					Message:  "Files stanza is missing the mandatory \"Copyright:\" field",
				})
			}
			if _, ok := s.fields["license"]; !ok {
				diags = append(diags, Diag{
					Line: s.start, Col: 0, EndLine: s.start, EndCol: 0,
					Severity: SeverityError,
					Message:  "Files stanza is missing the mandatory \"License:\" field",
				})
			}
		}
	}

	if hasFormatField && !hasCatchAll {
		diags = append(diags, Diag{
			Line: 0, Col: 0, EndLine: 0, EndCol: 0,
			Severity: SeverityWarning,
			Message:  "no catch-all \"Files: *\" stanza found in copyright file",
		})
	}

	return diags
}

// ---------------------------------------------------------------------------
// Stanza parser (reuses the same pattern as control)
// ---------------------------------------------------------------------------

type copyrightStanza struct {
	fields    map[string]string
	fieldLine map[string]int
	start     int
	end       int
}

func parseCopyrightStanzas(lines []string) []copyrightStanza {
	var stanzas []copyrightStanza
	var cur *copyrightStanza

	flush := func(end int) {
		if cur != nil {
			cur.end = end
			stanzas = append(stanzas, *cur)
			cur = nil
		}
	}

	for i, line := range lines {
		if isBlank(line) {
			flush(i)
			continue
		}
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			continue
		}
		name, val, found := strings.Cut(line, ":")
		if !found {
			continue
		}
		if cur == nil {
			cur = &copyrightStanza{
				fields:    make(map[string]string),
				fieldLine: make(map[string]int),
				start:     i,
			}
		}
		lower := strings.ToLower(strings.TrimSpace(name))
		cur.fields[lower] = strings.TrimSpace(val)
		cur.fieldLine[lower] = i
	}
	flush(len(lines))
	return stanzas
}
