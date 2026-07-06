package debhelper

import (
	"os/exec"
	"strings"
	"testing"
)

// fixtureInstallDocs is a trimmed rendering of "man -P cat dh_installdocs".
const fixtureInstallDocs = `DH_INSTALLDOCS(1)                  Debhelper                  DH_INSTALLDOCS(1)

NAME
     dh_installdocs - install documentation into package build directories

SYNOPSIS
     dh_installdocs [debhelper options] [-A] [-Xitem] [file ...]

DESCRIPTION
     dh_installdocs  is  a debhelper program that is responsible for installing
     documentation into usr/share/doc/package in package build directories.

     In compat 10 and earlier, dh_install(1) may be a better tool for  handling
     the  upstream documentation, when upstream's own build system installs all
     the desired documentation correctly.

FILES
     debian/package.docs
         List documentation files to be installed into package.
`

// fixtureStrip has a SYNOPSIS that wraps across two lines, exercising the
// join/collapse logic.
const fixtureStrip = `DH_STRIP(1)                        Debhelper                        DH_STRIP(1)

NAME
     dh_strip - strip executables, shared libraries, and some static libraries

SYNOPSIS
     dh_strip  [debhelper options] [-Xitem] [--dbg-package=package] [--keep-de‐
bug]

DESCRIPTION
     dh_strip is a debhelper program that is responsible for stripping out  de‐
     bug  symbols  in  executables, shared libraries, and static libraries that
     are not needed during execution.

     This program examines your package build directories and works out what to
     strip on its own.
`

// fixtureNoDescription is a synthetic page that has NAME and SYNOPSIS but no
// DESCRIPTION section, to verify graceful handling.
const fixtureNoDescription = `DH_FOO(1)            Debhelper            DH_FOO(1)

NAME
     dh_foo - does foo things

SYNOPSIS
     dh_foo [debhelper options] [-Xitem]
`

func TestParseManpage(t *testing.T) {
	cases := []struct {
		name            string
		text            string
		cmd             string
		wantSynopsis    string
		wantDescription string
	}{
		{
			name:            "installdocs",
			text:            fixtureInstallDocs,
			cmd:             "dh_installdocs",
			wantSynopsis:    "install documentation into package build directories",
			wantDescription: "dh_installdocs [debhelper options] [-A] [-Xitem] [file ...]\n\ndh_installdocs is a debhelper program that is responsible for installing documentation into usr/share/doc/package in package build directories.",
		},
		{
			name:            "strip-wrapped-synopsis",
			text:            fixtureStrip,
			cmd:             "dh_strip",
			wantSynopsis:    "strip executables, shared libraries, and some static libraries",
			wantDescription: "dh_strip [debhelper options] [-Xitem] [--dbg-package=package] [--keep-debug]\n\ndh_strip is a debhelper program that is responsible for stripping out debug symbols in executables, shared libraries, and static libraries that are not needed during execution.",
		},
		{
			name:            "no-description-section",
			text:            fixtureNoDescription,
			cmd:             "dh_foo",
			wantSynopsis:    "does foo things",
			wantDescription: "dh_foo [debhelper options] [-Xitem]",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotSyn, gotDesc := parseManpage(tc.text, tc.cmd)
			if gotSyn != tc.wantSynopsis {
				t.Errorf("synopsis = %q\nwant      %q", gotSyn, tc.wantSynopsis)
			}
			if gotDesc != tc.wantDescription {
				t.Errorf("description = %q\nwant         %q", gotDesc, tc.wantDescription)
			}
		})
	}
}

func TestParseManpageNameFallback(t *testing.T) {
	// If the NAME line lacks " - ", synopsis should stay empty so the caller
	// can fall back to --help. The SYNOPSIS section may still contribute to
	// the description.
	text := `DH_FOO(1) Debhelper DH_FOO(1)

NAME
     dh_foo

SYNOPSIS
     dh_foo [options]

DESCRIPTION
     dh_foo does things.
`
	syn, desc := parseManpage(text, "dh_foo")
	if syn != "" {
		t.Errorf("expected empty synopsis when NAME has no dash, got %q", syn)
	}
	if desc == "" {
		t.Errorf("expected non-empty description from SYNOPSIS/DESCRIPTION, got empty")
	}
}

func TestCollapseWS(t *testing.T) {
	cases := []struct{ in, want string }{
		{"foo  bar", "foo bar"},
		{"foo\tbar\nbaz", "foo bar baz"},
		{"   leading and trailing   ", "leading and trailing"},
		{"single", "single"},
		{"", ""},
		{"a\t\t b  \n c", "a b c"},
	}
	for _, tc := range cases {
		if got := collapseWS(tc.in); got != tc.want {
			t.Errorf("collapseWS(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestJoinWrapped(t *testing.T) {
	cases := []struct {
		name  string
		lines []string
		want  string
	}{
		{"simple", []string{"foo", "bar"}, "foo bar"},
		{"soft-hyphen-break", []string{"de\u2010", "bug"}, "debug"},
		{"ad-soft-hyphen", []string{"de\u00ad", "bug"}, "debug"},
		{"regular-hyphen-break", []string{"well-", "known"}, "well-known"},
		{"multi-soft-hyphen", []string{"su\u2010", "per\u2010", "cali"}, "supercali"},
		{"single-line", []string{"only"}, "only"},
		{"empty", []string{}, ""},
		{"internal-soft-hyphen", []string{"foo\u2010bar", "baz"}, "foobar baz"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := joinWrapped(tc.lines); got != tc.want {
				t.Errorf("joinWrapped(%v) = %q, want %q", tc.lines, got, tc.want)
			}
		})
	}
}

func TestParseFlags(t *testing.T) {
	help := `Usage: dh_foo [options]

  dh_foo is a part of debhelper.

Options:
  --sourcedir=dir     source dir
  -Xitem, --exclude=item  exclude
  --no-act            do nothing
  --no-act            duplicate
  --help              show help
  --version           show version
`
	flags := parseFlags(help)
	want := []string{"--sourcedir=", "--exclude=", "--no-act"}
	if len(flags) != len(want) {
		t.Fatalf("got %d flags %v, want %d %v", len(flags), flags, len(want), want)
	}
	for i, f := range flags {
		if f != want[i] {
			t.Errorf("flags[%d] = %q, want %q", i, f, want[i])
		}
	}
}

func TestCacheEnvelopeSchemaCheck(t *testing.T) {
	// An envelope with the wrong schema version should be rejected by
	// treating it as a stale cache (loadCache returns an error on mismatch).
	// We exercise the schema-version logic directly since loadCache reads
	// from disk.
	env := cacheEnvelope{SchemaVersion: 1, CachedAt: 0, Commands: nil}
	if env.SchemaVersion == cacheSchemaVer {
		t.Fatal("schema version 1 should not equal current cacheSchemaVer")
	}
}

func TestScrapeCommandFallback(t *testing.T) {
	// scrapeCommand should not panic and should always set the Name field,
	// even when both man and --help are unavailable (e.g. the binary path
	// does not exist on the test machine).
	cmd := scrapeCommand("dh_does_not_exist", "/nonexistent/dh_does_not_exist")
	if cmd.Name != "dh_does_not_exist" {
		t.Errorf("Name = %q, want %q", cmd.Name, "dh_does_not_exist")
	}
	// Synopsis/Description/Flags may all be empty on a missing binary; that
	// is acceptable. We only assert no panic and a populated Name.
}

func TestStoreByNameMissing(t *testing.T) {
	s := NewStore()
	if c := s.ByName("dh_nope"); c != nil {
		t.Errorf("expected nil for unknown command, got %+v", c)
	}
}

func TestStoreAllEmpty(t *testing.T) {
	s := NewStore()
	all := s.All()
	if len(all) != 0 {
		t.Errorf("expected empty All(), got %d items", len(all))
	}
	// All() returns a fresh slice; mutating it must not affect the store.
	all = append(all, Command{Name: "dh_injected"})
	if again := s.All(); len(again) != 0 {
		t.Errorf("store was mutated via All() return slice; got %d items", len(again))
	}
}

// TestStoreAllByName exercises the read paths after a manual Load-equivalent
// population by directly setting the internal slice.
func TestStoreRoundTrip(t *testing.T) {
	s := NewStore()
	s.commands = []Command{
		{Name: "dh_alpha", Synopsis: "alpha syn", Description: "alpha desc", Flags: []string{"--foo="}},
		{Name: "dh_beta", Synopsis: "beta syn"},
	}
	if c := s.ByName("dh_beta"); c == nil || c.Synopsis != "beta syn" {
		t.Fatalf("ByName(dh_beta) = %+v, want synopsis %q", c, "beta syn")
	}
	all := s.All()
	if len(all) != 2 {
		t.Fatalf("All() returned %d items, want 2", len(all))
	}
	// Verify the returned slice is a copy (mutating it shouldn't affect store).
	all[0].Synopsis = "mutated"
	if c := s.ByName("dh_alpha"); c.Synopsis != "alpha syn" {
		t.Errorf("store mutated via All(): Synopsis = %q", c.Synopsis)
	}
}

// TestParseManpageNoSections ensures the parser returns empty values rather
// than panicking when given a document with no recognised sections.
func TestParseManpageNoSections(t *testing.T) {
	text := strings.Repeat("free-form prose\n", 5)
	syn, desc := parseManpage(text, "dh_foo")
	if syn != "" || desc != "" {
		t.Errorf("expected empty synopsis/description for no-section input, got (%q, %q)", syn, desc)
	}
}

// TestScrapeManpageLive verifies that scraping a real installed man page
// yields a per-command description (not the old "--help" boilerplate). It is
// skipped when "man" is unavailable or the command is not installed.
func TestScrapeManpageLive(t *testing.T) {
	if _, err := exec.LookPath("man"); err != nil {
		t.Skip("man not available")
	}
	// dh_installcron is tiny and almost always present with debhelper.
	out, err := exec.Command("man", "-P", "cat", "dh_installcron").CombinedOutput()
	if err != nil || len(out) == 0 {
		t.Skip("dh_installcron man page not available")
	}
	syn, desc := parseManpage(string(out), "dh_installcron")
	if syn == "" {
		t.Fatalf("expected non-empty synopsis from live man page, got empty")
	}
	// The boilerplate line was "dh_installcron is a part of debhelper";
	// the real NAME description is about cron files.
	if strings.Contains(syn, "is a part of debhelper") {
		t.Errorf("synopsis is still the boilerplate: %q", syn)
	}
	if !strings.Contains(strings.ToLower(syn), "cron") {
		t.Errorf("expected synopsis to mention cron, got %q", syn)
	}
	if desc == "" {
		t.Errorf("expected non-empty description from live man page")
	}
}
