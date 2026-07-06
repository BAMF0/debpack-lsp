package debpkg_test

import (
	"testing"

	"github.com/BAMF0/debpack-lsp/debpkg"
)

func TestFileTypeFromURI(t *testing.T) {
	cases := []struct {
		uri  string
		want debpkg.FileType
	}{
		{"file:///home/user/pkg/debian/control", debpkg.FileTypeControl},
		{"file:///home/user/pkg/debian/changelog", debpkg.FileTypeChangelog},
		{"file:///home/user/pkg/debian/rules", debpkg.FileTypeRules},
		{"file:///home/user/pkg/debian/watch", debpkg.FileTypeWatch},
		{"file:///home/user/pkg/debian/copyright", debpkg.FileTypeCopyright},
		{"file:///home/user/pkg/debian/patches/fix-crash.patch", debpkg.FileTypePatch},
		{"file:///home/user/pkg/debian/patches/series", debpkg.FileTypeUnknown},
		{"file:///home/user/pkg/debian/patches/ubuntu/fix.patch", debpkg.FileTypePatch},
		{"file:///home/user/pkg/debian/patches/ubuntu/series", debpkg.FileTypeUnknown},
		{"file:///home/user/pkg/debian/curl.install", debpkg.FileTypeInstall},
		{"file:///home/user/pkg/debian/curl.dirs", debpkg.FileTypeDirs},
		{"file:///home/user/pkg/debian/curl.docs", debpkg.FileTypeDocs},
		{"file:///home/user/pkg/debian/curl.links", debpkg.FileTypeLinks},
		{"file:///home/user/pkg/debian/curl.manpages", debpkg.FileTypeManpages},
		{"file:///home/user/pkg/src/main.go", debpkg.FileTypeUnknown},
	}
	for _, tc := range cases {
		got := debpkg.FileTypeFromURI(tc.uri)
		if got != tc.want {
			t.Errorf("FileTypeFromURI(%q) = %v, want %v", tc.uri, got, tc.want)
		}
	}
}

func TestPackageFromChangelog(t *testing.T) {
	cases := []struct {
		text string
		want string
	}{
		{"curl (7.88.1-2ubuntu1) noble; urgency=medium\n\n  * Fix CVE\n", "curl"},
		{"golang-1.21 (1.21.0-1) unstable; urgency=low\n", "golang-1.21"},
		{"", ""},
		{"not a changelog\n", ""},
	}
	for _, tc := range cases {
		got := debpkg.PackageFromChangelog(tc.text)
		if got != tc.want {
			t.Errorf("PackageFromChangelog(%q) = %q, want %q", tc.text, got, tc.want)
		}
	}
}

func TestBugRefAtCursor(t *testing.T) {
	cases := []struct {
		line       string
		wantPrefix string
		wantDigits string
	}{
		{"  * Fix crash (LP: #", "LP: #", ""},
		{"  * Fix crash (LP: #2045", "LP: #", "2045"},
		{"  * Fix crash (LP: #2045432)", "", ""}, // cursor past closing paren
		// "Closes: #" is the Debian BTS convention; completion is disabled
		// until a BTS backend exists, so BugRefAtCursor returns "" for it.
		{"  * Closes: #12345", "", ""},
		{"  * closes: #999", "", ""},
		{"  * unrelated text", "", ""},
	}
	for _, tc := range cases {
		gotPfx, gotDig := debpkg.BugRefAtCursor(tc.line)
		if gotPfx != tc.wantPrefix || gotDig != tc.wantDigits {
			t.Errorf("BugRefAtCursor(%q) = (%q, %q), want (%q, %q)",
				tc.line, gotPfx, gotDig, tc.wantPrefix, tc.wantDigits)
		}
	}
}

func TestClosesRefAtCursor(t *testing.T) {
	cases := []struct {
		line       string
		wantPrefix string
		wantDigits string
	}{
		{"  * Closes: #", "Closes: #", ""},
		{"  * Closes: #12345", "Closes: #", "12345"},
		{"  * closes: #999", "closes: #", "999"},
		{"  * LP: #2045", "", ""}, // not a Closes ref
		{"  * unrelated text", "", ""},
	}
	for _, tc := range cases {
		gotPfx, gotDig := debpkg.ClosesRefAtCursor(tc.line)
		if gotPfx != tc.wantPrefix || gotDig != tc.wantDigits {
			t.Errorf("ClosesRefAtCursor(%q) = (%q, %q), want (%q, %q)",
				tc.line, gotPfx, gotDig, tc.wantPrefix, tc.wantDigits)
		}
	}
}

func TestClosesRefAtOffset(t *testing.T) {
	text := "  * Fix crash (Closes: #12345) and also (LP: #999)"
	cases := []struct {
		offset int
		wantID string
	}{
		{22, "12345"}, // inside #12345
		{25, "12345"},
		{0, ""},  // before the ref
		{45, ""}, // inside LP: #999, not a Closes ref
	}
	for _, tc := range cases {
		_, gotID := debpkg.ClosesRefAtOffset(text, tc.offset)
		if gotID != tc.wantID {
			t.Errorf("ClosesRefAtOffset(offset=%d) = %q, want %q", tc.offset, gotID, tc.wantID)
		}
	}
}

func TestBugNumberAtOffset(t *testing.T) {
	text := "  * Fix crash (LP: #2045432) and also (LP: #999)"
	cases := []struct {
		offset int
		want   string
	}{
		{20, "2045432"}, // inside #2045432
		{23, "2045432"},
		{40, "999"},     // inside #999
		{10, ""},        // in "Fix crash"
	}
	for _, tc := range cases {
		got := debpkg.BugNumberAtOffset(text, tc.offset)
		if got != tc.want {
			t.Errorf("BugNumberAtOffset(offset=%d) = %q, want %q", tc.offset, got, tc.want)
		}
	}
}
