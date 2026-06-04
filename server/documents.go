package server

import (
	"sync"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

// document holds the current text content of an open file.
type document struct {
	text string
}

// documentStore is a thread-safe in-memory store of open documents.
type documentStore struct {
	mu   sync.RWMutex
	docs map[protocol.DocumentUri]*document
}

func newDocumentStore() *documentStore {
	return &documentStore{docs: make(map[protocol.DocumentUri]*document)}
}

func (ds *documentStore) open(uri protocol.DocumentUri, text string) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.docs[uri] = &document{text: text}
}

func (ds *documentStore) close(uri protocol.DocumentUri) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	delete(ds.docs, uri)
}

func (ds *documentStore) get(uri protocol.DocumentUri) (string, bool) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	d, ok := ds.docs[uri]
	if !ok {
		return "", false
	}
	return d.text, true
}

// applyChanges applies a sequence of incremental or full text changes.
// ContentChanges is []any containing TextDocumentContentChangeEvent or
// TextDocumentContentChangeEventWhole values (as decoded by glsp).
func (ds *documentStore) applyChanges(uri protocol.DocumentUri, changes []any) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	d, ok := ds.docs[uri]
	if !ok {
		return
	}
	for _, raw := range changes {
		switch c := raw.(type) {
		case protocol.TextDocumentContentChangeEventWhole:
			d.text = c.Text
		case protocol.TextDocumentContentChangeEvent:
			if c.Range == nil {
				d.text = c.Text
			} else {
				d.text = applyRangeChange(d.text, *c.Range, c.Text)
			}
		}
	}
}

// applyRangeChange replaces the text within r with newText.
func applyRangeChange(text string, r protocol.Range, newText string) string {
	lines := splitLines(text)

	startOff := lineColToOffset(lines, int(r.Start.Line), int(r.Start.Character))
	endOff := lineColToOffset(lines, int(r.End.Line), int(r.End.Character))

	return text[:startOff] + newText + text[endOff:]
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i+1])
			start = i + 1
		}
	}
	lines = append(lines, s[start:])
	return lines
}

func lineColToOffset(lines []string, line, col int) int {
	off := 0
	for i, l := range lines {
		if i == line {
			return off + col
		}
		off += len(l)
	}
	return off
}
