// SPDX-License-Identifier: GPL-3.0-or-later

package debpkg

import (
	"fmt"
	"strings"
)

// lintInstallFile lints debian/*.install, *.dirs, *.docs, *.links, and
// *.manpages files. These are simple line-based path-list files; the checks
// are structural (field count per line) rather than semantic.
//
// The twoFields parameter selects whether each line must have exactly two
// whitespace-separated fields (*.install and *.links: source + dest) or
// exactly one (*.dirs, *.docs, *.manpages: a single path).
func lintInstallFile(text string, twoFields bool) []Diag {
	lines := splitLines(text)
	var diags []Diag

	hasContent := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		hasContent = true

		fields := strings.Fields(line)
		if twoFields {
			if len(fields) != 2 {
				diags = append(diags, Diag{
					Line: i, Col: 0, EndLine: i, EndCol: len(line),
					Severity: SeverityWarning,
					Message: fmt.Sprintf(
						"expected exactly 2 fields (source dest), got %d",
						len(fields),
					),
					Code:   "install-field-count",
					Source: "install",
				})
			}
		} else {
			if len(fields) != 1 {
				diags = append(diags, Diag{
					Line: i, Col: 0, EndLine: i, EndCol: len(line),
					Severity: SeverityWarning,
					Message: fmt.Sprintf(
						"expected exactly 1 path, got %d fields",
						len(fields),
					),
					Code:   "install-single-path",
					Source: "install",
				})
			}
		}
	}

	if !hasContent && len(lines) > 0 {
		diags = append(diags, Diag{
			Line: 0, Col: 0, EndLine: 0, EndCol: 0,
			Severity: SeverityInfo,
			Message:  "file is empty; debhelper will skip it — is this intended?",
			Code:     "install-empty",
			Source:   "install",
		})
	}

	return diags
}
