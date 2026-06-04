// Package bugs provides bug data retrieval for debpack-lsp.
//
// The BugSource interface is the extensibility seam: add a new implementation
// (e.g. bts.go for the Debian BTS) and register it in NewStore() to make its
// bugs available for completion and hover.
package bugs

import (
	"fmt"
	"sync"
)

// Bug is the canonical bug representation used throughout debpack-lsp.
type Bug struct {
	ID         int
	Title      string
	Status     string
	Importance string
	Assignee   string
	Tags       []string
	URL        string
}

// BugSource is the interface that any bug tracker backend must implement.
// Adding support for a new tracker (e.g. Debian BTS) requires only implementing
// this interface and registering the implementation in NewStore().
type BugSource interface {
	// Name returns a short identifier for the source (e.g. "launchpad", "bts").
	Name() string
	// Prefix is the text that precedes a bug number in changelog/patch files
	// (e.g. "LP: #" or "Closes: #").
	Prefix() string
	// Bugs returns all cached bugs for the given source package.
	Bugs(pkg string) ([]Bug, error)
	// BugByID returns a single bug by ID, or nil if not found.
	BugByID(pkg string, id int) (*Bug, error)
}

// Store aggregates bugs from all registered BugSources and exposes them for
// completion and hover. It is loaded once at server start.
type Store struct {
	mu      sync.RWMutex
	sources []BugSource
	// bugs holds all bugs indexed by ID, populated after Load().
	bugs map[int]*Bug
}

// NewStore creates a Store pre-registered with all built-in sources.
func NewStore() *Store {
	return &Store{
		sources: []BugSource{
			newLaunchpadSource(),
			// Future: newBTSSource(),
		},
		bugs: make(map[int]*Bug),
	}
}

// Load fetches bugs for the given source package from all registered sources
// and caches them in memory. Safe to call concurrently; blocks until done.
func (s *Store) Load(pkg string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.bugs = make(map[int]*Bug)
	for _, src := range s.sources {
		bugs, err := src.Bugs(pkg)
		if err != nil {
			// Non-fatal: log to stderr and continue with other sources.
			fmt.Printf("debpack-lsp: [%s] load error: %v\n", src.Name(), err)
			continue
		}
		for i := range bugs {
			b := &bugs[i]
			s.bugs[b.ID] = b
		}
	}
}

// All returns a copy of all currently cached bugs.
func (s *Store) All() []Bug {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Bug, 0, len(s.bugs))
	for _, b := range s.bugs {
		out = append(out, *b)
	}
	return out
}

// ByID returns the bug with the given ID, or nil if not in cache.
func (s *Store) ByID(id int) *Bug {
	s.mu.RLock()
	defer s.mu.RUnlock()
	b, ok := s.bugs[id]
	if !ok {
		return nil
	}
	cp := *b
	return &cp
}
