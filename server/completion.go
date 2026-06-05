package server

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"github.com/BAMF0/debpack-lsp/debpkg"
)

func (s *Server) completion(ctx *glsp.Context, params *protocol.CompletionParams) (any, error) {
	uri := params.TextDocument.URI
	text, ok := s.docs.get(uri)
	if !ok {
		return nil, nil
	}

	ft := debpkg.FileTypeFromURI(uri)
	lineUpTo := lineUpToCursor(text, int(params.Position.Line), int(params.Position.Character))

	switch ft {
	case debpkg.FileTypeChangelog:
		return s.changelogCompletions(lineUpTo, text, int(params.Position.Line), int(params.Position.Character)), nil
	case debpkg.FileTypeControl:
		return s.controlCompletions(text, lineUpTo, int(params.Position.Line)), nil
	case debpkg.FileTypeRules:
		return s.rulesCompletions(lineUpTo), nil
	case debpkg.FileTypeCopyright:
		return s.copyrightCompletions(lineUpTo), nil
	case debpkg.FileTypePatch:
		return s.patchCompletions(lineUpTo), nil
	case debpkg.FileTypeWatch:
		return s.watchCompletions(lineUpTo), nil
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// changelog completions: bug number references
// ---------------------------------------------------------------------------

func (s *Server) changelogCompletions(lineUpTo, fullText string, lineNum, col int) []protocol.CompletionItem {
	// Priority 1: cursor is inside an "LP: #" or "Closes: #" reference →
	// complete the bug number (existing behaviour), filtering already-listed bugs.
	if prefix, digits := debpkg.BugRefAtCursor(lineUpTo); prefix != "" {
		return s.changelogNumberCompletions(lineUpTo, fullText, lineNum, digits)
	}

	// Priority 2: cursor is on a sub-bullet under a "* Fixes:" parent →
	// complete the bug title and insert "title (LP: #N)".
	if startCol, typedText, ok := debpkg.ChangelogFixesBulletAtLine(fullText, lineNum); ok {
		return s.changelogTitleCompletions(fullText, typedText, lineNum, startCol, col)
	}

	return nil
}

// changelogNumberCompletions handles "LP: #<digits>" cursor context.
// It completes the bug number, filtered against bugs already referenced in the
// file and ranked by title similarity to the text preceding the trigger.
func (s *Server) changelogNumberCompletions(lineUpTo, fullText string, lineNum int, digits string) []protocol.CompletionItem {
	referenced := debpkg.BugNumbersInText(fullText)
	allBugs := s.bugs.All()

	// Rank by title similarity to the enriched context: text before LP: # on
	// the current line, plus the nearest parent "  * <text>" bullet when the
	// current line is a sparse sub-bullet.  Falls back to recency sort when
	// the combined context yields no tokens.
	queryTokens := debpkg.Tokenize(debpkg.ChangelogContextForBugRef(fullText, lineNum, lineUpTo))
	if len(queryTokens) > 0 {
		sort.SliceStable(allBugs, func(i, j int) bool {
			si := debpkg.TitleSimilarity(queryTokens, allBugs[i].Title)
			sj := debpkg.TitleSimilarity(queryTokens, allBugs[j].Title)
			if si != sj {
				return si > sj
			}
			return allBugs[i].ID > allBugs[j].ID
		})
	} else {
		sort.Slice(allBugs, func(i, j int) bool { return allBugs[i].ID > allBugs[j].ID })
	}

	var items []protocol.CompletionItem
	for _, bug := range allBugs {
		if referenced[bug.ID] {
			continue
		}
		idStr := strconv.Itoa(bug.ID)
		if digits != "" && !strings.HasPrefix(idStr, digits) {
			continue
		}
		label := idStr
		detail := bug.Title
		doc := fmt.Sprintf("%s · %s", bug.Status, bug.Importance)
		kind := protocol.CompletionItemKindValue
		items = append(items, protocol.CompletionItem{
			Label:         label,
			Kind:          &kind,
			Detail:        &detail,
			Documentation: &protocol.MarkupContent{Kind: protocol.MarkupKindMarkdown, Value: doc},
			InsertText:    &idStr,
		})
		if len(items) >= 50 {
			break
		}
	}
	return items
}

// changelogTitleCompletions handles a sub-bullet line under "* Fixes:".
// It suggests bug titles as labels and inserts "title (LP: #N)" via TextEdit,
// replacing everything the user has typed since the "- ".
func (s *Server) changelogTitleCompletions(fullText, typedText string, lineNum, startCol, col int) []protocol.CompletionItem {
	referenced := debpkg.BugNumbersInText(fullText)
	allBugs := s.bugs.All()

	queryTokens := debpkg.Tokenize(typedText)
	sort.SliceStable(allBugs, func(i, j int) bool {
		si := debpkg.TitleSimilarity(queryTokens, allBugs[i].Title)
		sj := debpkg.TitleSimilarity(queryTokens, allBugs[j].Title)
		if si != sj {
			return si > sj
		}
		return allBugs[i].ID > allBugs[j].ID
	})

	editRange := protocol.Range{
		Start: protocol.Position{Line: uint32(lineNum), Character: uint32(startCol)},
		End:   protocol.Position{Line: uint32(lineNum), Character: uint32(col)},
	}

	var items []protocol.CompletionItem
	for _, bug := range allBugs {
		if referenced[bug.ID] {
			continue
		}
		bug := bug // capture
		newText := fmt.Sprintf("%s (LP: #%d)", bug.Title, bug.ID)
		detail := fmt.Sprintf("%s · %s", bug.Status, bug.Importance)
		filterText := bug.Title
		kind := protocol.CompletionItemKindText
		items = append(items, protocol.CompletionItem{
			Label:      bug.Title,
			Kind:       &kind,
			Detail:     &detail,
			FilterText: &filterText,
			TextEdit:   protocol.TextEdit{Range: editRange, NewText: newText},
		})
		if len(items) >= 50 {
			break
		}
	}
	return items
}



func (s *Server) controlCompletions(fullText, lineUpTo string, lineIdx int) []protocol.CompletionItem {
	// If the cursor is before ':', complete field names.
	if fieldPrefix := debpkg.FieldAtCursor(lineUpTo); fieldPrefix != "" {
		return controlFieldNameItems(fieldPrefix)
	}

	// Otherwise complete field values.
	fullLine := fullLineAt(fullText, lineIdx)
	fieldName := debpkg.FieldNameFromLine(fullLine)
	if fieldName == "" {
		return nil
	}
	f := debpkg.LookupField(fieldName)
	if f == nil || len(f.Values) == 0 {
		return nil
	}
	return stringsToCompletions(f.Values, protocol.CompletionItemKindValue)
}

func controlFieldNameItems(prefix string) []protocol.CompletionItem {
	lower := strings.ToLower(prefix)
	var items []protocol.CompletionItem
	for _, f := range debpkg.KnownControlFields {
		if !strings.HasPrefix(strings.ToLower(f.Name), lower) {
			continue
		}
		kind := protocol.CompletionItemKindField
		doc := f.Description
		items = append(items, protocol.CompletionItem{
			Label:         f.Name,
			Kind:          &kind,
			Detail:        &doc,
			InsertText:    strPtr(f.Name + ": "),
		})
	}
	return items
}

// ---------------------------------------------------------------------------
// rules completions: dh_* commands
// ---------------------------------------------------------------------------

func (s *Server) rulesCompletions(lineUpTo string) []protocol.CompletionItem {
	// Trigger when user has typed "dh" or "dh_..."
	idx := strings.LastIndexAny(lineUpTo, " \t")
	word := lineUpTo[idx+1:]
	if !strings.HasPrefix(word, "dh") {
		return nil
	}

	all := s.dh.All()
	var items []protocol.CompletionItem
	for _, cmd := range all {
		if !strings.HasPrefix(cmd.Name, word) {
			continue
		}
		kind := protocol.CompletionItemKindFunction
		items = append(items, protocol.CompletionItem{
			Label:    cmd.Name,
			Kind:     &kind,
			Detail:   &cmd.Synopsis,
			Documentation: &protocol.MarkupContent{
				Kind:  protocol.MarkupKindMarkdown,
				Value: dhMarkdown(cmd),
			},
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Label < items[j].Label })
	return items
}

// ---------------------------------------------------------------------------
// copyright completions (DEP-5)
// ---------------------------------------------------------------------------

func (s *Server) copyrightCompletions(lineUpTo string) []protocol.CompletionItem {
	if fieldPrefix := debpkg.FieldAtCursor(lineUpTo); fieldPrefix != "" {
		lower := strings.ToLower(fieldPrefix)
		var items []protocol.CompletionItem
		for _, f := range debpkg.KnownCopyrightFields {
			if !strings.HasPrefix(strings.ToLower(f.Name), lower) {
				continue
			}
			kind := protocol.CompletionItemKindField
			items = append(items, protocol.CompletionItem{
				Label:      f.Name,
				Kind:       &kind,
				Detail:     &f.Description,
				InsertText: strPtr(f.Name + ": "),
			})
		}
		return items
	}
	// License value completions
	if strings.Contains(strings.ToLower(lineUpTo), "license:") {
		return stringsToCompletions(licenseValues(), protocol.CompletionItemKindValue)
	}
	return nil
}

func licenseValues() []string {
	var vals []string
	for _, f := range debpkg.KnownCopyrightFields {
		if strings.ToLower(f.Name) == "license" {
			return f.Values
		}
	}
	return vals
}

// ---------------------------------------------------------------------------
// patch completions (DEP-3)
// ---------------------------------------------------------------------------

func (s *Server) patchCompletions(lineUpTo string) []protocol.CompletionItem {
	fieldPrefix := debpkg.FieldAtCursor(lineUpTo)
	lower := strings.ToLower(fieldPrefix)
	var items []protocol.CompletionItem
	for _, f := range debpkg.KnownPatchFields {
		if !strings.HasPrefix(strings.ToLower(f.Name), lower) {
			continue
		}
		kind := protocol.CompletionItemKindField
		items = append(items, protocol.CompletionItem{
			Label:      f.Name,
			Kind:       &kind,
			Detail:     &f.Description,
			InsertText: strPtr(f.Name + ": "),
		})
	}
	return items
}

// ---------------------------------------------------------------------------
// watch completions
// ---------------------------------------------------------------------------

func (s *Server) watchCompletions(lineUpTo string) []protocol.CompletionItem {
	if !debpkg.IsInOpts(lineUpTo) {
		return nil
	}
	var items []protocol.CompletionItem
	for _, f := range debpkg.KnownWatchOptions {
		kind := protocol.CompletionItemKindProperty
		items = append(items, protocol.CompletionItem{
			Label:  f.Name,
			Kind:   &kind,
			Detail: &f.Description,
		})
	}
	return items
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func lineUpToCursor(text string, line, col int) string {
	lines := strings.Split(text, "\n")
	if line >= len(lines) {
		return ""
	}
	l := lines[line]
	if col > len(l) {
		col = len(l)
	}
	return l[:col]
}

func fullLineAt(text string, line int) string {
	lines := strings.Split(text, "\n")
	if line >= len(lines) {
		return ""
	}
	return lines[line]
}

func stringsToCompletions(vals []string, kind protocol.CompletionItemKind) []protocol.CompletionItem {
	items := make([]protocol.CompletionItem, 0, len(vals))
	for _, v := range vals {
		v := v
		items = append(items, protocol.CompletionItem{
			Label: v,
			Kind:  &kind,
		})
	}
	return items
}
