package debian_test

import (
	"testing"

	"github.com/yourusername/debpack-lsp/debian"
)

func TestFileTypeFromURI(t *testing.T) {
	cases := []struct {
		uri  string
		want debian.FileType
	}{
		{"file:///home/user/pkg/debian/control", debian.FileTypeControl},
		{"file:///home/user/pkg/debian/changelog", debian.FileTypeChangelog},
		{"file:///home/user/pkg/debian/rules", debian.FileTypeRules},
		{"file:///home/user/pkg/debian/watch", debian.FileTypeWatch},
		{"file:///home/user/pkg/debian/copyright", debian.FileTypeCopyright},
		{"file:///home/user/pkg/debian/patches/fix-crash.patch", debian.FileTypePatch},
		{"file:///home/user/pkg/debian/curl.install", debian.FileTypeInstall},
		{"file:///home/user/pkg/debian/curl.dirs", debian.FileTypeDirs},
		{"file:///home/user/pkg/debian/curl.docs", debian.FileTypeDocs},
		{"file:///home/user/pkg/debian/curl.links", debian.FileTypeLinks},
		{"file:///home/user/pkg/debian/curl.manpages", debian.FileTypeManpages},
		{"file:///home/user/pkg/src/main.go", debian.FileTypeUnknown},
	}
	for _, tc := range cases {
		got := debian.FileTypeFromURI(tc.uri)
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
		got := debian.PackageFromChangelog(tc.text)
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
		{"  * Fix crash (LP: #2045432)", "", ""},          // cursor past closing paren
		{"  * Closes: #12345", "Closes: #", "12345"},
		{"  * closes: #999", "closes: #", "999"},
		{"  * unrelated text", "", ""},
	}
	for _, tc := range cases {
		gotPfx, gotDig := debian.BugRefAtCursor(tc.line)
		if gotPfx != tc.wantPrefix || gotDig != tc.wantDigits {
			t.Errorf("BugRefAtCursor(%q) = (%q, %q), want (%q, %q)",
				tc.line, gotPfx, gotDig, tc.wantPrefix, tc.wantDigits)
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
		got := debian.BugNumberAtOffset(text, tc.offset)
		if got != tc.want {
			t.Errorf("BugNumberAtOffset(offset=%d) = %q, want %q", tc.offset, got, tc.want)
		}
	}
}
