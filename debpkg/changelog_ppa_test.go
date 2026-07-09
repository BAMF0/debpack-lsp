// SPDX-License-Identifier: GPL-3.0-or-later

package debpkg_test

import (
	"testing"

	"github.com/BAMF0/debpack-lsp/debpkg"
)

func TestPPASuffix(t *testing.T) {
	cases := []struct {
		version string
		want    string
	}{
		{"0.2.14-1ubuntu1", ""},
		{"0.2.14-1ubuntu1~ppa1", "~ppa1"},
		{"0.2.14-1ubuntu1~ppa10", "~ppa10"},
		{"0.2.14-1ubuntu1~ubuntu1", ""},
		{"1.0-1", ""},
		{"~ppa3", "~ppa3"},
	}
	for _, c := range cases {
		got := debpkg.PPASuffix(c.version)
		if got != c.want {
			t.Errorf("PPASuffix(%q) = %q, want %q", c.version, got, c.want)
		}
	}
}

func TestNextPPAVersion(t *testing.T) {
	cases := []struct {
		version string
		want    string
	}{
		{"0.2.14-1ubuntu1", "0.2.14-1ubuntu1~ppa1"},
		{"0.2.14-1ubuntu1~ppa1", "0.2.14-1ubuntu1~ppa2"},
		{"0.2.14-1ubuntu1~ppa9", "0.2.14-1ubuntu1~ppa10"},
		{"0.2.14-1ubuntu1~ppa99", "0.2.14-1ubuntu1~ppa100"},
		{"0.2.14-1ubuntu1~ubuntu1", "0.2.14-1ubuntu1~ubuntu1~ppa1"},
		{"1.0-1", "1.0-1~ppa1"},
	}
	for _, c := range cases {
		got := debpkg.NextPPAVersion(c.version)
		if got != c.want {
			t.Errorf("NextPPAVersion(%q) = %q, want %q", c.version, got, c.want)
		}
		// Idempotency-ish: bumping again increments the ppa number.
		if debpkg.PPASuffix(got) == "" {
			t.Errorf("NextPPAVersion(%q) result %q has no ppa suffix", c.version, got)
		}
	}
}

func TestChangelogVersionSpan(t *testing.T) {
	cases := []struct {
		line               string
		wantVersion        string
		wantStart, wantEnd int
		ok                 bool
	}{
		{"rust-sudo-rs (0.2.14-1ubuntu1) stonking; urgency=medium", "0.2.14-1ubuntu1", 14, 29, true},
		{"rust-sudo-rs (0.2.14-1ubuntu1~ppa1) stonking; urgency=medium", "0.2.14-1ubuntu1~ppa1", 14, 34, true},
		{"foo (1.0-1) unstable; urgency=low", "1.0-1", 5, 10, true},
		{"  * just a bullet", "", 0, 0, false},
		{"not a header", "", 0, 0, false},
		{"", "", 0, 0, false},
	}
	for _, c := range cases {
		ver, start, end, ok := debpkg.ChangelogVersionSpan(c.line)
		if ok != c.ok {
			t.Errorf("ChangelogVersionSpan(%q) ok = %v, want %v", c.line, ok, c.ok)
			continue
		}
		if !ok {
			continue
		}
		if ver != c.wantVersion {
			t.Errorf("ChangelogVersionSpan(%q) version = %q, want %q", c.line, ver, c.wantVersion)
		}
		if start != c.wantStart || end != c.wantEnd {
			t.Errorf("ChangelogVersionSpan(%q) span = (%d, %d), want (%d, %d)", c.line, start, end, c.wantStart, c.wantEnd)
		}
		// Sanity: the substring at [start:end] must equal the version.
		if c.line[start:end] != c.wantVersion {
			t.Errorf("ChangelogVersionSpan(%q) substring = %q, want %q", c.line, c.line[start:end], c.wantVersion)
		}
	}
}
