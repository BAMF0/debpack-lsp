package server

import "testing"

func TestColToByteOffset(t *testing.T) {
	cases := []struct {
		line string
		col  int
		want int
	}{
		{"hello", 0, 0},
		{"hello", 3, 3},
		{"hello", 5, 5},  // at end
		{"hello", 10, 5}, // beyond end -> full length
		{"hello", -1, 0}, // negative -> 0
		// Multi-byte: "café" has 4 runes but 5 bytes (é is 2 bytes).
		{"café", 0, 0},
		{"café", 1, 1},
		{"café", 3, 3},  // at 'é' rune start
		{"café", 4, 5},  // past 'é' -> byte offset 5
		{"café", 10, 5}, // beyond -> full length
		// Empty line.
		{"", 0, 0},
		{"", 5, 0},
	}
	for _, tc := range cases {
		got := colToByteOffset(tc.line, tc.col)
		if got != tc.want {
			t.Errorf("colToByteOffset(%q, %d) = %d, want %d", tc.line, tc.col, got, tc.want)
		}
	}
}

func TestColToByteOffsetMultiByteSlice(t *testing.T) {
	// Slicing a line at the byte offset returned by colToByteOffset must
	// not split a multi-byte rune.
	line := "café world"
	// col=3 points at 'é' (rune 3). Byte offset should be 3 (start of é).
	off := colToByteOffset(line, 3)
	if off != 3 {
		t.Fatalf("col 3 -> byte %d, want 3", off)
	}
	// Slicing [0:off] must be a valid UTF-8 string ("caf").
	prefix := line[:off]
	if prefix != "caf" {
		t.Errorf("prefix = %q, want %q", prefix, "caf")
	}
	// col=4 points past 'é'. Byte offset should be 5 (after the 2-byte é).
	off = colToByteOffset(line, 4)
	if off != 5 {
		t.Fatalf("col 4 -> byte %d, want 5", off)
	}
	prefix = line[:off]
	if prefix != "café" {
		t.Errorf("prefix = %q, want %q", prefix, "café")
	}
}

func TestWordAtColMultiByte(t *testing.T) {
	// Multi-byte content before an ASCII word must not break word extraction.
	// "café dh_install" — 'é' is 2 bytes, so col 6 (the 'd') maps to byte 7.
	line := "café dh_install"
	// col 6 is at 'd'. The word should be "dh_install".
	got := wordAtCol(line, 6)
	if got != "dh_install" {
		t.Errorf("wordAtCol(col=6) = %q, want %q", got, "dh_install")
	}
	// col 9 is inside "dh_install" (at 'n'). Same word.
	got = wordAtCol(line, 9)
	if got != "dh_install" {
		t.Errorf("wordAtCol(col=9) = %q, want %q", got, "dh_install")
	}
}

func TestLineColToByteOffsetMultiLine(t *testing.T) {
	text := "line1\ncafé\nline3\n"
	// Line 1, col 3 -> inside 'é' on the "café" line. Byte offset of line
	// 1 start is 6 (after "line1\n"). col 3 -> byte offset 3 within "café"
	// = 6+3 = 9.
	got := lineColToByteOffset(text, 1, 3)
	if got != 9 {
		t.Errorf("lineColToByteOffset(1,3) = %d, want 9", got)
	}
	// Line 1, col 4 -> past 'é', byte offset 6+5 = 11.
	got = lineColToByteOffset(text, 1, 4)
	if got != 11 {
		t.Errorf("lineColToByteOffset(1,4) = %d, want 11", got)
	}
}
