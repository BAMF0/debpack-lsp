// SPDX-License-Identifier: GPL-3.0-or-later

package debpkg

import (
	"regexp"
	"strings"
)

// tokenSplitRe splits text on anything that isn't a letter or digit.
var tokenSplitRe = regexp.MustCompile(`[^a-z0-9]+`)

// stopWords is a small set of common English words that carry no signal for
// bug-title matching.
var stopWords = map[string]bool{
	"the": true, "and": true, "with": true, "for": true, "not": true,
	"are": true, "was": true, "has": true, "have": true, "had": true,
	"but": true, "from": true, "this": true, "that": true, "its": true,
	"when": true, "via": true, "due": true, "per": true, "can": true,
	"does": true, "did": true, "use": true, "used": true,
	"using": true, "than": true, "into": true, "also": true, "about": true,
	"after": true, "some": true, "all": true, "any": true, "out": true,
	"new": true, "now": true, "one": true, "two": true, "been": true,
	"will": true, "would": true, "could": true, "should": true, "may": true,
}

// Tokenize lowercases text, splits on non-alphanumeric characters, and
// returns tokens that are at least 3 characters long and not stop words.
func Tokenize(text string) []string {
	lower := strings.ToLower(text)
	parts := tokenSplitRe.Split(lower, -1)
	tokens := parts[:0]
	for _, p := range parts {
		if len(p) >= 3 && !stopWords[p] {
			tokens = append(tokens, p)
		}
	}
	return tokens
}

// TitleSimilarity returns the number of queryTokens that appear in bugTitle
// (case-insensitive substring match).  Higher is more relevant.
func TitleSimilarity(queryTokens []string, bugTitle string) int {
	lower := strings.ToLower(bugTitle)
	score := 0
	for _, tok := range queryTokens {
		if strings.Contains(lower, tok) {
			score++
		}
	}
	return score
}
