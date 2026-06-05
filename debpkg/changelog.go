package debpkg

import (
	"regexp"
	"strings"
)

// firstLineRe matches: "packagename (version) suite; urgency=..."
var firstLineRe = regexp.MustCompile(`^(\S+)\s+\(`)

// firstVersionRe captures both the package name and the version string.
var firstVersionRe = regexp.MustCompile(`^(\S+)\s+\(([^)]+)\)`)

// PackageFromChangelog extracts the source package name from the first line
// of a debian/changelog file. Returns "" if not found.
func PackageFromChangelog(text string) string {
	line, _, _ := strings.Cut(text, "\n")
	m := firstLineRe.FindStringSubmatch(line)
	if m == nil {
		return ""
	}
	return m[1]
}

// IsUbuntuChangelog reports whether the package version in the first changelog
// entry contains the string "ubuntu" (case-insensitive), indicating an Ubuntu
// upload.  This is used to drive the Ubuntu Maintainer check in control files.
func IsUbuntuChangelog(text string) bool {
	line, _, _ := strings.Cut(text, "\n")
	m := firstVersionRe.FindStringSubmatch(line)
	if m == nil {
		return false
	}
	return strings.Contains(strings.ToLower(m[2]), "ubuntu")
}

// bugRefRe matches "LP: #" or "Closes: #" followed by optional digits.
// Used to detect when the user is typing a bug reference.
var bugRefRe = regexp.MustCompile(`(?i)(lp:\s*#|closes:\s*#)(\d*)$`)

// BugRefAtCursor returns the bug-reference prefix ("LP: #" or "Closes: #")
// and the digits typed so far (may be empty) when the cursor is in a bug
// reference context. Returns ("", "") otherwise.
func BugRefAtCursor(lineUpToCursor string) (prefix, digits string) {
	m := bugRefRe.FindStringSubmatch(lineUpToCursor)
	if m == nil {
		return "", ""
	}
	return m[1], m[2]
}

// bugNumberRe matches a complete bug number such as "LP: #2045432".
var bugNumberRe = regexp.MustCompile(`(?i)(lp:\s*#|closes:\s*#)(\d+)`)

// BugNumberAtOffset returns the bug ID (as a string) if the given byte
// offset within text falls inside a bug reference, or "" otherwise.
func BugNumberAtOffset(text string, offset int) string {
	for _, loc := range bugNumberRe.FindAllStringIndex(text, -1) {
		if offset >= loc[0] && offset <= loc[1] {
			m := bugNumberRe.FindStringSubmatch(text[loc[0]:loc[1]])
			if m != nil {
				return m[2]
			}
		}
	}
	return ""
}
