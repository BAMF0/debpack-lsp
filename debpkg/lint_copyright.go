// SPDX-License-Identifier: GPL-3.0-or-later

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
					Code:     "dep5-missing-format",
					Source:   "dep5",
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
					Code:     "dep5-missing-copyright",
					Source:   "dep5",
				})
			}
			if _, ok := s.fields["license"]; !ok {
				diags = append(diags, Diag{
					Line: s.start, Col: 0, EndLine: s.start, EndCol: 0,
					Severity: SeverityError,
					Message:  "Files stanza is missing the mandatory \"License:\" field",
					Code:     "dep5-missing-license",
					Source:   "dep5",
				})
			} else {
				// Validate the license short name against known SPDX ids.
				// The value may be a short name followed by full-text on
				// continuation lines; only check the first line.
				licVal := s.fields["license"]
				licLine := s.fieldLine["license"]
				licName, _, _ := strings.Cut(licVal, "\n")
				licName = strings.TrimSpace(licName)
				if licName != "" && !isKnownLicense(licName) {
					diags = append(diags, Diag{
						Line: licLine, Col: 0, EndLine: licLine, EndCol: 0,
						Severity: SeverityInfo,
						Message:  fmt.Sprintf("license %q is not a recognised SPDX identifier or LicenseRef-* name", licName),
						Code:     "dep5-unknown-license",
						Source:   "dep5",
					})
				}
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

// isKnownLicense reports whether name is a recognised SPDX identifier or a
// LicenseRef-* name (used for non-SPDX licenses in DEP-5).
func isKnownLicense(name string) bool {
	if strings.HasPrefix(name, "LicenseRef-") {
		return true
	}
	lower := strings.ToLower(name)
	for _, lic := range knownSPDXLicenses {
		if strings.ToLower(lic) == lower {
			return true
		}
	}
	return false
}
