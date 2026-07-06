// SPDX-License-Identifier: GPL-3.0-or-later

package debpkg

import "strings"

// Severity mirrors LSP DiagnosticSeverity so server/ can cast directly.
type Severity int

const (
	SeverityError   Severity = 1
	SeverityWarning Severity = 2
	SeverityInfo    Severity = 3
	SeverityHint    Severity = 4
)

// Diag is a single diagnostic produced by a linter. All coordinates are
// 0-indexed line/column offsets, matching LSP convention.
type Diag struct {
	Line     int
	Col      int
	EndLine  int
	EndCol   int
	Severity Severity
	Message  string
	Code     string // machine-readable code for code-action quick-fixes
	Source   string // human-readable source name (e.g. "dep3", "control")
}

// LintContext carries cross-file information that some linters need.
type LintContext struct {
	// IsUbuntu is true when the package version (from debian/changelog)
	// contains "ubuntu", indicating an Ubuntu-specific upload.
	IsUbuntu bool
}

// Lint runs the linter for the given file type and returns all diagnostics.
// Universal rules (trailing whitespace, whitespace-only blank lines) are
// applied as a pre-pass for every recognised file type.
func Lint(text string, ft FileType, ctx LintContext) []Diag {
	if ft == FileTypeUnknown {
		return nil
	}

	diags := lintUniversal(text)

	switch ft {
	case FileTypeChangelog:
		diags = append(diags, lintChangelog(text)...)
	case FileTypeControl:
		diags = append(diags, lintControl(text, ctx)...)
	case FileTypeCopyright:
		diags = append(diags, lintCopyright(text)...)
	case FileTypeWatch:
		diags = append(diags, lintWatch(text)...)
	case FileTypePatch:
		diags = append(diags, lintPatch(text)...)
	case FileTypeRules:
		diags = append(diags, lintRules(text)...)
	case FileTypeInstall, FileTypeLinks:
		diags = append(diags, lintInstallFile(text, true)...)
	case FileTypeDirs, FileTypeDocs, FileTypeManpages:
		diags = append(diags, lintInstallFile(text, false)...)
	}

	return diags
}

// lintUniversal checks rules that apply to every recognised file:
//   - trailing whitespace on content lines
//   - whitespace-only blank lines
//   - CRLF line endings (Debian files should use LF)
func lintUniversal(text string) []Diag {
	var diags []Diag
	for i, line := range splitLines(text) {
		if line == "" {
			continue
		}
		trimmed := strings.TrimRight(line, " \t\r")
		if trimmed == "" {
			// Whole line is whitespace — blank line with spaces.
			diags = append(diags, Diag{
				Line: i, Col: 0, EndLine: i, EndCol: len(line),
				Severity: SeverityWarning,
				Message:  "blank line contains whitespace",
				Code:     "blank-line-whitespace",
				Source:   "universal",
			})
		} else if len(trimmed) < len(line) {
			// Content line with trailing whitespace (includes \r).
			diags = append(diags, Diag{
				Line: i, Col: len(trimmed), EndLine: i, EndCol: len(line),
				Severity: SeverityWarning,
				Message:  "trailing whitespace",
				Code:     "trailing-whitespace",
				Source:   "universal",
			})
		}
	}

	// Detect CRLF line endings. Debian packaging files should use LF.
	if strings.Contains(text, "\r\n") {
		rawLines := strings.Split(strings.TrimRight(text, "\n"), "\n")
		for i, line := range rawLines {
			if !strings.HasSuffix(line, "\r") {
				continue
			}
			diags = append(diags, Diag{
				Line: i, Col: len(line) - 1, EndLine: i, EndCol: len(line),
				Severity: SeverityWarning,
				Message:  "CRLF line ending; Debian files should use LF",
				Code:     "crlf-line-ending",
				Source:   "universal",
			})
		}
	}

	return diags
}

// splitLines splits text on newlines, stripping a single trailing newline so
// the last element is never an empty phantom line from the final \n. A
// trailing \r (from CRLF endings) is also stripped from each line so that
// regex-based linters see clean content.
func splitLines(text string) []string {
	if text == "" {
		return nil
	}
	text = strings.TrimRight(text, "\n")
	lines := strings.Split(text, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, "\r")
	}
	return lines
}

// isBlank returns true for empty lines and whitespace-only lines.
func isBlank(s string) bool { return strings.TrimSpace(s) == "" }

// countLeadingSpaces returns the number of leading space characters (not tabs)
// in s.
func countLeadingSpaces(s string) int {
	n := 0
	for _, ch := range s {
		if ch != ' ' {
			break
		}
		n++
	}
	return n
}
