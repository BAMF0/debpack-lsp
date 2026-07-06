// SPDX-License-Identifier: GPL-3.0-or-later

package debpkg_test

import (
	"testing"

	"github.com/BAMF0/debpack-lsp/debpkg"
)

func TestHasDep3Header(t *testing.T) {
	cases := []struct {
		name string
		text string
		want bool
	}{
		{
			name: "empty document",
			text: "",
			want: false,
		},
		{
			name: "single field present",
			text: "Description: Fix crash\n",
			want: true,
		},
		{
			name: "multiple fields present",
			text: "Description: Fix crash\nOrigin: upstream, https://example.com\n",
			want: true,
		},
		{
			name: "fields below diff marker are ignored",
			text: "--- a/foo.c\n+++ b/foo.c\nDescription: this should not count\n",
			want: false,
		},
		{
			name: "free-form prose without fields",
			text: "This patch fixes a crash in the frobnicator.\nNo headers here.\n",
			want: false,
		},
		{
			name: "subject alias counts as header",
			text: "Subject: Fix crash\n",
			want: true,
		},
		{
			name: "bug-vendor field counts",
			text: "Bug-Debian: https://bugs.debian.org/123456\n",
			want: true,
		},
		{
			name: "unrecognised field does not count",
			text: "Foo-Bar: not a dep3 field\n",
			want: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := debpkg.HasDep3Header(tc.text)
			if got != tc.want {
				t.Errorf("HasDep3Header(%q) = %v, want %v", tc.text, got, tc.want)
			}
		})
	}
}
