package bugs

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// launchpadSource reads bug data from the lpad project's local cache
// at ~/.cache/lpad/<package>.json. No network calls are made.
type launchpadSource struct{}

func newLaunchpadSource() *launchpadSource { return &launchpadSource{} }

func (l *launchpadSource) Name() string   { return "launchpad" }
func (l *launchpadSource) Prefix() string { return "LP: #" }

// lpadCacheBug mirrors the JSON schema written by the lpad project.
type lpadCacheBug struct {
	ID         int      `json:"id"`
	Title      string   `json:"title"`
	Status     string   `json:"status"`
	Importance string   `json:"importance"`
	Assignee   *string  `json:"assignee"`
	Tags       []string `json:"tags"`
}

type lpadCacheFile struct {
	Bugs []lpadCacheBug `json:"bugs"`
}

func (l *launchpadSource) cachePath(pkg string) string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cache", "lpad", pkg+".json")
}

func (l *launchpadSource) load(pkg string) ([]lpadCacheBug, error) {
	p := l.cachePath(pkg)
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("lpad cache not found for %q: %w", pkg, err)
	}
	var cf lpadCacheFile
	if err := json.Unmarshal(data, &cf); err != nil {
		return nil, fmt.Errorf("lpad cache parse error: %w", err)
	}
	return cf.Bugs, nil
}

func (l *launchpadSource) Bugs(pkg string) ([]Bug, error) {
	raw, err := l.load(pkg)
	if err != nil {
		return nil, err
	}
	bugs := make([]Bug, 0, len(raw))
	for _, r := range raw {
		assignee := ""
		if r.Assignee != nil {
			assignee = *r.Assignee
		}
		tags := r.Tags
		if tags == nil {
			tags = []string{}
		}
		bugs = append(bugs, Bug{
			ID:         r.ID,
			Title:      r.Title,
			Status:     r.Status,
			Importance: r.Importance,
			Assignee:   assignee,
			Tags:       tags,
			URL:        fmt.Sprintf("https://bugs.launchpad.net/bugs/%d", r.ID),
		})
	}
	return bugs, nil
}

func (l *launchpadSource) BugByID(pkg string, id int) (*Bug, error) {
	bugs, err := l.Bugs(pkg)
	if err != nil {
		return nil, err
	}
	for i := range bugs {
		if bugs[i].ID == id {
			return &bugs[i], nil
		}
	}
	return nil, nil
}
