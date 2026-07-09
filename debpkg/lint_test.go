// SPDX-License-Identifier: GPL-3.0-or-later

package debpkg_test

import (
	"strings"
	"testing"

	"github.com/BAMF0/debpack-lsp/debpkg"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// assertNoDiags fails if any diagnostics were produced.
func assertNoDiags(t *testing.T, diags []debpkg.Diag) {
	t.Helper()
	if len(diags) != 0 {
		t.Errorf("expected no diagnostics, got %d:", len(diags))
		for _, d := range diags {
			t.Errorf("  line %d [sev=%d]: %s", d.Line, d.Severity, d.Message)
		}
	}
}

// assertHasDiag fails unless at least one diagnostic matches the given
// severity and has a message containing substr.
func assertHasDiag(t *testing.T, diags []debpkg.Diag, sev debpkg.Severity, substr string) {
	t.Helper()
	for _, d := range diags {
		if d.Severity == sev && strings.Contains(d.Message, substr) {
			return
		}
	}
	t.Errorf("expected diagnostic sev=%d containing %q; got:", sev, substr)
	for _, d := range diags {
		t.Errorf("  line %d [sev=%d]: %s", d.Line, d.Severity, d.Message)
	}
}

// assertNoDiagContaining fails if any diagnostic message contains substr.
func assertNoDiagContaining(t *testing.T, diags []debpkg.Diag, substr string) {
	t.Helper()
	for _, d := range diags {
		if strings.Contains(d.Message, substr) {
			t.Errorf("unexpected diagnostic containing %q at line %d: %s", substr, d.Line, d.Message)
		}
	}
}

// lint is a shorthand that calls debpkg.Lint with an optional LintContext.
func lint(text string, ft debpkg.FileType, ctx ...debpkg.LintContext) []debpkg.Diag {
	var c debpkg.LintContext
	if len(ctx) > 0 {
		c = ctx[0]
	}
	return debpkg.Lint(text, ft, c)
}

// ---------------------------------------------------------------------------
// Universal pre-pass tests
// ---------------------------------------------------------------------------

func TestLintUniversal_TrailingWhitespace(t *testing.T) {
	// Use a minimal watch file so the file type is recognised.
	text := "version=4  \nhttps://example.com/foo-(.+).tar.gz\n"
	diags := lint(text, debpkg.FileTypeWatch)
	assertHasDiag(t, diags, debpkg.SeverityWarning, "trailing whitespace")
}

func TestLintUniversal_WhitespaceOnlyBlankLine(t *testing.T) {
	text := "version=4\n   \nhttps://example.com/foo-(.+).tar.gz\n"
	diags := lint(text, debpkg.FileTypeWatch)
	assertHasDiag(t, diags, debpkg.SeverityWarning, "blank line contains whitespace")
}

func TestLintUniversal_UnknownFileType(t *testing.T) {
	// FileTypeUnknown must return nil immediately — no universal pass.
	diags := debpkg.Lint("trailing   \n", debpkg.FileTypeUnknown, debpkg.LintContext{})
	if len(diags) != 0 {
		t.Errorf("expected no diagnostics for FileTypeUnknown, got %d", len(diags))
	}
}

// ---------------------------------------------------------------------------
// Changelog lint tests
// ---------------------------------------------------------------------------

const goodChangelog = `curl (7.88.1-2) unstable; urgency=medium

  * Fix crash in HTTP/2 handling.

 -- Test User <test@example.com>  Mon, 01 Jan 2024 12:00:00 +0000
`

func TestLintChangelog_Valid(t *testing.T) {
	assertNoDiags(t, lint(goodChangelog, debpkg.FileTypeChangelog))
}

func TestLintChangelog_ValidMultipleEntries(t *testing.T) {
	text := `curl (7.88.1-2) unstable; urgency=medium

  * Fix crash.

 -- Test User <test@example.com>  Mon, 01 Jan 2024 12:00:00 +0000

curl (7.88.1-1) unstable; urgency=low

  * Initial release.

 -- Test User <test@example.com>  Mon, 01 Jan 2023 12:00:00 +0000
`
	assertNoDiags(t, lint(text, debpkg.FileTypeChangelog))
}

func TestLintChangelog_MissingBlankAfterHeader(t *testing.T) {
	text := `curl (7.88.1-2) unstable; urgency=medium
  * Fix crash.

 -- Test User <test@example.com>  Mon, 01 Jan 2024 12:00:00 +0000
`
	assertHasDiag(t, lint(text, debpkg.FileTypeChangelog), debpkg.SeverityError, "line after entry header must be blank")
}

func TestLintChangelog_MissingBlankBeforeTrailer(t *testing.T) {
	text := `curl (7.88.1-2) unstable; urgency=medium

  * Fix crash.
 -- Test User <test@example.com>  Mon, 01 Jan 2024 12:00:00 +0000
`
	assertHasDiag(t, lint(text, debpkg.FileTypeChangelog), debpkg.SeverityError, "blank line before the trailer")
}

func TestLintChangelog_MultipleBlankBeforeTrailer(t *testing.T) {
	text := `curl (7.88.1-2) unstable; urgency=medium

  * Fix crash.


 -- Test User <test@example.com>  Mon, 01 Jan 2024 12:00:00 +0000
`
	assertHasDiag(t, lint(text, debpkg.FileTypeChangelog), debpkg.SeverityError, "blank line before the trailer")
}

func TestLintChangelog_MissingBlankAfterTrailer(t *testing.T) {
	// Two entries with no blank line between them.
	text := `curl (7.88.1-2) unstable; urgency=medium

  * Fix crash.

 -- Test User <test@example.com>  Mon, 01 Jan 2024 12:00:00 +0000
curl (7.88.1-1) unstable; urgency=low

  * Initial release.

 -- Test User <test@example.com>  Mon, 01 Jan 2023 12:00:00 +0000
`
	assertHasDiag(t, lint(text, debpkg.FileTypeChangelog), debpkg.SeverityError, "blank line after the trailer")
}

func TestLintChangelog_TrailerSingleSpace(t *testing.T) {
	// One space between '>' and date instead of two.
	text := `curl (7.88.1-2) unstable; urgency=medium

  * Fix crash.

 -- Test User <test@example.com> Mon, 01 Jan 2024 12:00:00 +0000
`
	assertHasDiag(t, lint(text, debpkg.FileTypeChangelog), debpkg.SeverityError, "two spaces")
}

func TestLintChangelog_TrailerMalformed(t *testing.T) {
	text := `curl (7.88.1-2) unstable; urgency=medium

  * Fix crash.

 -- totally wrong trailer
`
	assertHasDiag(t, lint(text, debpkg.FileTypeChangelog), debpkg.SeverityError, "malformed trailer")
}

func TestLintChangelog_UnknownUrgency(t *testing.T) {
	text := `curl (7.88.1-2) unstable; urgency=extreme

  * Fix crash.

 -- Test User <test@example.com>  Mon, 01 Jan 2024 12:00:00 +0000
`
	assertHasDiag(t, lint(text, debpkg.FileTypeChangelog), debpkg.SeverityWarning, "unknown urgency")
}

func TestLintChangelog_ValidUrgencies(t *testing.T) {
	for _, u := range []string{"low", "medium", "high", "emergency", "critical"} {
		text := "curl (7.88.1-2) unstable; urgency=" + u + "\n\n  * Fix.\n\n -- T <t@t.com>  Mon, 01 Jan 2024 12:00:00 +0000\n"
		diags := lint(text, debpkg.FileTypeChangelog)
		assertNoDiagContaining(t, diags, "unknown urgency")
	}
}

func TestLintChangelog_BulletOddIndent(t *testing.T) {
	// Three spaces before '*' — not a multiple of 2.
	text := `curl (7.88.1-2) unstable; urgency=medium

   * Oddly indented bullet.

 -- Test User <test@example.com>  Mon, 01 Jan 2024 12:00:00 +0000
`
	assertHasDiag(t, lint(text, debpkg.FileTypeChangelog), debpkg.SeverityWarning, "multiple of 2")
}

func TestLintChangelog_ContinuationUnderIndented(t *testing.T) {
	// Continuation line at 2 spaces; the enclosing '  *' bullet requires >= 4.
	text := `curl (7.88.1-2) unstable; urgency=medium

  * Fix crash in HTTP/2 handling which affects a
  large number of connections.

 -- Test User <test@example.com>  Mon, 01 Jan 2024 12:00:00 +0000
`
	assertHasDiag(t, lint(text, debpkg.FileTypeChangelog), debpkg.SeverityWarning, "continuation line indented")
}

func TestLintChangelog_ContributorBlockExempt(t *testing.T) {
	// '  [ Name ]' lines must not trigger body-indentation warnings.
	text := `curl (7.88.1-2) unstable; urgency=medium

  [ Some Contributor ]
  * Fix crash.

 -- Test User <test@example.com>  Mon, 01 Jan 2024 12:00:00 +0000
`
	assertNoDiags(t, lint(text, debpkg.FileTypeChangelog))
}

// ---------------------------------------------------------------------------
// Control lint tests
// ---------------------------------------------------------------------------

const goodControl = `Source: curl
Section: libs
Priority: optional
Maintainer: Test User <test@example.com>
Standards-Version: 4.6.2

Package: libcurl4
Architecture: any
Description: Library for URL transfers
`

func TestLintControl_Valid(t *testing.T) {
	assertNoDiags(t, lint(goodControl, debpkg.FileTypeControl))
}

func TestLintControl_MissingSource(t *testing.T) {
	text := `Section: libs
Maintainer: Test User <test@example.com>
Standards-Version: 4.6.2
`
	assertHasDiag(t, lint(text, debpkg.FileTypeControl), debpkg.SeverityError, `"Source"`)
}

func TestLintControl_MissingMaintainer(t *testing.T) {
	text := `Source: curl
Section: libs
Standards-Version: 4.6.2
`
	assertHasDiag(t, lint(text, debpkg.FileTypeControl), debpkg.SeverityError, `"Maintainer"`)
}

func TestLintControl_MissingStandardsVersion(t *testing.T) {
	text := `Source: curl
Section: libs
Maintainer: Test User <test@example.com>
`
	assertHasDiag(t, lint(text, debpkg.FileTypeControl), debpkg.SeverityError, `"Standards-Version"`)
}

func TestLintControl_MissingSection(t *testing.T) {
	// Section is recommended, not mandatory — expect a warning only.
	text := `Source: curl
Maintainer: Test User <test@example.com>
Standards-Version: 4.6.2
`
	assertHasDiag(t, lint(text, debpkg.FileTypeControl), debpkg.SeverityWarning, "Section")
}

func TestLintControl_BadStandardsVersion(t *testing.T) {
	// Two-component version string — must be X.Y.Z or X.Y.Z.W.
	text := `Source: curl
Section: libs
Maintainer: Test User <test@example.com>
Standards-Version: 4.6
`
	assertHasDiag(t, lint(text, debpkg.FileTypeControl), debpkg.SeverityWarning, "Standards-Version")
}

func TestLintControl_StandardsVersionFourComponents(t *testing.T) {
	text := `Source: curl
Section: libs
Maintainer: Test User <test@example.com>
Standards-Version: 4.6.2.1
`
	assertNoDiagContaining(t, lint(text, debpkg.FileTypeControl), "Standards-Version")
}

func TestLintControl_UbuntuWrongMaintainer(t *testing.T) {
	// Ubuntu package (IsUbuntu=true) but Maintainer is not the canonical value.
	text := `Source: curl
Section: libs
Maintainer: Test User <test@example.com>
Standards-Version: 4.6.2
`
	diags := lint(text, debpkg.FileTypeControl, debpkg.LintContext{IsUbuntu: true})
	assertHasDiag(t, diags, debpkg.SeverityWarning, "Ubuntu")
}

func TestLintControl_UbuntuCorrectMaintainer(t *testing.T) {
	text := `Source: curl
Section: libs
Maintainer: Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>
Standards-Version: 4.6.2
`
	diags := lint(text, debpkg.FileTypeControl, debpkg.LintContext{IsUbuntu: true})
	assertNoDiagContaining(t, diags, "Ubuntu")
}

func TestLintControl_NonUbuntuUbuntuMaintainer(t *testing.T) {
	// Non-Ubuntu package (IsUbuntu=false) but Maintainer is the Ubuntu
	// canonical. This should NOT be flagged — native Ubuntu packages use
	// the Ubuntu Maintainer even when the version doesn't contain "ubuntu".
	text := `Source: curl
Section: libs
Maintainer: Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>
Standards-Version: 4.6.2
`
	diags := lint(text, debpkg.FileTypeControl, debpkg.LintContext{IsUbuntu: false})
	for _, d := range diags {
		if strings.Contains(d.Message, "non-Ubuntu package but Maintainer") {
			t.Errorf("should not flag non-Ubuntu with Ubuntu Maintainer (native package): %s", d.Message)
		}
	}
}

func TestLintControl_UnknownField(t *testing.T) {
	text := `Source: curl
Section: libs
Maintainer: Test User <test@example.com>
Standards-Version: 4.6.2
Foo-Bar: something
`
	assertHasDiag(t, lint(text, debpkg.FileTypeControl), debpkg.SeverityWarning, "unknown control field")
}

func TestLintControl_XPrefixedFieldsExempt(t *testing.T) {
	// XB-, XC-, XS-, and X- prefixed extension fields must not trigger warnings.
	text := `Source: curl
Section: libs
Maintainer: Test User <test@example.com>
Standards-Version: 4.6.2
XB-Custom-Field: value
XC-Another: value
XS-Third: value
X-Generic: value
`
	assertNoDiagContaining(t, lint(text, debpkg.FileTypeControl), "unknown control field")
}

func TestLintControl_UnknownSection(t *testing.T) {
	text := `Source: curl
Section: nonexistent-section
Maintainer: Test User <test@example.com>
Standards-Version: 4.6.2
`
	assertHasDiag(t, lint(text, debpkg.FileTypeControl), debpkg.SeverityWarning, "unknown section")
}

func TestLintControl_SectionWithAreaPrefix(t *testing.T) {
	// "non-free/libs" strips the area prefix to check "libs" — valid.
	text := `Source: curl
Section: non-free/libs
Maintainer: Test User <test@example.com>
Standards-Version: 4.6.2
`
	assertNoDiagContaining(t, lint(text, debpkg.FileTypeControl), "unknown section")
}

func TestLintControl_UnknownArchitecture(t *testing.T) {
	text := `Source: curl
Section: libs
Maintainer: Test User <test@example.com>
Standards-Version: 4.6.2

Package: libcurl4
Architecture: mips42
Description: Library for URL transfers
`
	assertHasDiag(t, lint(text, debpkg.FileTypeControl), debpkg.SeverityWarning, "unknown architecture")
}

func TestLintControl_NegatedArchitectureExempt(t *testing.T) {
	text := `Source: curl
Section: libs
Maintainer: Test User <test@example.com>
Standards-Version: 4.6.2

Package: libcurl4
Architecture: !amd64
Description: Library for URL transfers
`
	assertNoDiagContaining(t, lint(text, debpkg.FileTypeControl), "unknown architecture")
}

func TestLintControl_WildcardArchitectureExempt(t *testing.T) {
	text := `Source: curl
Section: libs
Maintainer: Test User <test@example.com>
Standards-Version: 4.6.2

Package: libcurl4
Architecture: linux-any
Description: Library for URL transfers
`
	assertNoDiagContaining(t, lint(text, debpkg.FileTypeControl), "unknown architecture")
}

func TestLintControl_InvalidPriority(t *testing.T) {
	text := `Source: curl
Section: libs
Priority: super-high
Maintainer: Test User <test@example.com>
Standards-Version: 4.6.2
`
	assertHasDiag(t, lint(text, debpkg.FileTypeControl), debpkg.SeverityWarning, "Priority")
}

func TestLintControl_ValidPriorities(t *testing.T) {
	for _, p := range []string{"required", "important", "standard", "optional", "extra"} {
		text := "Source: curl\nSection: libs\nPriority: " + p + "\nMaintainer: T <t@t.com>\nStandards-Version: 4.6.2\n"
		assertNoDiagContaining(t, lint(text, debpkg.FileTypeControl), "Priority")
	}
}

func TestLintControl_MissingBinaryPackage(t *testing.T) {
	text := `Source: curl
Section: libs
Maintainer: Test User <test@example.com>
Standards-Version: 4.6.2

Architecture: any
Description: Library for URL transfers
`
	assertHasDiag(t, lint(text, debpkg.FileTypeControl), debpkg.SeverityError, `"Package"`)
}

func TestLintControl_MissingBinaryArchitecture(t *testing.T) {
	text := `Source: curl
Section: libs
Maintainer: Test User <test@example.com>
Standards-Version: 4.6.2

Package: libcurl4
Description: Library for URL transfers
`
	assertHasDiag(t, lint(text, debpkg.FileTypeControl), debpkg.SeverityError, `"Architecture"`)
}

func TestLintControl_MissingBinaryDescription(t *testing.T) {
	text := `Source: curl
Section: libs
Maintainer: Test User <test@example.com>
Standards-Version: 4.6.2

Package: libcurl4
Architecture: any
`
	assertHasDiag(t, lint(text, debpkg.FileTypeControl), debpkg.SeverityError, `"Description"`)
}

// ---------------------------------------------------------------------------
// Copyright lint tests
// ---------------------------------------------------------------------------

const goodCopyright = `Format: https://www.debian.org/doc/packaging-manuals/copyright-format/1.0/
Upstream-Name: curl

Files: *
Copyright: 2024 Test User <test@example.com>
License: MIT
`

func TestLintCopyright_Valid(t *testing.T) {
	assertNoDiags(t, lint(goodCopyright, debpkg.FileTypeCopyright))
}

func TestLintCopyright_LegacyHTTPURIAccepted(t *testing.T) {
	// The http:// legacy URI is an accepted alternative.
	text := `Format: http://www.debian.org/doc/packaging-manuals/copyright-format/1.0/

Files: *
Copyright: 2024 Test User
License: MIT
`
	assertNoDiagContaining(t, lint(text, debpkg.FileTypeCopyright), "canonical DEP-5 URI")
}

func TestLintCopyright_MissingFormat(t *testing.T) {
	text := `Upstream-Name: curl

Files: *
Copyright: 2024 Test User
License: MIT
`
	assertHasDiag(t, lint(text, debpkg.FileTypeCopyright), debpkg.SeverityError, `"Format:"`)
}

func TestLintCopyright_WrongFormatURI(t *testing.T) {
	text := `Format: https://www.debian.org/doc/packaging-manuals/copyright-format/0.9/

Files: *
Copyright: 2024 Test User
License: MIT
`
	assertHasDiag(t, lint(text, debpkg.FileTypeCopyright), debpkg.SeverityWarning, "canonical DEP-5 URI")
}

func TestLintCopyright_MissingCopyrightInFilesStanza(t *testing.T) {
	text := `Format: https://www.debian.org/doc/packaging-manuals/copyright-format/1.0/

Files: *
License: MIT
`
	assertHasDiag(t, lint(text, debpkg.FileTypeCopyright), debpkg.SeverityError, `"Copyright:"`)
}

func TestLintCopyright_MissingLicenseInFilesStanza(t *testing.T) {
	text := `Format: https://www.debian.org/doc/packaging-manuals/copyright-format/1.0/

Files: *
Copyright: 2024 Test User
`
	assertHasDiag(t, lint(text, debpkg.FileTypeCopyright), debpkg.SeverityError, `"License:"`)
}

func TestLintCopyright_MissingCatchAll(t *testing.T) {
	// A Files: stanza exists but none matches '*'.
	text := `Format: https://www.debian.org/doc/packaging-manuals/copyright-format/1.0/

Files: src/*
Copyright: 2024 Test User
License: MIT
`
	assertHasDiag(t, lint(text, debpkg.FileTypeCopyright), debpkg.SeverityWarning, "catch-all")
}

// ---------------------------------------------------------------------------
// Watch lint tests
// ---------------------------------------------------------------------------

func TestLintWatch_ValidV4(t *testing.T) {
	assertNoDiags(t, lint("version=4\nhttps://example.com/foo-(.+).tar.gz\n", debpkg.FileTypeWatch))
}

func TestLintWatch_ValidV5(t *testing.T) {
	assertNoDiags(t, lint("Version: 5\nhttps://example.com/foo-(.+).tar.gz\n", debpkg.FileTypeWatch))
}

func TestLintWatch_CommentsSkipped(t *testing.T) {
	text := "# This is a watch file\nversion=4\nhttps://example.com/foo-(.+).tar.gz\n"
	assertNoDiags(t, lint(text, debpkg.FileTypeWatch))
}

func TestLintWatch_UnknownVersion(t *testing.T) {
	assertHasDiag(t, lint("version=3\n", debpkg.FileTypeWatch), debpkg.SeverityWarning, "version")
}

func TestLintWatch_EmptyFile(t *testing.T) {
	assertHasDiag(t, lint("", debpkg.FileTypeWatch), debpkg.SeverityError, "version declaration")
}

func TestLintWatch_CommentOnlyFile(t *testing.T) {
	assertHasDiag(t, lint("# Only a comment\n", debpkg.FileTypeWatch), debpkg.SeverityError, "version declaration")
}

func TestLintWatch_MissingVersionDeclaration(t *testing.T) {
	// First non-comment line is a URL, not a version declaration.
	assertHasDiag(t, lint("https://example.com/foo-(.+).tar.gz\n", debpkg.FileTypeWatch), debpkg.SeverityError, "version declaration")
}

// ---------------------------------------------------------------------------
// Patch (DEP-3) lint tests
// ---------------------------------------------------------------------------

const goodPatch = `Description: Fix null pointer dereference in HTTP handler
Origin: upstream, https://github.com/example/repo/commit/abc123
Forwarded: https://github.com/example/repo/pull/1
Last-Update: 2024-01-15

--- a/src/http.c
+++ b/src/http.c
@@ -1 +1 @@
-bad
+good
`

func TestLintPatch_Valid(t *testing.T) {
	assertNoDiags(t, lint(goodPatch, debpkg.FileTypePatch))
}

func TestLintPatch_MissingDescriptionAndSubject(t *testing.T) {
	text := `Origin: upstream, https://github.com/example/repo
Forwarded: no
Last-Update: 2024-01-15

--- a/src/file.c
+++ b/src/file.c
`
	assertHasDiag(t, lint(text, debpkg.FileTypePatch), debpkg.SeverityError, "Description")
}

func TestLintPatch_SubjectAccepted(t *testing.T) {
	// "Subject:" is an accepted alias for "Description:".
	text := `Subject: Fix null pointer dereference
Origin: upstream, https://github.com/example/repo

--- a/src/file.c
+++ b/src/file.c
`
	assertNoDiagContaining(t, lint(text, debpkg.FileTypePatch), "Description")
}

func TestLintPatch_AuthorInsteadOfOrigin(t *testing.T) {
	// "Author:" satisfies the Origin requirement.
	text := `Description: Fix crash in handler
Author: Test User <test@example.com>
Forwarded: no

--- a/src/file.c
+++ b/src/file.c
`
	assertNoDiagContaining(t, lint(text, debpkg.FileTypePatch), "Origin")
}

func TestLintPatch_FromInsteadOfOrigin(t *testing.T) {
	// "From:" also satisfies the Origin requirement.
	text := `Description: Fix crash
From: Test User <test@example.com>

--- a/src/file.c
+++ b/src/file.c
`
	assertNoDiagContaining(t, lint(text, debpkg.FileTypePatch), "Origin")
}

func TestLintPatch_MissingOriginNoAuthor(t *testing.T) {
	// No Origin, Author, or From — must warn.
	text := `Description: Fix crash in handler
Forwarded: no

--- a/src/file.c
+++ b/src/file.c
`
	assertHasDiag(t, lint(text, debpkg.FileTypePatch), debpkg.SeverityWarning, "Origin")
}

func TestLintPatch_UnknownOriginKeyword(t *testing.T) {
	text := `Description: Fix crash
Origin: experimental, https://github.com/example/repo

--- a/src/file.c
+++ b/src/file.c
`
	assertHasDiag(t, lint(text, debpkg.FileTypePatch), debpkg.SeverityWarning, "unknown Origin keyword")
}

func TestLintPatch_ValidOriginKeywords(t *testing.T) {
	for _, kw := range []string{"upstream", "backport", "vendor", "other"} {
		text := "Description: Fix\nOrigin: " + kw + ", https://example.com\n\n--- a/f\n+++ b/f\n"
		assertNoDiagContaining(t, lint(text, debpkg.FileTypePatch), "unknown Origin keyword")
	}
}

func TestLintPatch_ForwardedNo(t *testing.T) {
	text := `Description: Fix crash
Origin: upstream, https://github.com/example/repo
Forwarded: no

--- a/src/file.c
+++ b/src/file.c
`
	assertNoDiagContaining(t, lint(text, debpkg.FileTypePatch), "Forwarded")
}

func TestLintPatch_ForwardedNotNeeded(t *testing.T) {
	text := `Description: Fix crash
Origin: upstream, https://github.com/example/repo
Forwarded: not-needed

--- a/src/file.c
+++ b/src/file.c
`
	assertNoDiagContaining(t, lint(text, debpkg.FileTypePatch), "Forwarded")
}

func TestLintPatch_ForwardedURL(t *testing.T) {
	text := `Description: Fix crash
Origin: upstream, https://github.com/example/repo
Forwarded: https://github.com/example/repo/pull/42

--- a/src/file.c
+++ b/src/file.c
`
	assertNoDiagContaining(t, lint(text, debpkg.FileTypePatch), "Forwarded")
}

func TestLintPatch_BadForwarded(t *testing.T) {
	text := `Description: Fix crash
Origin: upstream, https://github.com/example/repo
Forwarded: maybe

--- a/src/file.c
+++ b/src/file.c
`
	assertHasDiag(t, lint(text, debpkg.FileTypePatch), debpkg.SeverityWarning, "Forwarded")
}

func TestLintPatch_BadLastUpdate(t *testing.T) {
	text := `Description: Fix crash
Origin: upstream, https://github.com/example/repo
Last-Update: January 15, 2024

--- a/src/file.c
+++ b/src/file.c
`
	assertHasDiag(t, lint(text, debpkg.FileTypePatch), debpkg.SeverityWarning, "Last-Update")
}

func TestLintPatch_GoodLastUpdate(t *testing.T) {
	text := `Description: Fix crash
Origin: upstream, https://github.com/example/repo
Last-Update: 2024-01-15

--- a/src/file.c
+++ b/src/file.c
`
	assertNoDiagContaining(t, lint(text, debpkg.FileTypePatch), "Last-Update")
}

func TestLintPatch_LongDescriptionLine(t *testing.T) {
	longLine := " " + strings.Repeat("x", 81)
	text := "Description: Fix crash\n" + longLine + "\nOrigin: upstream, https://example.com\n\n--- a/f\n+++ b/f\n"
	assertHasDiag(t, lint(text, debpkg.FileTypePatch), debpkg.SeverityWarning, "description line")
}
