package server

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/BAMF0/debpack-lsp/debpkg"
)

// workspaceCache lazily reads sibling files under debian/ so that linters
// can perform cross-file checks (e.g. debhelper-compat vs debian/compat).
// Reads are cached for the lifetime of the session; callers may invalidate
// a specific entry when its document changes via invalidate().
type workspaceCache struct {
	mu      sync.RWMutex
	root    string
	files   map[string]string // debian/-relative path -> content
	missing map[string]bool   // debian/-relative path known to not exist
}

func newWorkspaceCache(root string) *workspaceCache {
	return &workspaceCache{
		root:    root,
		files:   make(map[string]string),
		missing: make(map[string]bool),
	}
}

// debianDir returns the path to the debian/ directory under the workspace
// root, or "" if root is unset.
func (w *workspaceCache) debianDir() string {
	if w.root == "" {
		return ""
	}
	return filepath.Join(w.root, "debian")
}

// read returns the content of the given debian/-relative path (e.g.
// "compat", "source/format"). The result is cached. A missing file returns
// ("", false) without error. An empty file returns ("", true).
func (w *workspaceCache) read(rel string) (string, bool) {
	w.mu.RLock()
	if v, ok := w.files[rel]; ok {
		w.mu.RUnlock()
		if v == "" {
			// Distinguish "missing" (nil sentinel) from "empty file".
			if _, exists := w.missing[rel]; exists {
				return "", false
			}
			return "", true
		}
		return v, true
	}
	w.mu.RUnlock()

	full := filepath.Join(w.debianDir(), rel)
	data, err := os.ReadFile(full)
	content := ""
	missing := false
	if err != nil {
		missing = true
	} else {
		content = string(data)
	}

	w.mu.Lock()
	if missing {
		w.missing[rel] = true
	} else {
		w.files[rel] = content
	}
	w.mu.Unlock()

	if missing {
		return "", false
	}
	return content, true
}

// invalidate removes a cached entry so the next read re-reads from disk.
func (w *workspaceCache) invalidate(rel string) {
	w.mu.Lock()
	delete(w.files, rel)
	delete(w.missing, rel)
	w.mu.Unlock()
}

// crossFileDiagnostics returns diagnostics that depend on more than one
// debian/ file. It is called from publishDiagnostics after the per-file
// lint pass. Only checks relevant to the given file type are run.
func (s *Server) crossFileDiagnostics(ft debpkg.FileType, text string) []debpkg.Diag {
	if s.workspace == nil {
		return nil
	}
	var diags []debpkg.Diag

	switch ft {
	case debpkg.FileTypeControl:
		diags = append(diags, s.checkDebhelperCompat(text)...)
	case debpkg.FileTypePatch:
		diags = append(diags, s.checkQuiltSeries()...)
	}

	return diags
}

// checkDebhelperCompat warns when both Build-Depends: debhelper-compat (= N)
// and a debian/compat file are present (redundant), or when they disagree.
func (s *Server) checkDebhelperCompat(controlText string) []debpkg.Diag {
	// Read debian/compat.
	compatText, hasCompat := s.workspace.read("compat")
	if !hasCompat {
		return nil
	}
	compatVal := strings.TrimSpace(compatText)

	compatInControl := extractDebhelperCompatVersion(controlText)
	if compatInControl == "" {
		// debian/compat present but no debhelper-compat in Build-Depends.
		// That's fine — debian/compat is the classic mechanism.
		return nil
	}

	// Both present — warn about redundancy.
	var diags []debpkg.Diag
	diags = append(diags, debpkg.Diag{
		Line: 0, Col: 0, EndLine: 0, EndCol: 0,
		Severity: debpkg.SeverityWarning,
		Message:  "both debian/compat and Build-Depends: debhelper-compat (= " + compatInControl + ") are present; use only one (prefer debhelper-compat)",
		Code:     "redundant-debhelper-compat",
		Source:   "cross-file",
	})

	if compatInControl != compatVal {
		diags = append(diags, debpkg.Diag{
			Line: 0, Col: 0, EndLine: 0, EndCol: 0,
			Severity: debpkg.SeverityError,
			Message:  "debian/compat (" + compatVal + ") disagrees with Build-Depends: debhelper-compat (= " + compatInControl + ")",
			Code:     "debhelper-compat-mismatch",
			Source:   "cross-file",
		})
	}

	return diags
}

// checkQuiltSeries warns when source/format indicates 3.0 (quilt) but
// debian/patches/series is absent.
func (s *Server) checkQuiltSeries() []debpkg.Diag {
	formatText, ok := s.workspace.read("source/format")
	if !ok {
		return nil
	}
	formatVal := strings.TrimSpace(formatText)
	if !strings.Contains(formatVal, "3.0 (quilt)") {
		return nil
	}

	// Check for debian/patches/series.
	_, hasSeries := s.workspace.read("patches/series")
	if hasSeries {
		return nil
	}

	// Also check if any patches exist (series might be unnecessary if no
	// patches are present — that's fine, not an error).
	return []debpkg.Diag{{
		Line: 0, Col: 0, EndLine: 0, EndCol: 0,
		Severity: debpkg.SeverityInfo,
		Message:  "source format is 3.0 (quilt) but debian/patches/series is missing; create it even if empty to avoid lintian warnings",
		Code:     "quilt-missing-series",
		Source:   "cross-file",
	}}
}

// extractDebhelperCompatVersion scans a control file for
// "debhelper-compat (= N)" and returns N, or "" if not found.
func extractDebhelperCompatVersion(text string) string {
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		// Match across continuation lines too — just search the raw text.
		if idx := strings.Index(line, "debhelper-compat"); idx >= 0 {
			rest := line[idx:]
			if eqIdx := strings.Index(rest, "(="); eqIdx >= 0 {
				afterEq := strings.TrimSpace(rest[eqIdx+2:])
				end := strings.IndexAny(afterEq, ") \t")
				if end > 0 {
					return afterEq[:end]
				}
				if strings.HasSuffix(strings.TrimRight(afterEq, " \t\r"), ")") {
					return strings.TrimRight(afterEq, " \t\r)")
				}
			}
		}
	}
	return ""
}
