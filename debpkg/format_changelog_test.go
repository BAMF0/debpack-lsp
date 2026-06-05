package debpkg

import (
	"strings"
	"testing"
)

// canonical is a correctly-formatted two-entry changelog used as the baseline
// for the idempotency test and as the expected output for the normalisation tests.
const canonicalChangelog = `mypkg (2.0) unstable; urgency=medium

  * Fix crash when foo is nil
    Detailed explanation on the continuation line.
  * Improve performance of bar
    - Sub-item one
    - Sub-item two
      Continuation of sub-item two.

 -- Alice Dev <alice@example.com>  Mon, 01 Jan 2024 12:00:00 +0000

mypkg (1.0) unstable; urgency=low

  * Initial release

 -- Alice Dev <alice@example.com>  Fri, 01 Jan 2021 00:00:00 +0000
`

func TestFormatChangelog_Idempotent(t *testing.T) {
	got := FormatChangelog(canonicalChangelog)
	if got != canonicalChangelog {
		t.Errorf("idempotency failure:\nwant:\n%s\ngot:\n%s", canonicalChangelog, got)
	}
	// Apply a second time.
	got2 := FormatChangelog(got)
	if got2 != got {
		t.Errorf("second-pass idempotency failure:\ngot:\n%s", got2)
	}
}

func TestFormatChangelog_BulletNormalisation(t *testing.T) {
	// Bullets with wrong indentation and non-canonical characters.
	input := `mypkg (1.0) unstable; urgency=medium

   * Three-space top-level bullet
  + Plus top-level (normalise to * and keep indent)
      - Deeply-indented sub-bullet
  - Sub-bullet at top-level indent

 -- Dev <dev@example.com>  Mon, 01 Jan 2024 00:00:00 +0000
`
	want := `mypkg (1.0) unstable; urgency=medium

  * Three-space top-level bullet
  * Plus top-level (normalise to * and keep indent)
    - Deeply-indented sub-bullet
    - Sub-bullet at top-level indent

 -- Dev <dev@example.com>  Mon, 01 Jan 2024 00:00:00 +0000
`
	got := FormatChangelog(input)
	if got != want {
		t.Errorf("bullet normalisation:\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatChangelog_ContinuationReindent(t *testing.T) {
	input := `mypkg (1.0) unstable; urgency=medium

  * Top-level bullet
  continuation with wrong indent
    - Sub-bullet
  continuation of sub with wrong indent

 -- Dev <dev@example.com>  Mon, 01 Jan 2024 00:00:00 +0000
`
	want := `mypkg (1.0) unstable; urgency=medium

  * Top-level bullet
    continuation with wrong indent
    - Sub-bullet
      continuation of sub with wrong indent

 -- Dev <dev@example.com>  Mon, 01 Jan 2024 00:00:00 +0000
`
	got := FormatChangelog(input)
	if got != want {
		t.Errorf("continuation re-indent:\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatChangelog_BlankLineCollapsing(t *testing.T) {
	// Three blanks before the trailer and two between entries.
	input := `mypkg (2.0) unstable; urgency=medium

  * Change one



 -- Dev <dev@example.com>  Mon, 01 Jan 2024 00:00:00 +0000


mypkg (1.0) unstable; urgency=low

  * Initial release

 -- Dev <dev@example.com>  Fri, 01 Jan 2021 00:00:00 +0000
`
	want := `mypkg (2.0) unstable; urgency=medium

  * Change one

 -- Dev <dev@example.com>  Mon, 01 Jan 2024 00:00:00 +0000

mypkg (1.0) unstable; urgency=low

  * Initial release

 -- Dev <dev@example.com>  Fri, 01 Jan 2021 00:00:00 +0000
`
	got := FormatChangelog(input)
	if got != want {
		t.Errorf("blank collapsing:\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatChangelog_MissingBlankAfterHeader(t *testing.T) {
	input := `mypkg (1.0) unstable; urgency=medium
  * No blank after header

 -- Dev <dev@example.com>  Mon, 01 Jan 2024 00:00:00 +0000
`
	want := `mypkg (1.0) unstable; urgency=medium

  * No blank after header

 -- Dev <dev@example.com>  Mon, 01 Jan 2024 00:00:00 +0000
`
	got := FormatChangelog(input)
	if got != want {
		t.Errorf("missing blank after header:\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatChangelog_MissingBlankBeforeTrailer(t *testing.T) {
	input := `mypkg (1.0) unstable; urgency=medium

  * A change
 -- Dev <dev@example.com>  Mon, 01 Jan 2024 00:00:00 +0000
`
	want := `mypkg (1.0) unstable; urgency=medium

  * A change

 -- Dev <dev@example.com>  Mon, 01 Jan 2024 00:00:00 +0000
`
	got := FormatChangelog(input)
	if got != want {
		t.Errorf("missing blank before trailer:\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestFormatChangelog_TrailingWhitespace(t *testing.T) {
	// Build input with trailing spaces on various lines.
	lines := []string{
		"mypkg (1.0) unstable; urgency=medium",
		"",
		"  * A change with trailing spaces   ",
		"    continuation also with spaces   ",
		"",
		" -- Dev <dev@example.com>  Mon, 01 Jan 2024 00:00:00 +0000   ",
	}
	input := strings.Join(lines, "\n") + "\n"

	want := `mypkg (1.0) unstable; urgency=medium

  * A change with trailing spaces
    continuation also with spaces

 -- Dev <dev@example.com>  Mon, 01 Jan 2024 00:00:00 +0000

`
	// Note: the trailer itself has trailing spaces stripped, but the blank
	// line after it plus the final newline are preserved by the formatter.
	// Strip the extra trailing blank from want for comparison — the formatter
	// emits a final "\n" but no trailing blank line after the last trailer.
	want = `mypkg (1.0) unstable; urgency=medium

  * A change with trailing spaces
    continuation also with spaces

 -- Dev <dev@example.com>  Mon, 01 Jan 2024 00:00:00 +0000
`
	got := FormatChangelog(input)
	if got != want {
		t.Errorf("trailing whitespace:\nwant:\n%q\ngot:\n%q", want, got)
	}
}

func TestFormatChangelog_ContributorBlock(t *testing.T) {
	// Contributor blocks ("  [ Name ]") must pass through unchanged.
	input := `mypkg (1.0) unstable; urgency=medium

  [ Alice ]
  * Alice's change

  [ Bob ]
  * Bob's change

 -- Alice Dev <alice@example.com>  Mon, 01 Jan 2024 00:00:00 +0000
`
	got := FormatChangelog(input)
	if got != input {
		t.Errorf("contributor block should be preserved:\nwant:\n%s\ngot:\n%s", input, got)
	}
}

func TestFormatChangelog_WhitespaceOnlyBlankLines(t *testing.T) {
	// Blank lines that contain only spaces must be collapsed to truly empty lines.
	input := "mypkg (1.0) unstable; urgency=medium\n\n  * Change\n   \n -- Dev <dev@example.com>  Mon, 01 Jan 2024 00:00:00 +0000\n"
	want := `mypkg (1.0) unstable; urgency=medium

  * Change

 -- Dev <dev@example.com>  Mon, 01 Jan 2024 00:00:00 +0000
`
	got := FormatChangelog(input)
	if got != want {
		t.Errorf("whitespace-only blanks:\nwant:\n%q\ngot:\n%q", want, got)
	}
}
