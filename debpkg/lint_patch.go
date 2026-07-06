// SPDX-License-Identifier: GPL-3.0-or-later

package debpkg

import (
	"fmt"
	"regexp"
	"strings"
)

// ---------------------------------------------------------------------------
// Known DEP-3 field names (used to detect the header block)
// ---------------------------------------------------------------------------

var dep3KnownFields = map[string]bool{
	"description": true, "subject": true,
	"origin": true, "bug": true,
	"forwarded": true, "author": true, "from": true,
	"reviewed-by": true, "acked-by": true,
	"last-update": true, "applied-upstream": true,
}

// dep3BugFieldRe matches Bug or Bug-<Vendor> field names.
var dep3BugFieldRe = regexp.MustCompile(`^bug(-\w+)?$`)

// validOriginKeywords is the closed set allowed as the optional "keyword, "
// prefix on the Origin field.
var validOriginKeywords = map[string]bool{
	"upstream": true, "backport": true, "vendor": true, "other": true,
}

// lastUpdateRe validates YYYY-MM-DD format.
var lastUpdateRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// originKeywordRe extracts an optional keyword prefix from an Origin value.
// e.g. "upstream, https://..." → keyword="upstream"
var originKeywordRe = regexp.MustCompile(`^([a-zA-Z]+),\s`)

// ---------------------------------------------------------------------------
// Parsed DEP-3 header field
// ---------------------------------------------------------------------------

type dep3Field struct {
	name      string
	value     string
	lineStart int      // line of the "Name: value" declaration
	bodyLines []string // continuation lines of the value (for Description)
	bodyStart []int    // corresponding line numbers for bodyLines
}

// ---------------------------------------------------------------------------
// Entry point
// ---------------------------------------------------------------------------

func lintPatch(text string) []Diag {
	fields, diffStart := parseDep3Headers(splitLines(text))

	_ = diffStart // used during parsing, not needed further

	// Index fields for quick lookup.
	byName := make(map[string]*dep3Field)
	for i := range fields {
		byName[fields[i].name] = &fields[i]
	}

	var diags []Diag

	// --- Description / Subject (required) ---
	_, hasDesc := byName["description"]
	_, hasSubj := byName["subject"]
	if !hasDesc && !hasSubj {
		diags = append(diags, Diag{
			Line: 0, Col: 0, EndLine: 0, EndCol: 0,
			Severity: SeverityError,
			Message:  "DEP-3 patch header is missing the required \"Description:\" (or \"Subject:\") field",
			Code:     "dep3-missing-description",
			Source:   "dep3",
		})
	}

	// --- Description line length (recommended < 80) ---
	if desc, ok := byName["description"]; ok {
		for j, bline := range desc.bodyLines {
			if len(bline) > 80 {
				ln := desc.bodyStart[j]
				diags = append(diags, Diag{
					Line: ln, Col: 80, EndLine: ln, EndCol: len(bline),
					Severity: SeverityWarning,
					Message:  fmt.Sprintf("description line is %d characters; recommended maximum is 80", len(bline)),
				})
			}
		}
		// Also check the short description on the declaration line itself.
		shortDesc := desc.value
		if len(shortDesc) > 80 {
			diags = append(diags, Diag{
				Line: desc.lineStart, Col: 80, EndLine: desc.lineStart, EndCol: len(shortDesc) + len("Description: "),
				Severity: SeverityWarning,
				Message:  fmt.Sprintf("description line is %d characters; recommended maximum is 80", len(shortDesc)),
			})
		}
	}

	// --- Origin (required unless Author / From present) ---
	_, hasOrigin := byName["origin"]
	_, hasAuthor := byName["author"]
	_, hasFrom := byName["from"]
	if !hasOrigin && !hasAuthor && !hasFrom {
		diags = append(diags, Diag{
			Line: 0, Col: 0, EndLine: 0, EndCol: 0,
			Severity: SeverityWarning,
			Message:  "DEP-3 patch header is missing \"Origin:\" (required when no \"Author:\" field is present)",
			Code:     "dep3-missing-origin",
			Source:   "dep3",
		})
	}

	// --- Origin keyword prefix validation ---
	if origin, ok := byName["origin"]; ok {
		if m := originKeywordRe.FindStringSubmatch(origin.value); m != nil {
			kw := strings.ToLower(m[1])
			if !validOriginKeywords[kw] {
				diags = append(diags, Diag{
					Line: origin.lineStart, Col: 0, EndLine: origin.lineStart, EndCol: len(origin.value),
					Severity: SeverityWarning,
					Message: fmt.Sprintf(
						"unknown Origin keyword %q; expected one of: upstream, backport, vendor, other",
						m[1],
					),
				})
			}
		}
	}

	// --- Forwarded value ---
	if fwd, ok := byName["forwarded"]; ok {
		val := strings.ToLower(strings.TrimSpace(fwd.value))
		if val != "no" && val != "not-needed" &&
			!strings.HasPrefix(val, "http://") && !strings.HasPrefix(val, "https://") {
			diags = append(diags, Diag{
				Line: fwd.lineStart, Col: 0, EndLine: fwd.lineStart, EndCol: len(fwd.value),
				Severity: SeverityWarning,
				Message:  "Forwarded value should be \"no\", \"not-needed\", or an http(s) URL",
			})
		}
	}

	// --- Last-Update date format ---
	if lu, ok := byName["last-update"]; ok {
		if !lastUpdateRe.MatchString(strings.TrimSpace(lu.value)) {
			diags = append(diags, Diag{
				Line: lu.lineStart, Col: 0, EndLine: lu.lineStart, EndCol: len(lu.value),
				Severity: SeverityWarning,
				Message:  fmt.Sprintf("Last-Update %q is not a valid YYYY-MM-DD date", lu.value),
			})
		}
	}

	return diags
}

// ---------------------------------------------------------------------------
// DEP-3 header parser
// ---------------------------------------------------------------------------

// parseDep3Headers scans lines before the first diff marker and extracts
// DEP-3 structured fields.  It returns the field list and the index of the
// first diff/--- line (or len(lines) if none).
func parseDep3Headers(lines []string) ([]dep3Field, int) {
	diffStart := len(lines)
	for i, line := range lines {
		if strings.HasPrefix(line, "---") || strings.HasPrefix(line, "diff ") ||
			strings.HasPrefix(line, "index ") {
			diffStart = i
			break
		}
	}

	var fields []dep3Field
	var cur *dep3Field

	for i := 0; i < diffStart; i++ {
		line := lines[i]

		// Continuation line.
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			if cur != nil {
				cur.bodyLines = append(cur.bodyLines, line)
				cur.bodyStart = append(cur.bodyStart, i)
			}
			continue
		}

		// Blank line resets current field (but we keep collecting fields).
		if isBlank(line) {
			cur = nil
			continue
		}

		// Field line: "Name: value"
		name, val, found := strings.Cut(line, ":")
		if !found {
			cur = nil
			continue
		}
		lowerName := strings.ToLower(strings.TrimSpace(name))

		// Only track recognised DEP-3 fields (or Bug-* variants).
		if !dep3KnownFields[lowerName] && !dep3BugFieldRe.MatchString(lowerName) {
			cur = nil
			continue
		}

		f := dep3Field{
			name:      lowerName,
			value:     strings.TrimSpace(val),
			lineStart: i,
		}
		fields = append(fields, f)
		cur = &fields[len(fields)-1]
	}

	return fields, diffStart
}
