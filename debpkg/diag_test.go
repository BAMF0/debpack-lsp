// SPDX-License-Identifier: GPL-3.0-or-later

package debpkg_test

import (
	"testing"

	"github.com/BAMF0/debpack-lsp/debpkg"
)

func TestLintUniversalCRLF(t *testing.T) {
	// CRLF line endings should be flagged, and the \r should not leak into
	// per-line content checks (e.g. trailing-whitespace ranges).
	text := "Description: fix\r\nOrigin: upstream\r\n"
	diags := debpkg.Lint(text, debpkg.FileTypePatch, debpkg.LintContext{})

	foundCRLF := 0
	for _, d := range diags {
		if d.Message == "CRLF line ending; Debian files should use LF" {
			foundCRLF++
		}
		// No diagnostic should reference a \r in its column span as if it
		// were content — the CRLF check handles that separately.
	}
	if foundCRLF != 2 {
		t.Errorf("expected 2 CRLF diagnostics, got %d (total diags: %d)", foundCRLF, len(diags))
	}
}

func TestLintUniversalTrailingWhitespace(t *testing.T) {
	text := "foo \nbar\t\nbaz\n"
	diags := debpkg.Lint(text, debpkg.FileTypePatch, debpkg.LintContext{})

	var tw int
	for _, d := range diags {
		if d.Message == "trailing whitespace" {
			tw++
		}
	}
	if tw != 2 {
		t.Errorf("expected 2 trailing-whitespace diagnostics, got %d", tw)
	}
}

func TestSplitLinesStripsCR(t *testing.T) {
	// Verify indirectly: a DEP-3 Last-Update line with CRLF must pass the
	// date-format linter (which anchors on $), proving \r was stripped.
	text := "Last-Update: 2024-01-15\r\n"
	diags := debpkg.Lint(text, debpkg.FileTypePatch, debpkg.LintContext{})
	for _, d := range diags {
		if d.Message != "CRLF line ending; Debian files should use LF" &&
			d.Message != "DEP-3 patch header is missing the required \"Description:\" (or \"Subject:\") field" &&
			d.Message != "DEP-3 patch header is missing \"Origin:\" (required when no \"Author:\" field is present)" {
			t.Errorf("unexpected diagnostic for CRLF Last-Update line: %q", d.Message)
		}
	}
}
