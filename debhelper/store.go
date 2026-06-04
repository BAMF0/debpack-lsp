// Package debhelper provides completion and hover data for dh_* commands.
//
// On first load it scrapes the synopsis from each dh_* binary found in $PATH
// by running "<cmd> --help" and caches the result to
// ~/.cache/debpack-lsp/dh-cmds.json (TTL: 7 days).
// Subsequent loads read the cache directly, making startup fast.
package debhelper

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	cacheTTL  = 7 * 24 * time.Hour
	cacheFile = "dh-cmds.json"
)

// Command describes a single dh_* command.
type Command struct {
	Name     string   `json:"name"`
	Synopsis string   `json:"synopsis"`
	Flags    []string `json:"flags"`
}

// Store holds all known dh_* commands. Load() must be called before use.
type Store struct {
	mu       sync.RWMutex
	commands []Command
}

// NewStore creates an empty Store.
func NewStore() *Store { return &Store{} }

// Load reads dh_* command data from the disk cache if it is fresh, or
// re-scrapes installed commands and writes a new cache otherwise.
func (s *Store) Load() {
	cmds, err := loadCache()
	if err != nil {
		cmds = scrapeCommands()
		if len(cmds) > 0 {
			_ = saveCache(cmds) // best-effort
		}
	}
	s.mu.Lock()
	s.commands = cmds
	s.mu.Unlock()
}

// All returns a copy of all known commands.
func (s *Store) All() []Command {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Command, len(s.commands))
	copy(out, s.commands)
	return out
}

// ByName returns the Command matching name exactly, or nil.
func (s *Store) ByName(name string) *Command {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for i := range s.commands {
		if s.commands[i].Name == name {
			cp := s.commands[i]
			return &cp
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Cache
// ---------------------------------------------------------------------------

type cacheEnvelope struct {
	CachedAt int64     `json:"cached_at"` // Unix timestamp
	Commands []Command `json:"commands"`
}

func cachePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cache", "debpack-lsp", cacheFile)
}

func loadCache() ([]Command, error) {
	data, err := os.ReadFile(cachePath())
	if err != nil {
		return nil, err
	}
	var env cacheEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, err
	}
	age := time.Since(time.Unix(env.CachedAt, 0))
	if age > cacheTTL {
		return nil, fmt.Errorf("cache stale (age %s)", age.Round(time.Minute))
	}
	return env.Commands, nil
}

func saveCache(cmds []Command) error {
	dir := filepath.Dir(cachePath())
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	env := cacheEnvelope{
		CachedAt: time.Now().Unix(),
		Commands: cmds,
	}
	data, err := json.MarshalIndent(env, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cachePath(), data, 0o644)
}

// ---------------------------------------------------------------------------
// Scraping
// ---------------------------------------------------------------------------

// scrapeCommands enumerates all dh_* binaries in $PATH and extracts their
// synopsis and common flags by running "<cmd> --help".
func scrapeCommands() []Command {
	seen := map[string]bool{}
	var cmds []Command

	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			name := e.Name()
			if !strings.HasPrefix(name, "dh_") {
				continue
			}
			if seen[name] {
				continue
			}
			seen[name] = true

			synopsis, flags := extractHelpInfo(filepath.Join(dir, name))
			cmds = append(cmds, Command{
				Name:     name,
				Synopsis: synopsis,
				Flags:    flags,
			})
		}
	}
	return cmds
}

// extractHelpInfo runs "<cmd> --help" and extracts the first non-empty
// description line and any flags (lines starting with whitespace followed
// by --).
func extractHelpInfo(cmdPath string) (synopsis string, flags []string) {
	out, err := exec.Command(cmdPath, "--help").CombinedOutput()
	if err != nil && len(out) == 0 {
		return "", nil
	}

	lines := strings.Split(string(out), "\n")
	inDesc := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Capture flags: lines that contain "--flag" patterns.
		if flagStart := strings.Index(trimmed, "--"); flagStart >= 0 {
			// Extract just the flag name (up to space, comma, or =).
			rest := trimmed[flagStart:]
			end := strings.IndexAny(rest, " ,=\t")
			var flag string
			if end < 0 {
				flag = rest
			} else {
				flag = rest[:end+1] // include = so clients know it takes a value
			}
			if flag != "--help" && flag != "--version" {
				flags = append(flags, flag)
			}
		}

		// Synopsis: first non-empty line after a "Usage:" or "Synopsis:" heading,
		// or the first non-header content line.
		lower := strings.ToLower(trimmed)
		if strings.HasPrefix(lower, "usage:") || strings.HasPrefix(lower, "synopsis:") {
			inDesc = true
			continue
		}
		if inDesc && synopsis == "" && trimmed != "" {
			synopsis = trimmed
		}
	}

	// Fallback: use the first non-empty, non-header line.
	if synopsis == "" {
		for _, line := range lines {
			t := strings.TrimSpace(line)
			if t != "" && !strings.HasPrefix(strings.ToLower(t), "usage") {
				synopsis = t
				break
			}
		}
	}

	// Deduplicate flags.
	seen := map[string]bool{}
	unique := flags[:0]
	for _, f := range flags {
		if !seen[f] {
			seen[f] = true
			unique = append(unique, f)
		}
	}
	return synopsis, unique
}
