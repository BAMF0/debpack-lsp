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
