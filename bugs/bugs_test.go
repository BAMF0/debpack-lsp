package bugs_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/yourusername/debpack-lsp/bugs"
)

// writeLpadCache writes a minimal lpad-format cache file to a temp directory
// and points the HOME env var there so the Launchpad source finds it.
func writeLpadCache(t *testing.T, pkg string, raw []map[string]any) string {
	t.Helper()
	home := t.TempDir()
	dir := filepath.Join(home, ".cache", "lpad")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(map[string]any{
		"cached_at": 9999999999.0,
		"bugs":      raw,
	})
	if err := os.WriteFile(filepath.Join(dir, pkg+".json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	return home
}

func TestStoreLoad(t *testing.T) {
	writeLpadCache(t, "curl", []map[string]any{
		{"id": 1001, "title": "segfault on redirect", "status": "New", "importance": "High", "assignee": nil, "tags": []string{}},
		{"id": 1002, "title": "TLS handshake failure", "status": "In Progress", "importance": "Critical", "assignee": "alice", "tags": []string{"regression"}},
	})

	store := bugs.NewStore()
	store.Load("curl")

	all := store.All()
	if len(all) != 2 {
		t.Fatalf("expected 2 bugs, got %d", len(all))
	}

	b := store.ByID(1002)
	if b == nil {
		t.Fatal("expected to find bug 1002")
	}
	if b.Title != "TLS handshake failure" {
		t.Errorf("unexpected title: %q", b.Title)
	}
	if b.Assignee != "alice" {
		t.Errorf("unexpected assignee: %q", b.Assignee)
	}

	missing := store.ByID(9999)
	if missing != nil {
		t.Errorf("expected nil for unknown bug, got %+v", missing)
	}
}

func TestStoreLoadMissingCache(t *testing.T) {
	t.Setenv("HOME", t.TempDir()) // empty home — no cache

	store := bugs.NewStore()
	store.Load("nonexistent-package")

	all := store.All()
	if len(all) != 0 {
		t.Errorf("expected 0 bugs from missing cache, got %d", len(all))
	}
}
