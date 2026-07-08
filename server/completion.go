// SPDX-License-Identifier: GPL-3.0-or-later

package server

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/BAMF0/debpack-lsp/debpkg"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
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
		return s.controlCompletions(text, lineUpTo, int(params.Position.Line), s.snippetsSupported), nil
	case debpkg.FileTypeRules:
		return s.rulesCompletions(lineUpTo, text, s.snippetsSupported), nil
	case debpkg.FileTypeCopyright:
		return s.copyrightCompletions(lineUpTo, text, s.snippetsSupported), nil
	case debpkg.FileTypePatch:
		return s.patchCompletions(lineUpTo, text, s.snippetsSupported), nil
	case debpkg.FileTypeWatch:
		return s.watchCompletions(lineUpTo, text, s.snippetsSupported), nil
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// changelog completions: bug number references
// ---------------------------------------------------------------------------

func (s *Server) changelogCompletions(lineUpTo, fullText string, lineNum, col int) []protocol.CompletionItem {
	// Priority 1: cursor is inside an "LP: #" reference →
	// complete the bug number, filtering already-listed bugs.
	if prefix, digits := debpkg.BugRefAtCursor(lineUpTo); prefix != "" {
		return s.changelogNumberCompletions(lineUpTo, fullText, lineNum, digits)
	}

	// Priority 2: cursor is after ")" on a header line, before ";" →
	// complete the distribution suite name.
	if suite := debpkg.ChangelogSuiteAtCursor(lineUpTo); suite != "" || strings.HasSuffix(lineUpTo, ") ") {
		return changelogSuiteCompletions(suite)
	}

	// Priority 3: cursor is after "urgency=" → complete the urgency value.
	if urgency := debpkg.ChangelogUrgencyAtCursor(lineUpTo); urgency != "" || strings.HasSuffix(strings.ToLower(lineUpTo), "urgency=") {
		return changelogUrgencyCompletions(urgency)
	}

	// Priority 4: cursor is on a sub-bullet under a "* Fixes:" parent →
	// complete the bug title and insert "title (LP: #N)".
	if startCol, typedText, ok := debpkg.ChangelogFixesBulletAtLine(fullText, lineNum); ok {
		return s.changelogTitleCompletions(fullText, typedText, lineNum, startCol, col)
	}

	// Priority 5: keyword "entry" or "changelog" → insert a new changelog
	// entry header with placeholders.
	trimmed := strings.ToLower(strings.TrimSpace(lineUpTo))
	if trimmed == "entry" || trimmed == "changelog" {
		return genericSnippetItem("Changelog entry", changelogEntrySnippet, changelogEntryPlain, s.snippetsSupported)
	}

	return nil
}

// changelogSuiteCompletions completes distribution suite names (unstable,
// focal, jammy, …) when the cursor is between ")" and ";" on a changelog
// header line.
func changelogSuiteCompletions(prefix string) []protocol.CompletionItem {
	lower := strings.ToLower(prefix)
	var items []protocol.CompletionItem
	for _, suite := range debpkg.KnownChangelogSuites {
		if !strings.HasPrefix(strings.ToLower(suite), lower) {
			continue
		}
		s := suite
		items = append(items, protocol.CompletionItem{
			Label:      suite,
			InsertText: &s,
		})
	}
	return items
}

// changelogUrgencyCompletions completes urgency values (low, medium, high,
// emergency, critical) when the cursor is after "urgency=".
func changelogUrgencyCompletions(prefix string) []protocol.CompletionItem {
	lower := strings.ToLower(prefix)
	var items []protocol.CompletionItem
	for _, urg := range debpkg.KnownUrgencies {
		if !strings.HasPrefix(strings.ToLower(urg), lower) {
			continue
		}
		u := urg
		items = append(items, protocol.CompletionItem{
			Label:      urg,
			InsertText: &u,
		})
	}
	return items
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

func (s *Server) controlCompletions(fullText, lineUpTo string, lineIdx int, snippetsSupported bool) []protocol.CompletionItem {
	var items []protocol.CompletionItem

	// Blank line + empty control file → offer full source stanza snippet.
	if strings.TrimSpace(lineUpTo) == "" && !debpkg.HasControlContent(fullText) {
		items = append(items, genericSnippetItem("Control source stanza", controlSourceStanzaSnippet, controlSourceStanzaPlain, snippetsSupported)...)
	}

	// Keyword "package" → offer binary package stanza snippet (in addition
	// to the Package: field-name completion).
	if strings.EqualFold(strings.TrimSpace(lineUpTo), "package") {
		items = append(items, genericSnippetItem("Binary package stanza", controlBinaryStanzaSnippet, controlBinaryStanzaPlain, snippetsSupported)...)
	}

	// If the cursor is before ':', complete field names.
	if fieldPrefix := debpkg.FieldAtCursor(lineUpTo); fieldPrefix != "" {
		items = append(items, controlFieldNameItems(fieldPrefix, snippetsSupported)...)
		return items
	}

	// Otherwise complete field values.
	fullLine := fullLineAt(fullText, lineIdx)
	fieldName := debpkg.FieldNameFromLine(fullLine)
	if fieldName == "" {
		return items
	}
	f := debpkg.LookupField(fieldName)
	if f == nil || len(f.Values) == 0 {
		return items
	}
	items = append(items, stringsToCompletions(f.Values, protocol.CompletionItemKindValue)...)
	return items
}

func controlFieldNameItems(prefix string, snippetsSupported bool) []protocol.CompletionItem {
	lower := strings.ToLower(prefix)
	var items []protocol.CompletionItem
	for _, f := range debpkg.KnownControlFields {
		if !strings.HasPrefix(strings.ToLower(f.Name), lower) {
			continue
		}
		kind := protocol.CompletionItemKindField
		doc := f.Description
		item := protocol.CompletionItem{
			Label:      f.Name,
			Kind:       &kind,
			Detail:     &doc,
			InsertText: strPtr(f.Name + ": "),
		}
		if snippetsSupported {
			insert, fmt := fieldSnippetInsert(f.Name, f.Values)
			item.InsertText = &insert
			item.InsertTextFormat = &fmt
		}
		items = append(items, item)
	}
	return items
}

// ---------------------------------------------------------------------------
// rules completions: dh_* commands
// ---------------------------------------------------------------------------

func (s *Server) rulesCompletions(lineUpTo, fullText string, snippetsSupported bool) []protocol.CompletionItem {
	var items []protocol.CompletionItem

	// Blank line + no shebang → offer minimal rules template.
	if strings.TrimSpace(lineUpTo) == "" && !debpkg.HasRulesShebang(fullText) {
		items = append(items, genericSnippetItem("Rules file template", rulesTemplateSnippet, rulesTemplatePlain, snippetsSupported)...)
	}

	// Keyword "override" → offer override target snippet.
	trimmed := strings.ToLower(strings.TrimSpace(lineUpTo))
	if strings.HasPrefix(trimmed, "override") {
		items = append(items, genericSnippetItem("Override target", rulesOverrideSnippet, rulesOverridePlain, snippetsSupported)...)
	}

	// dh_* command completion (existing behaviour).
	idx := strings.LastIndexAny(lineUpTo, " \t")
	word := lineUpTo[idx+1:]
	if strings.HasPrefix(word, "dh") {
		all := s.dh.All()
		for _, cmd := range all {
			if !strings.HasPrefix(cmd.Name, word) {
				continue
			}
			kind := protocol.CompletionItemKindFunction
			items = append(items, protocol.CompletionItem{
				Label:  cmd.Name,
				Kind:   &kind,
				Detail: &cmd.Synopsis,
				Documentation: &protocol.MarkupContent{
					Kind:  protocol.MarkupKindMarkdown,
					Value: dhMarkdown(cmd),
				},
			})
		}
		sort.Slice(items, func(i, j int) bool { return items[i].Label < items[j].Label })
	}
	return items
}

// ---------------------------------------------------------------------------
// copyright completions (DEP-5)
// ---------------------------------------------------------------------------

func (s *Server) copyrightCompletions(lineUpTo, fullText string, snippetsSupported bool) []protocol.CompletionItem {
	var items []protocol.CompletionItem

	// Blank line + no Format: field → offer DEP-5 copyright header snippet.
	if strings.TrimSpace(lineUpTo) == "" && !debpkg.HasCopyrightFormat(fullText) {
		items = append(items, genericSnippetItem("DEP-5 copyright header", copyrightHeaderSnippet, copyrightHeaderPlain, snippetsSupported)...)
	}

	if fieldPrefix := debpkg.FieldAtCursor(lineUpTo); fieldPrefix != "" {
		lower := strings.ToLower(fieldPrefix)
		for _, f := range debpkg.KnownCopyrightFields {
			if !strings.HasPrefix(strings.ToLower(f.Name), lower) {
				continue
			}
			kind := protocol.CompletionItemKindField
			item := protocol.CompletionItem{
				Label:      f.Name,
				Kind:       &kind,
				Detail:     &f.Description,
				InsertText: strPtr(f.Name + ": "),
			}
			if snippetsSupported {
				insert, fmt := fieldSnippetInsert(f.Name, f.Values)
				item.InsertText = &insert
				item.InsertTextFormat = &fmt
			}
			items = append(items, item)
		}
		return items
	}
	// License value completions
	if strings.Contains(strings.ToLower(lineUpTo), "license:") {
		return stringsToCompletions(licenseValues(), protocol.CompletionItemKindValue)
	}
	return items
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

func (s *Server) patchCompletions(lineUpTo, fullText string, snippetsSupported bool) []protocol.CompletionItem {
	var items []protocol.CompletionItem

	// Offer the full DEP-3 header snippet when the user is on a blank line
	// and the file does not yet contain a recognised DEP-3 header block.
	if strings.TrimSpace(lineUpTo) == "" && !debpkg.HasDep3Header(fullText) {
		items = append(items, patchSnippetItems(snippetsSupported)...)
	}

	// Field-name completion (existing behaviour).
	fieldPrefix := debpkg.FieldAtCursor(lineUpTo)
	lower := strings.ToLower(fieldPrefix)
	for _, f := range debpkg.KnownPatchFields {
		if !strings.HasPrefix(strings.ToLower(f.Name), lower) {
			continue
		}
		kind := protocol.CompletionItemKindField
		item := protocol.CompletionItem{
			Label:      f.Name,
			Kind:       &kind,
			Detail:     &f.Description,
			InsertText: strPtr(f.Name + ": "),
		}
		if snippetsSupported {
			insert, fmt := fieldSnippetInsert(f.Name, f.Values)
			item.InsertText = &insert
			item.InsertTextFormat = &fmt
		}
		items = append(items, item)
	}
	return items
}

// ---------------------------------------------------------------------------
// Snippet templates (LSP snippet syntax + plain-text fallbacks)
// ---------------------------------------------------------------------------

// genericSnippetItem builds a single snippet completion item. When the
// client supports snippets, the snippetText (with ${1:...} placeholders) is
// used with InsertTextFormatSnippet. Otherwise, plainText is used so the
// inserted text contains no placeholder markers.
func genericSnippetItem(label, snippetText, plainText string, snippetsSupported bool) []protocol.CompletionItem {
	kind := protocol.CompletionItemKindSnippet
	detail := "Snippet"
	if snippetsSupported {
		format := protocol.InsertTextFormatSnippet
		insert := snippetText
		return []protocol.CompletionItem{{
			Label:            label,
			Kind:             &kind,
			Detail:           &detail,
			InsertText:       &insert,
			InsertTextFormat: &format,
		}}
	}
	insert := plainText
	return []protocol.CompletionItem{{
		Label:      label,
		Kind:       &kind,
		Detail:     &detail,
		InsertText: &insert,
	}}
}

// Changelog entry header snippet.
const changelogEntrySnippet = `${1:package} (${2:version}) ${3:unstable}; urgency=${4:medium}

  * ${5:changes}

 -- ${6:Name} <${7:email}>  ${8:Weekday, DD Mon YYYY HH:MM:SS +0000}`

const changelogEntryPlain = `package (version) unstable; urgency=medium

  * changes

 -- Name <email>  Weekday, DD Mon YYYY HH:MM:SS +0000`

// Control source stanza snippet (source + one binary package).
// \$ escapes literal $ in Debian substvars like ${shlibs:Depends}.
const controlSourceStanzaSnippet = `Source: ${1:package}
Section: ${2:utils}
Priority: ${3:optional}
Maintainer: ${4:Name <email>}
Standards-Version: ${5:4.7.1}
Homepage: ${6:https://}

Package: ${7:package}
Architecture: ${8:any}
Depends: \${shlibs:Depends}, \${misc:Depends}
Description: ${9:short description}
 ${10:long description}`

const controlSourceStanzaPlain = `Source: package
Section: utils
Priority: optional
Maintainer: Name <email>
Standards-Version: 4.7.1
Homepage: https://

Package: package
Architecture: any
Depends: ${shlibs:Depends}, ${misc:Depends}
Description: short description
 long description`

// Control binary package stanza snippet.
const controlBinaryStanzaSnippet = `Package: ${1:package}
Architecture: ${2:any}
Depends: \${shlibs:Depends}, \${misc:Depends}
Description: ${3:short description}
 ${4:long description}`

const controlBinaryStanzaPlain = `Package: package
Architecture: any
Depends: ${shlibs:Depends}, ${misc:Depends}
Description: short description
 long description`

// DEP-5 copyright header snippet.
const copyrightHeaderSnippet = `Format: https://www.debian.org/doc/packaging-manuals/copyright-format/1.0/
Upstream-Name: ${1:name}
Upstream-Contact: ${2:Name <email>}
Source: ${3:URL}

Files: *
Copyright: ${4:2024 Name <email>}
License: ${5:GPL-3+}`

const copyrightHeaderPlain = `Format: https://www.debian.org/doc/packaging-manuals/copyright-format/1.0/
Upstream-Name: name
Upstream-Contact: Name <email>
Source: URL

Files: *
Copyright: 2024 Name <email>
License: GPL-3+`

// Watch file template snippet.
const watchTemplateSnippet = `version=4
${1:https://example.com/downloads/} .*/${2:package}-(\d[\d.]*)\.tar\.gz`

const watchTemplatePlain = `version=4
https://example.com/downloads/ .*/package-(\d[\d.]*)\.tar\.gz`

// Minimal rules file template snippet.
const rulesTemplateSnippet = `#!/usr/bin/make -f
%:
	dh $@
`

const rulesTemplatePlain = `#!/usr/bin/make -f
%:
	dh $@
`

// Rules override target snippet.
const rulesOverrideSnippet = `override_dh_${1:auto_install}:
	${2:}`

const rulesOverridePlain = `override_dh_auto_install:
	`

// patchHeaderSnippet is a single comprehensive DEP-3 patch header, using LSP
// snippet syntax. Continuation lines start with a single space per DEP-3
// convention (matches what lint_patch.go's parser accepts).
const patchHeaderSnippet = `Description: ${1:short description}
 ${2:longer explanation of what the patch does and why}
Origin: ${3:upstream, https://example.com/commit}
Forwarded: ${4:no}
Author: ${5:Name <email@example.com>}
Bug: ${6:https://example.com/bug}
Last-Update: ${7:YYYY-MM-DD}
$0`

// patchHeaderPlain is the plain-text fallback for clients that do not
// support LSP snippet syntax — placeholder markers are stripped so the
// user gets a syntactically valid header they can edit by hand.
const patchHeaderPlain = `Description: short description
 longer explanation of what the patch does and why
Origin: upstream, https://example.com/commit
Forwarded: no
Author: Name <email@example.com>
Bug: https://example.com/bug
Last-Update: YYYY-MM-DD
`

// patchSnippetItems returns the DEP-3 header snippet as completion items.
// When the client does not support snippets, the plain-text variant is used
// so the inserted text contains no ${1:...} markers.
func patchSnippetItems(snippetsSupported bool) []protocol.CompletionItem {
	kind := protocol.CompletionItemKindSnippet
	detail := "Full DEP-3 patch header block"
	if snippetsSupported {
		format := protocol.InsertTextFormatSnippet
		insert := patchHeaderSnippet
		return []protocol.CompletionItem{{
			Label:            "DEP-3 header",
			Kind:             &kind,
			Detail:           &detail,
			InsertText:       &insert,
			InsertTextFormat: &format,
		}}
	}
	insert := patchHeaderPlain
	return []protocol.CompletionItem{{
		Label:      "DEP-3 header",
		Kind:       &kind,
		Detail:     &detail,
		InsertText: &insert,
	}}
}

// ---------------------------------------------------------------------------
// watch completions
// ---------------------------------------------------------------------------

func (s *Server) watchCompletions(lineUpTo, fullText string, snippetsSupported bool) []protocol.CompletionItem {
	var items []protocol.CompletionItem

	// Blank line + no version= line → offer watch template snippet.
	if strings.TrimSpace(lineUpTo) == "" && !debpkg.HasWatchVersion(fullText) {
		items = append(items, genericSnippetItem("Watch file template", watchTemplateSnippet, watchTemplatePlain, snippetsSupported)...)
	}

	if !debpkg.IsInOpts(lineUpTo) {
		return items
	}
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
	return l[:colToByteOffset(l, col)]
}

func fullLineAt(text string, line int) string {
	lines := strings.Split(text, "\n")
	if line >= len(lines) {
		return ""
	}
	return lines[line]
}

// colToByteOffset converts a 0-indexed character column (as reported by the
// LSP client) to a byte offset within line. If col exceeds the number of
// characters, the full line length is returned.
//
// Note: LSP 3.16 positions are UTF-16 code units; this function uses rune
// counts, which match for BMP characters. Debian packaging files are
// overwhelmingly ASCII/BMP, so this is sufficient in practice.
func colToByteOffset(line string, col int) int {
	if col <= 0 {
		return 0
	}
	n := 0
	for i := range line {
		if n == col {
			return i
		}
		n++
	}
	return len(line)
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

// fieldSnippetInsert builds the snippet insert text and format for a field
// completion. When the field has enumerated Values, it uses LSP choice
// syntax (${1|val1,val2,...|}) so the client offers a dropdown. For free-form
// fields it uses a simple placeholder (${1:value}).
func fieldSnippetInsert(name string, values []string) (string, protocol.InsertTextFormat) {
	if len(values) > 0 {
		// LTP choice syntax: ${1|opt1,opt2,opt3|}
		return name + ": ${1|" + strings.Join(values, ",") + "|}", protocol.InsertTextFormatSnippet
	}
	return name + ": ${1:value}", protocol.InsertTextFormatSnippet
}
