package debpkg

import (
	"fmt"
	"regexp"
	"strings"
)

// watchVersionEqRe matches the version 4 syntax: version=4 / version = 4
var watchVersionEqRe = regexp.MustCompile(`(?i)^\s*version\s*=\s*(\d+)\s*$`)

// watchVersionFieldRe matches the version 5 field syntax: Version: 5
var watchVersionFieldRe = regexp.MustCompile(`(?i)^\s*version\s*:\s*(\d+)\s*$`)

// knownWatchVersions lists the version numbers recognised by this linter.
var knownWatchVersions = map[string]bool{"4": true, "5": true}

func lintWatch(text string) []Diag {
	for i, line := range splitLines(text) {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Try v4 syntax first (version=N), then v5 field syntax (Version: N).
		var ver string
		if m := watchVersionEqRe.FindStringSubmatch(line); m != nil {
			ver = m[1]
		} else if m := watchVersionFieldRe.FindStringSubmatch(line); m != nil {
			ver = m[1]
		} else {
			return []Diag{{
				Line: i, Col: 0, EndLine: i, EndCol: len(line),
				Severity: SeverityError,
				Message:  "watch file must start with a version declaration (\"version=4\" or \"Version: 5\")",
				Code:     "watch-missing-version",
				Source:   "watch",
			}}
		}

		if !knownWatchVersions[ver] {
			return []Diag{{
				Line: i, Col: 0, EndLine: i, EndCol: len(line),
				Severity: SeverityWarning,
				Message:  fmt.Sprintf("unrecognised watch file version %s; expected 4 or 5", ver),
			}}
		}
		return nil
	}

	// No non-comment content found.
	return []Diag{{
		Line: 0, Col: 0, EndLine: 0, EndCol: 0,
		Severity: SeverityError,
		Message:  "watch file is missing a version declaration (\"version=4\" or \"Version: 5\")",
		Code:     "watch-missing-version",
		Source:   "watch",
	}}
}
