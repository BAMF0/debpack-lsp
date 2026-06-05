package debpkg

import (
	"strings"
)

// FormatChangelog returns a canonically formatted version of a debian/changelog
// file.  It is idempotent: applying it twice produces the same result.
//
// Rules applied:
//   - Any "*" bullet at any indentation  → "  * text"
//   - Any "-" or "+" bullet at any indentation → "    - text" ("+" normalised to "-")
//   - Continuation of a "  * " bullet → 4-space indent (aligns with bullet text)
//   - Continuation of a "    - " bullet → 6-space indent (aligns with sub-bullet text)
//   - Whitespace-only blank lines → truly empty ("")
//   - Multiple consecutive blank lines → collapsed to one
//   - Blank line after the entry header → enforced (inserted if absent)
//   - Blank line before the trailer → enforced (extra blanks collapsed, inserted if absent)
//   - Blank line between entries → exactly one
//   - Trailing whitespace → stripped from every line
//   - "  [ Name ]" contributor blocks → left verbatim
//   - Header and trailer lines → left verbatim (dch-managed)
func FormatChangelog(text string) string {
	raw := strings.Split(text, "\n")
	// Drop a trailing empty element produced by a final newline so we don't
	// process a phantom blank line at the end.
	if len(raw) > 0 && raw[len(raw)-1] == "" {
		raw = raw[:len(raw)-1]
	}
	n := len(raw)

	// out accumulates the formatted lines.
	var out []string

	// emit appends s to out after stripping trailing whitespace.
	emit := func(s string) {
		out = append(out, strings.TrimRight(s, " \t"))
	}

	// emitBlank appends an empty line only when the last output line is not
	// already empty (idempotent blank insertion).
	emitBlank := func() {
		if len(out) == 0 || out[len(out)-1] != "" {
			out = append(out, "")
		}
	}

	// popTrailingBlanks removes any trailing empty lines from out.
	popTrailingBlanks := func() {
		for len(out) > 0 && out[len(out)-1] == "" {
			out = out[:len(out)-1]
		}
	}

	// formatBodyLine re-indents a non-blank, non-contributor body line.
	// contIndent is updated in place when a bullet is encountered.
	formatBodyLine := func(line string, contIndent *int) string {
		if m := bulletLineRe.FindStringSubmatch(line); m != nil {
			content := line[len(m[0]):]
			switch m[2] { // bullet character
			case "*":
				*contIndent = 4
				return "  * " + content
			case "+":
				// "+" is used both as a non-standard top-level variant (≤2 spaces
				// of source indentation) and as the canonical third-level bullet
				// (≥3 spaces, i.e. after a "    - " sub-bullet).  Distinguish by
				// source indent so we don't collapse level-3 bullets into level 1.
				if len(m[1]) <= 2 {
					*contIndent = 4
					return "  * " + content
				}
				*contIndent = 8
				return "      + " + content
			default: // "-" → canonical level-2 sub-bullet
				*contIndent = 6
				return "    - " + content
			}
		}
		// Continuation line: strip existing indentation and re-apply canonical.
		content := strings.TrimLeft(line, " \t")
		return strings.Repeat(" ", *contIndent) + content
	}

	i := 0
	for i < n {
		line := strings.TrimRight(raw[i], " \t")

		// Skip blank lines at the top level; inter-entry spacing is managed
		// explicitly when we encounter each entry header.
		if line == "" {
			i++
			continue
		}

		// Non-header, non-blank top-level line (unexpected) — emit as-is.
		if !changelogHeaderRe.MatchString(line) {
			emit(line)
			i++
			continue
		}

		// ----------------------------------------------------------------
		// Entry header found.
		// ----------------------------------------------------------------

		// Exactly one blank line between entries (not before the very first).
		if len(out) > 0 {
			emitBlank()
		}
		emit(line) // header verbatim
		i++

		// Locate the trailer for this entry by scanning forward.
		trailerIdx := -1
		for j := i; j < n; j++ {
			if changelogTrailerRe.MatchString(raw[j]) {
				trailerIdx = j
				break
			}
			// Another header with no trailer found — treat body as open-ended.
			if j > i && changelogHeaderRe.MatchString(raw[j]) {
				break
			}
		}

		// Emit the mandatory blank line that follows the header.
		emitBlank()
		// If the source already has a blank there, skip past it so we don't
		// emit it again in the body loop.
		if i < n && strings.TrimRight(raw[i], " \t") == "" {
			i++
		}

		if trailerIdx == -1 {
			// Partial entry (file being edited) — format remaining lines and stop.
			contIndent := 4
			for i < n {
				bl := strings.TrimRight(raw[i], " \t")
				if bl == "" {
					emitBlank()
					contIndent = 4
				} else if contributorLineRe.MatchString(bl) {
					emit(bl)
					contIndent = 4
				} else {
					emit(formatBodyLine(bl, &contIndent))
				}
				i++
			}
			continue
		}

		// ----------------------------------------------------------------
		// Format body lines from i up to (not including) trailerIdx.
		// ----------------------------------------------------------------
		contIndent := 4
		for i < trailerIdx {
			bl := strings.TrimRight(raw[i], " \t")
			if bl == "" {
				emitBlank()
				contIndent = 4
			} else if contributorLineRe.MatchString(bl) {
				emit(bl)
				contIndent = 4
			} else {
				emit(formatBodyLine(bl, &contIndent))
			}
			i++
		}

		// Ensure exactly one blank line before the trailer.
		popTrailingBlanks()
		emitBlank()

		// Emit the trailer verbatim.
		emit(strings.TrimRight(raw[trailerIdx], " \t"))
		i = trailerIdx + 1
	}

	return strings.Join(out, "\n") + "\n"
}
