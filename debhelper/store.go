// Package debhelper provides completion and hover data for dh_* commands.
//
// On first load it scrapes the synopsis and description from each dh_*
// binary's man page (rendered via "man -P cat") and supplements the flag
// list from "<cmd> --help". The result is cached to
// ~/.cache/debpack-lsp/dh-cmds.json (TTL: 7 days).
// Subsequent loads read the cache directly, making startup fast.
package debhelper

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	cacheTTL       = 7 * 24 * time.Hour
	cacheFile      = "dh-cmds.json"
	cacheSchemaVer = 2
)

// Command describes a single dh_* command.
type Command struct {
	Name        string   `json:"name"`
	Synopsis    string   `json:"synopsis"`
	Description string   `json:"description,omitempty"`
	Flags       []string `json:"flags"`
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
	SchemaVersion int       `json:"schema_version"`
	CachedAt      int64     `json:"cached_at"` // Unix timestamp
	Commands      []Command `json:"commands"`
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
	if env.SchemaVersion != cacheSchemaVer {
		return nil, fmt.Errorf("cache schema %d is stale (want %d)", env.SchemaVersion, cacheSchemaVer)
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
		SchemaVersion: cacheSchemaVer,
		CachedAt:      time.Now().Unix(),
		Commands:      cmds,
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
// synopsis, description (from the man page) and flags (from --help).
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

			cmd := scrapeCommand(name, filepath.Join(dir, name))
			cmds = append(cmds, cmd)
		}
	}
	return cmds
}

// scrapeCommand builds a Command for a single dh_* program. It prefers the
// man page (for synopsis + description) and supplements flags from --help,
// falling back to --help entirely when no man page is available.
func scrapeCommand(name, cmdPath string) Command {
	synopsis, description := scrapeManpage(name)
	flags := extractFlagsFromHelp(cmdPath)

	// Fallback: if the man page yielded no synopsis, use --help output.
	if synopsis == "" {
		synopsis, flags = extractHelpInfo(cmdPath)
	}

	return Command{
		Name:        name,
		Synopsis:    synopsis,
		Description: description,
		Flags:       flags,
	}
}

// ---------------------------------------------------------------------------
// Man-page scraping
// ---------------------------------------------------------------------------

// sectionHeaderRe matches rendered man-page section headers (e.g. "NAME",
// "SYNOPSIS", "DESCRIPTION", "OPTIONS", "FILES", "SEE ALSO").
var sectionHeaderRe = regexp.MustCompile(`^[A-Z][A-Z ]+$`)

// scrapeManpage renders the man page for name via "man -P cat" and extracts
// the synopsis (from SYNOPSIS) and the first paragraph of DESCRIPTION.
// Returns ("", "") if the man page cannot be rendered.
func scrapeManpage(name string) (synopsis, description string) {
	out, err := exec.Command("man", "-P", "cat", name).CombinedOutput()
	if err != nil || len(out) == 0 {
		return "", ""
	}
	return parseManpage(string(out), name)
}

// parseManpage extracts the synopsis and description from a rendered man page.
//
// Mapping (chosen so the hover shows useful per-command info instead of the
// old "--help" boilerplate):
//   - Synopsis     <- NAME section one-liner ("install documentation into ...")
//   - Description  <- SYNOPSIS usage line + first paragraph of DESCRIPTION
//
// It is a pure function for testability.
func parseManpage(text, name string) (synopsis, description string) {
	lines := strings.Split(text, "\n")

	// Walk the lines tracking the current section. Section content is the
	// indented block following a header line until the next header or blank.
	section := ""
	var synopsisLines, descLines []string
	descStarted := false

	for _, raw := range lines {
		trimmed := strings.TrimRight(raw, " \t\r")

		// Section header: an all-uppercase line on its own.
		if sectionHeaderRe.MatchString(strings.TrimSpace(trimmed)) {
			section = strings.TrimSpace(trimmed)
			continue
		}

		// Skip the leading man-page header/footer line (contains "(N)").
		if section == "" {
			continue
		}

		// Blank line ends a paragraph within a section.
		if strings.TrimSpace(trimmed) == "" {
			if section == "DESCRIPTION" && descStarted && len(descLines) > 0 {
				break // first paragraph complete
			}
			continue
		}

		switch section {
		case "NAME":
			// "     dh_install - install files into package build directories"
			if synopsis == "" {
				if idx := strings.Index(trimmed, " - "); idx >= 0 {
					synopsis = strings.TrimSpace(trimmed[idx+3:])
				}
			}
		case "SYNOPSIS":
			// Collect indented content lines; joins wraps below.
			synopsisLines = append(synopsisLines, strings.TrimSpace(trimmed))
		case "DESCRIPTION":
			if !descStarted {
				descStarted = true
			}
			descLines = append(descLines, strings.TrimSpace(trimmed))
		}
	}

	// Build the description: SYNOPSIS usage line first, then the prose.
	var parts []string
	if usage := joinWrapped(synopsisLines); usage != "" {
		parts = append(parts, usage)
	}
	if prose := joinWrapped(descLines); prose != "" {
		parts = append(parts, prose)
	}
	if len(parts) > 0 {
		description = strings.Join(parts, "\n\n")
	}

	return synopsis, description
}

// joinWrapped joins man-page content lines that were broken across line
// boundaries, handling hyphenation:
//   - Soft hyphens (U+00AD, U+2010) at a line break are removed and the word
//     is rejoined ("de\u2010\nbug" -> "debug").
//   - A regular hyphen at a line break is kept and the word is rejoined
//     ("well-\nknown" -> "well-known").
//   - Other line breaks become a single space.
// Trailing/leading whitespace and internal whitespace runs are collapsed.
func joinWrapped(lines []string) string {
	joined := strings.Join(lines, "\n")
	// Soft hyphens at line breaks: drop the hyphen, rejoin the word.
	joined = strings.ReplaceAll(joined, "\u00ad\n", "")
	joined = strings.ReplaceAll(joined, "\u2010\n", "")
	// Regular hyphen at line break: keep the hyphen, rejoin the word.
	joined = strings.ReplaceAll(joined, "-\n", "-")
	// Remaining line breaks become spaces.
	joined = strings.ReplaceAll(joined, "\n", " ")
	// Also strip any soft hyphens that survived (mid-line hyphenation hints).
	joined = strings.ReplaceAll(joined, "\u00ad", "")
	joined = strings.ReplaceAll(joined, "\u2010", "")
	return collapseWS(joined)
}

// collapseWS collapses runs of whitespace into single spaces.
func collapseWS(s string) string {
	var b strings.Builder
	prevSpace := false
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			if !prevSpace {
				b.WriteByte(' ')
				prevSpace = true
			}
			continue
		}
		prevSpace = false
		b.WriteRune(r)
	}
	return strings.TrimSpace(b.String())
}

// ---------------------------------------------------------------------------
// --help scraping (flag extraction + fallback)
// ---------------------------------------------------------------------------

// extractFlagsFromHelp runs "<cmd> --help" and extracts the flag list only.
func extractFlagsFromHelp(cmdPath string) []string {
	out, err := exec.Command(cmdPath, "--help").CombinedOutput()
	if err != nil && len(out) == 0 {
		return nil
	}
	return parseFlags(string(out))
}

// extractHelpInfo runs "<cmd> --help" and extracts the synopsis and flags
// (the original behaviour, used as a fallback when no man page is found).
func extractHelpInfo(cmdPath string) (synopsis string, flags []string) {
	out, err := exec.Command(cmdPath, "--help").CombinedOutput()
	if err != nil && len(out) == 0 {
		return "", nil
	}
	body := string(out)
	flags = parseFlags(body)

	lines := strings.Split(body, "\n")
	inDesc := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
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
	return synopsis, flags
}

// parseFlags extracts "--flag" patterns from --help output, deduplicated.
// A flag is "--" followed by letters/digits/underscores/hyphens; if it is
// directly followed by "=", the "=" is included to signal a value-taking
// option (e.g. "--sourcedir=").
func parseFlags(helpOutput string) []string {
	lines := strings.Split(helpOutput, "\n")
	var flags []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		idx := strings.Index(trimmed, "--")
		if idx < 0 {
			continue
		}
		rest := trimmed[idx:]
		// Consume the flag name: [a-zA-Z0-9_-] after the leading "--".
		end := 2
		for end < len(rest) {
			c := rest[end]
			if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
				(c >= '0' && c <= '9') || c == '-' || c == '_' {
				end++
				continue
			}
			break
		}
		if end <= 2 { // "--" with no name
			continue
		}
		flag := rest[:end]
		if flag == "--help" || flag == "--version" {
			continue
		}
		// If the char after the name is "=", include it to mark a value option.
		if end < len(rest) && rest[end] == '=' {
			flag += "="
		}
		flags = append(flags, flag)
	}
	// Deduplicate, preserving first-seen order.
	seen := map[string]bool{}
	unique := flags[:0]
	for _, f := range flags {
		if !seen[f] {
			seen[f] = true
			unique = append(unique, f)
		}
	}
	return unique
}
