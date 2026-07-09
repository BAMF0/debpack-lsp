// SPDX-License-Identifier: GPL-3.0-or-later

package debpkg

import (
	"strings"
)

// lintRules lints debian/rules for common mistakes:
//   - missing or incorrect shebang
//   - missing the required `%:` catch-all target
//   - deprecated constructs (hardcoded dh_* sequences instead of `dh $@`)
func lintRules(text string) []Diag {
	lines := splitLines(text)
	var diags []Diag

	// Shebang check (line 0). Allow optional space after "#!".
	if len(lines) > 0 {
		first := strings.TrimSpace(lines[0])
		if !isRulesShebang(first) {
			diags = append(diags, Diag{
				Line: 0, Col: 0, EndLine: 0, EndCol: len(lines[0]),
				Severity: SeverityWarning,
				Message:  `debian/rules should start with "#!/usr/bin/make -f"`,
				Code:     "rules-shebang",
				Source:   "rules",
			})
		}
	}

	// Required `%:` catch-all target.
	hasPercentTarget := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "%:") {
			hasPercentTarget = true
			break
		}
	}
	if !hasPercentTarget {
		diags = append(diags, Diag{
			Line: 0, Col: 0, EndLine: 0, EndCol: 0,
			Severity: SeverityWarning,
			Message:  `debian/rules is missing the required "%:" catch-all target`,
			Code:     "rules-missing-target",
			Source:   "rules",
		})
	}

	// Deprecated: hardcoded dh_* calls without a `dh $@` or override pattern.
	// We only flag lines that call dh_* commands directly (not inside
	// override_ targets) as a mild hint.
	dhSequenceCount := 0
	inOverride := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "override_") {
			inOverride = true
			continue
		}
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		// Reset override context on new target lines.
		if strings.HasSuffix(trimmed, ":") && !strings.HasPrefix(trimmed, "override_") {
			inOverride = false
		}
		if inOverride {
			continue
		}
		if strings.Contains(trimmed, "dh_") && !strings.Contains(trimmed, "dh $@") {
			dhSequenceCount++
		}
	}
	if dhSequenceCount > 0 && !hasDhDollar(text) {
		diags = append(diags, Diag{
			Line: 0, Col: 0, EndLine: 0, EndCol: 0,
			Severity: SeverityInfo,
			Message:  "consider using 'dh $@' instead of calling dh_* commands directly",
			Code:     "rules-use-dh-dollar",
			Source:   "rules",
		})
	}

	return diags
}

// isRulesShebang reports whether a line is a valid debian/rules shebang.
// Allows an optional space after "#!" (e.g. "#! /usr/bin/make -f").
func isRulesShebang(line string) bool {
	for _, s := range []string{
		"#!/usr/bin/make -f",
		"#! /usr/bin/make -f",
		"#!/usr/bin/make -sf",
		"#! /usr/bin/make -sf",
	} {
		if line == s {
			return true
		}
	}
	return false
}

// hasDhDollar reports whether the rules file uses 'dh $@' anywhere.
func hasDhDollar(text string) bool {
	for _, line := range splitLines(text) {
		if strings.Contains(line, "dh $@") {
			return true
		}
	}
	return false
}

// HasRulesShebang reports whether text starts with the required
// "#!/usr/bin/make -f" shebang. Used to decide whether to offer the full
// rules template snippet in completions.
func HasRulesShebang(text string) bool {
	lines := splitLines(text)
	if len(lines) == 0 {
		return false
	}
	return isRulesShebang(strings.TrimSpace(lines[0]))
}
