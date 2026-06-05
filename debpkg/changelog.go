package debpkg

import (
	"regexp"
	"strconv"
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

// ContextBeforeBugRef returns the portion of lineUpToCursor that precedes the
// LP: # / Closes: # trigger.  This text is used as a query when ranking bug
// completions by title similarity.  Returns "" when no bug-ref trigger is
// present on the line.
func ContextBeforeBugRef(lineUpToCursor string) string {
	loc := bugRefRe.FindStringIndex(lineUpToCursor)
	if loc == nil {
		return ""
	}
	return lineUpToCursor[:loc[0]]
}

// ChangelogContextForBugRef returns a query string for ranking bug completions
// when the cursor is inside an "LP: #" or "Closes: #" reference.
//
// It starts with the text before the trigger on the current line.  When that
// line is a sub-bullet (and therefore often sparse), it also appends the text
// of the nearest parent top-level bullet ("  * <text>") found by scanning
// backwards.  This lets keywords on the parent line — e.g. "merge" in
// "  * Merge linux-glibc-dev from upstream" — influence the ranking even when
// the sub-bullet itself contains no descriptive words.
func ChangelogContextForBugRef(text string, lineNum int, lineUpTo string) string {
	context := ContextBeforeBugRef(lineUpTo)

	lines := strings.Split(text, "\n")
	if lineNum <= 0 || lineNum >= len(lines) {
		return context
	}

	// Only enrich when the current line is a sub-bullet; top-level bullets
	// already carry their full description as context.
	if !subBulletRe.MatchString(lines[lineNum]) {
		return context
	}

	// Scan backwards for the nearest top-level bullet.
	for i := lineNum - 1; i >= 0; i-- {
		l := lines[i]
		if loc := topBulletRe.FindStringIndex(l); loc != nil {
			// Append everything after "  * ".
			context = context + " " + l[loc[1]:]
			break
		}
		// Blank line or non-indented header — stop.
		if strings.TrimSpace(l) == "" || !strings.HasPrefix(l, " ") {
			break
		}
		// Sub-bullet or indented continuation — keep scanning.
	}
	return context
}

// BugNumbersInText returns the set of bug IDs already referenced anywhere in
// text (via "LP: #N" or "Closes: #N").  Used to filter already-mentioned bugs
// from completion lists.
func BugNumbersInText(text string) map[int]bool {
	out := make(map[int]bool)
	for _, m := range bugNumberRe.FindAllStringSubmatch(text, -1) {
		if n, err := strconv.Atoi(m[2]); err == nil {
			out[n] = true
		}
	}
	return out
}

// subBulletRe matches a changelog sub-bullet line: leading whitespace, a dash,
// a space, then the rest of the text typed so far.
var subBulletRe = regexp.MustCompile(`^(\s+-\s)(.*)$`)

// topBulletRe matches a changelog top-level bullet: leading whitespace, an
// asterisk, then a space (the content follows).
var topBulletRe = regexp.MustCompile(`^\s+\*\s`)

// ChangelogFixesBulletAtLine reports whether line lineNum (0-indexed) within
// text is a sub-bullet (`    - <typed text>`) whose nearest parent top-level
// bullet (`  * ...`) contains the word "fixes:".  When it does, the function
// returns the column of the first user-typed character (just after the "- ")
// and the text the user has typed so far on that line.
//
// The backwards scan stops at the first top-level bullet found.  If that
// bullet does not contain "fixes:", or if a blank line / header line is
// encountered first, ok is false.
func ChangelogFixesBulletAtLine(text string, lineNum int) (startCol int, typedText string, ok bool) {
	lines := strings.Split(text, "\n")
	if lineNum < 0 || lineNum >= len(lines) {
		return 0, "", false
	}

	// 1. Current line must be a sub-bullet.
	m := subBulletRe.FindStringSubmatch(lines[lineNum])
	if m == nil {
		return 0, "", false
	}
	col := len(m[1])
	typed := m[2]

	// 2. Scan backwards for the nearest top-level bullet.
	for i := lineNum - 1; i >= 0; i-- {
		l := lines[i]
		if topBulletRe.MatchString(l) {
			// Found a parent bullet — check for "fixes:" anywhere in it.
			if strings.Contains(strings.ToLower(l), "fixes:") {
				return col, typed, true
			}
			return 0, "", false
		}
		// Another sub-bullet or continuation line — keep scanning.
		if subBulletRe.MatchString(l) {
			continue
		}
		// Blank line or header line (non-indented) — stop.
		trimmed := strings.TrimSpace(l)
		if trimmed == "" || !strings.HasPrefix(l, " ") {
			return 0, "", false
		}
		// Indented continuation line of the previous bullet — keep scanning.
	}
	return 0, "", false
}
