// SPDX-License-Identifier: GPL-3.0-or-later

package debpkg_test

import (
	"testing"

	"github.com/BAMF0/debpack-lsp/debpkg"
)

func TestLintRulesShebang(t *testing.T) {
	// Missing shebang.
	text := "%:\n\tdh $@\n"
	diags := debpkg.Lint(text, debpkg.FileTypeRules, debpkg.LintContext{})
	found := false
	for _, d := range diags {
		if d.Code == "rules-shebang" {
			found = true
		}
	}
	if !found {
		t.Error("expected rules-shebang diagnostic for missing shebang")
	}

	// Correct shebang — no shebang diagnostic.
	text = "#!/usr/bin/make -f\n%:\n\tdh $@\n"
	diags = debpkg.Lint(text, debpkg.FileTypeRules, debpkg.LintContext{})
	for _, d := range diags {
		if d.Code == "rules-shebang" {
			t.Error("did not expect rules-shebang diagnostic when shebang is correct")
		}
	}
}

func TestLintRulesMissingTarget(t *testing.T) {
	text := "#!/usr/bin/make -f\n\nbuild:\n\tdh $@\n"
	diags := debpkg.Lint(text, debpkg.FileTypeRules, debpkg.LintContext{})
	found := false
	for _, d := range diags {
		if d.Code == "rules-missing-target" {
			found = true
		}
	}
	if !found {
		t.Error("expected rules-missing-target diagnostic")
	}

	// With %: target — no diagnostic.
	text = "#!/usr/bin/make -f\n%:\n\tdh $@\n"
	diags = debpkg.Lint(text, debpkg.FileTypeRules, debpkg.LintContext{})
	for _, d := range diags {
		if d.Code == "rules-missing-target" {
			t.Error("did not expect rules-missing-target when %: is present")
		}
	}
}

func TestLintRulesDhDollarHint(t *testing.T) {
	// Using dh_* directly without dh $@ should produce a hint.
	text := "#!/usr/bin/make -f\n%:\n\tdh_install\n\tdh_installdocs\n"
	diags := debpkg.Lint(text, debpkg.FileTypeRules, debpkg.LintContext{})
	found := false
	for _, d := range diags {
		if d.Code == "rules-use-dh-dollar" {
			found = true
		}
	}
	if !found {
		t.Error("expected rules-use-dh-dollar hint when dh_* used without dh $@")
	}

	// With dh $@ — no hint.
	text = "#!/usr/bin/make -f\n%:\n\tdh $@\n"
	diags = debpkg.Lint(text, debpkg.FileTypeRules, debpkg.LintContext{})
	for _, d := range diags {
		if d.Code == "rules-use-dh-dollar" {
			t.Error("did not expect rules-use-dh-dollar when dh $@ is used")
		}
	}
}

func TestLintControlStanzaFieldPlacement(t *testing.T) {
	// Architecture in source stanza should be flagged.
	text := "Source: foo\nMaintainer: A <a@b.c>\nStandards-Version: 4.7.1\nArchitecture: any\n\nPackage: foo\nArchitecture: any\nDescription: test\n"
	diags := debpkg.Lint(text, debpkg.FileTypeControl, debpkg.LintContext{})
	found := false
	for _, d := range diags {
		if d.Code == "control-field-wrong-stanza" && d.Message != "" {
			found = true
		}
	}
	if !found {
		t.Error("expected control-field-wrong-stanza for Architecture in source stanza")
	}
}

func TestLintControlPackageNameValidity(t *testing.T) {
	// Invalid package name with uppercase.
	text := "Source: FooBar\nMaintainer: A <a@b.c>\nStandards-Version: 4.7.1\n\nPackage: FooBar\nArchitecture: any\nDescription: test\n"
	diags := debpkg.Lint(text, debpkg.FileTypeControl, debpkg.LintContext{})
	found := false
	for _, d := range diags {
		if d.Code == "control-invalid-package-name" {
			found = true
		}
	}
	if !found {
		t.Error("expected control-invalid-package-name for uppercase package name")
	}

	// Valid package name — no diagnostic.
	text = "Source: foo-bar\nMaintainer: A <a@b.c>\nStandards-Version: 4.7.1\n\nPackage: foo-bar\nArchitecture: any\nDescription: test\n"
	diags = debpkg.Lint(text, debpkg.FileTypeControl, debpkg.LintContext{})
	for _, d := range diags {
		if d.Code == "control-invalid-package-name" {
			t.Error("did not expect control-invalid-package-name for valid name")
		}
	}
}

func TestLintControlURLFields(t *testing.T) {
	text := "Source: foo\nMaintainer: A <a@b.c>\nStandards-Version: 4.7.1\nHomepage: ftp://example.com\n\nPackage: foo\nArchitecture: any\nDescription: test\n"
	diags := debpkg.Lint(text, debpkg.FileTypeControl, debpkg.LintContext{})
	found := false
	for _, d := range diags {
		if d.Code == "control-invalid-url" {
			found = true
		}
	}
	if !found {
		t.Error("expected control-invalid-url for non-http Homepage")
	}
}

func TestLintControlStandardsVersionOutdated(t *testing.T) {
	text := "Source: foo\nMaintainer: A <a@b.c>\nStandards-Version: 3.9.8\n\nPackage: foo\nArchitecture: any\nDescription: test\n"
	diags := debpkg.Lint(text, debpkg.FileTypeControl, debpkg.LintContext{})
	found := false
	for _, d := range diags {
		if d.Code == "control-standards-version-outdated" {
			found = true
		}
	}
	if !found {
		t.Error("expected control-standards-version-outdated for 3.9.8")
	}

	// Current version — no diagnostic.
	text = "Source: foo\nMaintainer: A <a@b.c>\nStandards-Version: 4.7.1\n\nPackage: foo\nArchitecture: any\nDescription: test\n"
	diags = debpkg.Lint(text, debpkg.FileTypeControl, debpkg.LintContext{})
	for _, d := range diags {
		if d.Code == "control-standards-version-outdated" {
			t.Error("did not expect control-standards-version-outdated for 4.7.1")
		}
	}
}

func TestLintCopyrightUnknownLicense(t *testing.T) {
	text := "Format: https://www.debian.org/doc/packaging-manuals/copyright-format/1.0/\n\nFiles: *\nCopyright: 2024 A <a@b.c>\nLicense: BogoLicense\n"
	diags := debpkg.Lint(text, debpkg.FileTypeCopyright, debpkg.LintContext{})
	found := false
	for _, d := range diags {
		if d.Code == "dep5-unknown-license" {
			found = true
		}
	}
	if !found {
		t.Error("expected dep5-unknown-license for BogoLicense")
	}

	// Known license — no diagnostic.
	text = "Format: https://www.debian.org/doc/packaging-manuals/copyright-format/1.0/\n\nFiles: *\nCopyright: 2024 A <a@b.c>\nLicense: GPL-2+\n"
	diags = debpkg.Lint(text, debpkg.FileTypeCopyright, debpkg.LintContext{})
	for _, d := range diags {
		if d.Code == "dep5-unknown-license" {
			t.Error("did not expect dep5-unknown-license for GPL-2+")
		}
	}

	// LicenseRef-* is always valid.
	text = "Format: https://www.debian.org/doc/packaging-manuals/copyright-format/1.0/\n\nFiles: *\nCopyright: 2024 A <a@b.c>\nLicense: LicenseRef-Custom\n"
	diags = debpkg.Lint(text, debpkg.FileTypeCopyright, debpkg.LintContext{})
	for _, d := range diags {
		if d.Code == "dep5-unknown-license" {
			t.Error("did not expect dep5-unknown-license for LicenseRef-Custom")
		}
	}
}

func TestLintInstallTwoFields(t *testing.T) {
	// *.install with correct 2-field lines — no field-count diagnostics.
	text := "src/foo usr/bin\nsrc/bar usr/share/doc\n"
	diags := debpkg.Lint(text, debpkg.FileTypeInstall, debpkg.LintContext{})
	for _, d := range diags {
		if d.Code == "install-field-count" {
			t.Errorf("unexpected install-field-count for valid line: %s", d.Message)
		}
	}

	// *.install with a 1-field line — should warn.
	text = "src/foo\n"
	diags = debpkg.Lint(text, debpkg.FileTypeInstall, debpkg.LintContext{})
	found := false
	for _, d := range diags {
		if d.Code == "install-field-count" {
			found = true
		}
	}
	if !found {
		t.Error("expected install-field-count for 1-field line in *.install")
	}

	// *.install with a 3-field line — should warn.
	text = "src/foo usr/bin extra\n"
	diags = debpkg.Lint(text, debpkg.FileTypeInstall, debpkg.LintContext{})
	found = false
	for _, d := range diags {
		if d.Code == "install-field-count" {
			found = true
		}
	}
	if !found {
		t.Error("expected install-field-count for 3-field line in *.install")
	}
}

func TestLintInstallLinksTwoFields(t *testing.T) {
	// *.links with correct 2-field lines — no diagnostic.
	text := "src/foo usr/bin/foo\n"
	diags := debpkg.Lint(text, debpkg.FileTypeLinks, debpkg.LintContext{})
	for _, d := range diags {
		if d.Code == "install-field-count" {
			t.Errorf("unexpected install-field-count for valid .links line: %s", d.Message)
		}
	}

	// *.links with 1-field line — should warn.
	text = "src/foo\n"
	diags = debpkg.Lint(text, debpkg.FileTypeLinks, debpkg.LintContext{})
	found := false
	for _, d := range diags {
		if d.Code == "install-field-count" {
			found = true
		}
	}
	if !found {
		t.Error("expected install-field-count for 1-field line in *.links")
	}
}

func TestLintDirsSinglePath(t *testing.T) {
	// *.dirs with correct single-path lines — no diagnostic.
	text := "usr/bin\nusr/share/doc\n"
	diags := debpkg.Lint(text, debpkg.FileTypeDirs, debpkg.LintContext{})
	for _, d := range diags {
		if d.Code == "install-single-path" {
			t.Errorf("unexpected install-single-path for valid .dirs line: %s", d.Message)
		}
	}

	// *.dirs with 2-field line — should warn.
	text = "usr/bin extra\n"
	diags = debpkg.Lint(text, debpkg.FileTypeDirs, debpkg.LintContext{})
	found := false
	for _, d := range diags {
		if d.Code == "install-single-path" {
			found = true
		}
	}
	if !found {
		t.Error("expected install-single-path for 2-field line in *.dirs")
	}
}

func TestLintInstallEmptyFile(t *testing.T) {
	text := "\n\n\n"
	diags := debpkg.Lint(text, debpkg.FileTypeInstall, debpkg.LintContext{})
	found := false
	for _, d := range diags {
		if d.Code == "install-empty" {
			found = true
		}
	}
	if !found {
		t.Error("expected install-empty for empty file")
	}
}

func TestLintInstallCommentsIgnored(t *testing.T) {
	// Comments should not trigger field-count warnings.
	text := "# This is a comment\nsrc/foo usr/bin\n"
	diags := debpkg.Lint(text, debpkg.FileTypeInstall, debpkg.LintContext{})
	for _, d := range diags {
		if d.Code == "install-field-count" {
			t.Errorf("unexpected install-field-count for commented line: %s", d.Message)
		}
	}
}

func TestFileTypeString(t *testing.T) {
	cases := []struct {
		ft   debpkg.FileType
		want string
	}{
		{debpkg.FileTypeControl, "control"},
		{debpkg.FileTypeChangelog, "changelog"},
		{debpkg.FileTypeRules, "rules"},
		{debpkg.FileTypeWatch, "watch"},
		{debpkg.FileTypeCopyright, "copyright"},
		{debpkg.FileTypePatch, "patch"},
		{debpkg.FileTypeInstall, "install"},
		{debpkg.FileTypeDirs, "dirs"},
		{debpkg.FileTypeDocs, "docs"},
		{debpkg.FileTypeLinks, "links"},
		{debpkg.FileTypeManpages, "manpages"},
		{debpkg.FileTypeUnknown, "unknown"},
	}
	for _, tc := range cases {
		if got := tc.ft.String(); got != tc.want {
			t.Errorf("FileType(%d).String() = %q, want %q", tc.ft, got, tc.want)
		}
	}
}
