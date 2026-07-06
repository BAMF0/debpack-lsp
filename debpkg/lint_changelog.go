// SPDX-License-Identifier: GPL-3.0-or-later

package debpkg

import (
	"fmt"
	"regexp"
	"strings"
)

// ---------------------------------------------------------------------------
// Compiled regexes
// ---------------------------------------------------------------------------

// changelogHeaderRe matches the first line of a changelog entry:
//
//	package (version) suite; urgency=value
var changelogHeaderRe = regexp.MustCompile(`^\S+ \([^)]+\) \S+; urgency=\S+`)

// changelogTrailerRe matches a trailer line that starts with " -- ".
var changelogTrailerRe = regexp.MustCompile(`^ -- `)

// trailerFullRe matches a well-formed trailer line including double-space
// before the date and a basic RFC 2822 date structure.
var trailerFullRe = regexp.MustCompile(
	`^ -- .+ <[^>]+>  \w+, \d+ \w+ \d{4} \d{2}:\d{2}:\d{2} [+-]\d{4}$`,
)

// trailerSingleSpaceRe detects the common mistake of one space instead of two
// between the closing '>' of the email and the weekday.
var trailerSingleSpaceRe = regexp.MustCompile(`^ -- .+ <[^>]+> \w`)

// urgencyRe extracts the urgency keyword from a header line.
var urgencyRe = regexp.MustCompile(`(?i)urgency=([a-z]+)`)

// bulletLineRe matches a bullet item at any indentation level.
// Group 1 = leading spaces, group 2 = bullet character (* - +).
var bulletLineRe = regexp.MustCompile(`^( +)([*\-+]) `)

// contributorLineRe matches contributor attribution blocks like "  [ Name ]".
var contributorLineRe = regexp.MustCompile(`^  \[.+\]\s*$`)

// ---------------------------------------------------------------------------
// Known urgency values
// ---------------------------------------------------------------------------

var validUrgencies = map[string]bool{
	"low": true, "medium": true, "high": true,
	"emergency": true, "critical": true,
}

// ---------------------------------------------------------------------------
// Entry point
// ---------------------------------------------------------------------------

func lintChangelog(text string) []Diag {
	var diags []Diag
	lines := splitLines(text)
	n := len(lines)
	i := 0

	for i < n {
		// Skip blank / whitespace-only lines between entries.
		if isBlank(lines[i]) {
			i++
			continue
		}

		// Expect an entry header line.
		if !changelogHeaderRe.MatchString(lines[i]) {
			// Not a recognised header — skip (be lenient during editing).
			i++
			continue
		}

		headerIdx := i
		diags = append(diags, checkEntryHeader(lines[headerIdx], headerIdx)...)
		diags = append(diags, checkUrgency(lines[headerIdx], headerIdx)...)

		// Second line of an entry MUST be truly blank (empty string).
		if headerIdx+1 < n {
			secondLine := lines[headerIdx+1]
			if !isBlank(secondLine) {
				diags = append(diags, Diag{
					Line: headerIdx + 1, Col: 0,
					EndLine: headerIdx + 1, EndCol: len(secondLine),
					Severity: SeverityError,
					Message:  "line after entry header must be blank",
				})
			}
		}

		// Locate the trailer line for this entry.
		trailerIdx := -1
		for j := headerIdx + 1; j < n; j++ {
			if changelogTrailerRe.MatchString(lines[j]) {
				trailerIdx = j
				break
			}
			// Another entry header means the trailer is missing — stop here.
			if j > headerIdx+1 && changelogHeaderRe.MatchString(lines[j]) {
				break
			}
		}

		if trailerIdx == -1 {
			// No trailer found (partial entry during editing) — move on.
			i = headerIdx + 1
			continue
		}

		// Check blank line(s) immediately before the trailer.
		blanksBefore := countConsecutiveBlanksBackward(lines, trailerIdx-1, headerIdx+1)
		if blanksBefore == 0 {
			diags = append(diags, Diag{
				Line: trailerIdx, Col: 0, EndLine: trailerIdx, EndCol: 0,
				Severity: SeverityError,
				Message:  "expected exactly one blank line before the trailer ( -- ) line",
			})
		} else if blanksBefore > 1 {
			diags = append(diags, Diag{
				Line:     trailerIdx - blanksBefore,
				Col:      0,
				EndLine:  trailerIdx - 1,
				EndCol:   0,
				Severity: SeverityError,
				Message:  "expected exactly one blank line before the trailer ( -- ) line",
			})
		}

		// Check blank line(s) immediately after the trailer (except last entry).
		if trailerIdx+1 < n {
			blanksAfter := countConsecutiveBlanksForward(lines, trailerIdx+1, n)
			if blanksAfter == 0 {
				diags = append(diags, Diag{
					Line: trailerIdx + 1, Col: 0, EndLine: trailerIdx + 1, EndCol: 0,
					Severity: SeverityError,
					Message:  "expected exactly one blank line after the trailer ( -- ) line",
				})
			} else if blanksAfter > 1 {
				diags = append(diags, Diag{
					Line:     trailerIdx + 1,
					Col:      0,
					EndLine:  trailerIdx + blanksAfter,
					EndCol:   0,
					Severity: SeverityError,
					Message:  "expected exactly one blank line after the trailer ( -- ) line",
				})
			}
		}

		// Check trailer format.
		diags = append(diags, checkTrailer(lines[trailerIdx], trailerIdx)...)

		// Check body indentation.
		bodyStart := headerIdx + 2
		bodyEnd := trailerIdx - blanksBefore
		if bodyEnd > bodyStart {
			diags = append(diags, checkBodyIndentation(lines, bodyStart, bodyEnd)...)
		}

		i = trailerIdx + 1
	}

	return diags
}

// ---------------------------------------------------------------------------
// Header / urgency checks
// ---------------------------------------------------------------------------

func checkEntryHeader(line string, lineNum int) []Diag {
	if changelogHeaderRe.MatchString(line) {
		return nil
	}
	return []Diag{{
		Line: lineNum, Col: 0, EndLine: lineNum, EndCol: len(line),
		Severity: SeverityError,
		Message:  `changelog entry header must be "package (version) suite; urgency=value"`,
	}}
}

func checkUrgency(line string, lineNum int) []Diag {
	m := urgencyRe.FindStringSubmatch(line)
	if m == nil {
		return nil
	}
	val := strings.ToLower(m[1])
	if validUrgencies[val] {
		return nil
	}
	col := strings.Index(line, m[0])
	return []Diag{{
		Line: lineNum, Col: col, EndLine: lineNum, EndCol: col + len(m[0]),
		Severity: SeverityWarning,
		Message:  fmt.Sprintf("unknown urgency %q; expected one of: low, medium, high, emergency, critical", m[1]),
	}}
}

// ---------------------------------------------------------------------------
// Trailer check
// ---------------------------------------------------------------------------

func checkTrailer(line string, lineNum int) []Diag {
	if trailerFullRe.MatchString(line) {
		return nil
	}
	// Give a more specific message for the single-space mistake.
	if trailerSingleSpaceRe.MatchString(line) {
		return []Diag{{
			Line: lineNum, Col: 0, EndLine: lineNum, EndCol: len(line),
			Severity: SeverityError,
			Message:  `trailer requires two spaces between closing '>' and the date: " -- Name <email>  Weekday, ..."`,
		}}
	}
	return []Diag{{
		Line: lineNum, Col: 0, EndLine: lineNum, EndCol: len(line),
		Severity: SeverityError,
		Message:  `malformed trailer; expected " -- Name <email>  Weekday, DD Mon YYYY HH:MM:SS ±ZZZZ"`,
	}}
}

// ---------------------------------------------------------------------------
// Body indentation check
// ---------------------------------------------------------------------------

func checkBodyIndentation(lines []string, start, end int) []Diag {
	var diags []Diag
	currentBulletIndent := -1 // spaces before the bullet character

	for i := start; i < end; i++ {
		line := lines[i]
		if isBlank(line) {
			currentBulletIndent = -1
			continue
		}

		// Contributor attribution block: "  [ Name ]"
		if contributorLineRe.MatchString(line) {
			currentBulletIndent = -1
			continue
		}

		// Bullet line?
		if m := bulletLineRe.FindStringSubmatch(line); m != nil {
			indent := len(m[1])
			currentBulletIndent = indent
			if indent%2 != 0 {
				diags = append(diags, Diag{
					Line: i, Col: 0, EndLine: i, EndCol: indent,
					Severity: SeverityWarning,
					Message:  fmt.Sprintf("bullet indented with %d space(s); indentation must be a multiple of 2", indent),
				})
			}
			continue
		}

		// Continuation or other content line.
		actual := countLeadingSpaces(line)
		if actual < 2 {
			diags = append(diags, Diag{
				Line: i, Col: 0, EndLine: i, EndCol: len(line),
				Severity: SeverityWarning,
				Message:  "body line must be indented by at least 2 spaces",
			})
		} else if currentBulletIndent >= 0 {
			expected := currentBulletIndent + 2 // text starts after "X " (bullet + space)
			if actual < expected {
				diags = append(diags, Diag{
					Line: i, Col: 0, EndLine: i, EndCol: actual,
					Severity: SeverityWarning,
					Message: fmt.Sprintf(
						"continuation line indented %d space(s); expected at least %d to align with enclosing bullet",
						actual, expected,
					),
				})
			}
		}
	}
	return diags
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// countConsecutiveBlanksBackward counts how many consecutive blank lines there
// are immediately before index from, stopping at stopAt (inclusive).
func countConsecutiveBlanksBackward(lines []string, from, stopAt int) int {
	count := 0
	for k := from; k >= stopAt && isBlank(lines[k]); k-- {
		count++
	}
	return count
}

// countConsecutiveBlanksForward counts how many consecutive blank lines there
// are starting at index from, stopping before n.
func countConsecutiveBlanksForward(lines []string, from, n int) int {
	count := 0
	for k := from; k < n && isBlank(lines[k]); k++ {
		count++
	}
	return count
}
