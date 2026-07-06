// SPDX-License-Identifier: GPL-3.0-or-later

package server

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWorkspaceCacheRead(t *testing.T) {
	tmp := t.TempDir()
	debDir := filepath.Join(tmp, "debian")
	if err := os.MkdirAll(filepath.Join(debDir, "source"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(debDir, "compat"), []byte("13\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(debDir, "source", "format"), []byte("3.0 (quilt)\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	w := newWorkspaceCache(tmp)

	// Read compat.
	content, ok := w.read("compat")
	if !ok || content != "13\n" {
		t.Errorf("read(compat) = (%q, %v), want (%q, true)", content, ok, "13\n")
	}

	// Read source/format.
	content, ok = w.read("source/format")
	if !ok || content != "3.0 (quilt)\n" {
		t.Errorf("read(source/format) = (%q, %v), want (%q, true)", content, ok, "3.0 (quilt)\n")
	}

	// Read missing file.
	_, ok = w.read("nonexistent")
	if ok {
		t.Error("expected ok=false for missing file")
	}

	// Cached read returns same content.
	content2, ok := w.read("compat")
	if !ok || content2 != "13\n" {
		t.Errorf("cached read(compat) = (%q, %v)", content2, ok)
	}
}

func TestWorkspaceCacheInvalidate(t *testing.T) {
	tmp := t.TempDir()
	debDir := filepath.Join(tmp, "debian")
	if err := os.MkdirAll(debDir, 0o755); err != nil {
		t.Fatal(err)
	}
	compatPath := filepath.Join(debDir, "compat")
	if err := os.WriteFile(compatPath, []byte("13\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	w := newWorkspaceCache(tmp)
	content, _ := w.read("compat")
	if content != "13\n" {
		t.Fatalf("initial read = %q", content)
	}

	// Change the file and invalidate.
	if err := os.WriteFile(compatPath, []byte("14\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	w.invalidate("compat")
	content, _ = w.read("compat")
	if content != "14\n" {
		t.Errorf("after invalidate, read = %q, want %q", content, "14\n")
	}
}

func TestExtractDebhelperCompatVersion(t *testing.T) {
	cases := []struct {
		text string
		want string
	}{
		{"Source: foo\nBuild-Depends: debhelper-compat (= 13)\n", "13"},
		{"Build-Depends: debhelper (>= 13)\n", ""}, // old-style, no compat
		{"Build-Depends: debhelper-compat (= 12), foo\n", "12"},
		{"", ""},
		{"No build deps here\n", ""},
	}
	for _, tc := range cases {
		got := extractDebhelperCompatVersion(tc.text)
		if got != tc.want {
			t.Errorf("extractDebhelperCompatVersion(%q) = %q, want %q", tc.text, got, tc.want)
		}
	}
}

func TestCheckDebhelperCompatRedundant(t *testing.T) {
	tmp := t.TempDir()
	debDir := filepath.Join(tmp, "debian")
	if err := os.MkdirAll(debDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(debDir, "compat"), []byte("13\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	s := &Server{workspace: newWorkspaceCache(tmp)}
	controlText := "Source: foo\nBuild-Depends: debhelper-compat (= 13)\n"
	diags := s.checkDebhelperCompat(controlText)

	foundRedundant := false
	for _, d := range diags {
		if d.Code == "redundant-debhelper-compat" {
			foundRedundant = true
		}
	}
	if !foundRedundant {
		t.Error("expected redundant-debhelper-compat diagnostic")
	}
}

func TestCheckDebhelperCompatMismatch(t *testing.T) {
	tmp := t.TempDir()
	debDir := filepath.Join(tmp, "debian")
	if err := os.MkdirAll(debDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(debDir, "compat"), []byte("12\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	s := &Server{workspace: newWorkspaceCache(tmp)}
	controlText := "Source: foo\nBuild-Depends: debhelper-compat (= 13)\n"
	diags := s.checkDebhelperCompat(controlText)

	foundMismatch := false
	for _, d := range diags {
		if d.Code == "debhelper-compat-mismatch" {
			foundMismatch = true
		}
	}
	if !foundMismatch {
		t.Error("expected debhelper-compat-mismatch diagnostic")
	}
}

func TestCheckDebhelperCompatNoCompatFile(t *testing.T) {
	tmp := t.TempDir()
	debDir := filepath.Join(tmp, "debian")
	if err := os.MkdirAll(debDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// No compat file written.

	s := &Server{workspace: newWorkspaceCache(tmp)}
	controlText := "Source: foo\nBuild-Depends: debhelper-compat (= 13)\n"
	diags := s.checkDebhelperCompat(controlText)
	if len(diags) != 0 {
		t.Errorf("expected 0 diagnostics without debian/compat, got %d", len(diags))
	}
}

func TestCheckQuiltSeriesMissing(t *testing.T) {
	tmp := t.TempDir()
	debDir := filepath.Join(tmp, "debian")
	if err := os.MkdirAll(filepath.Join(debDir, "source"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(debDir, "source", "format"), []byte("3.0 (quilt)\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// No patches/series file.

	s := &Server{workspace: newWorkspaceCache(tmp)}
	diags := s.checkQuiltSeries()
	if len(diags) != 1 || diags[0].Code != "quilt-missing-series" {
		t.Errorf("expected 1 quilt-missing-series diagnostic, got %d", len(diags))
	}
}

func TestCheckQuiltSeriesPresent(t *testing.T) {
	tmp := t.TempDir()
	debDir := filepath.Join(tmp, "debian")
	if err := os.MkdirAll(filepath.Join(debDir, "source"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(debDir, "patches"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(debDir, "source", "format"), []byte("3.0 (quilt)\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(debDir, "patches", "series"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	s := &Server{workspace: newWorkspaceCache(tmp)}
	diags := s.checkQuiltSeries()
	if len(diags) != 0 {
		t.Errorf("expected 0 diagnostics when series exists, got %d", len(diags))
	}
}

func TestCheckQuiltSeriesNativeFormat(t *testing.T) {
	tmp := t.TempDir()
	debDir := filepath.Join(tmp, "debian")
	if err := os.MkdirAll(filepath.Join(debDir, "source"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(debDir, "source", "format"), []byte("3.0 (native)\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	s := &Server{workspace: newWorkspaceCache(tmp)}
	diags := s.checkQuiltSeries()
	if len(diags) != 0 {
		t.Errorf("expected 0 diagnostics for native format, got %d", len(diags))
	}
}

func TestStripFileScheme(t *testing.T) {
	cases := []struct{ in, want string }{
		{"file:///home/user/pkg", "/home/user/pkg"},
		{"/home/user/pkg", "/home/user/pkg"},
		{"", ""},
	}
	for _, tc := range cases {
		if got := stripFileScheme(tc.in); got != tc.want {
			t.Errorf("stripFileScheme(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
