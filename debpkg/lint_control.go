// SPDX-License-Identifier: GPL-3.0-or-later

package debpkg

import (
	"fmt"
	"regexp"
	"strings"
)

// ---------------------------------------------------------------------------
// Stanza parser
// ---------------------------------------------------------------------------

type controlStanza struct {
	fields    map[string]string // lowercase field name → raw value
	fieldLine map[string]int    // lowercase field name → 0-indexed line number
	start     int               // first line of stanza
	end       int               // one past last line of stanza
}

func parseControlStanzas(lines []string) []controlStanza {
	var stanzas []controlStanza
	var cur *controlStanza

	flush := func(end int) {
		if cur != nil {
			cur.end = end
			stanzas = append(stanzas, *cur)
			cur = nil
		}
	}

	for i, line := range lines {
		if isBlank(line) {
			flush(i)
			continue
		}
		// Continuation line (value continues from previous field).
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			continue
		}
		// Field line: "Name: value"
		name, val, found := strings.Cut(line, ":")
		if !found {
			continue
		}
		if cur == nil {
			cur = &controlStanza{
				fields:    make(map[string]string),
				fieldLine: make(map[string]int),
				start:     i,
			}
		}
		lower := strings.ToLower(strings.TrimSpace(name))
		cur.fields[lower] = strings.TrimSpace(val)
		cur.fieldLine[lower] = i
	}
	flush(len(lines))
	return stanzas
}

// ---------------------------------------------------------------------------
// Regexes
// ---------------------------------------------------------------------------

var standardsVersionRe = regexp.MustCompile(`^\d+\.\d+\.\d+(\.\d+)?$`)

// ---------------------------------------------------------------------------
// Ubuntu Maintainer constant
// ---------------------------------------------------------------------------

const ubuntuMaintainer = "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>"

// ---------------------------------------------------------------------------
// Entry point
// ---------------------------------------------------------------------------

func lintControl(text string, ctx LintContext) []Diag {
	lines := splitLines(text)
	stanzas := parseControlStanzas(lines)

	var diags []Diag
	for i, s := range stanzas {
		isSource := i == 0
		if isSource {
			diags = append(diags, lintSourceStanza(s, ctx)...)
		} else {
			diags = append(diags, lintBinaryStanza(s)...)
		}
		diags = append(diags, checkUnknownControlFields(s)...)
		diags = append(diags, checkEnumeratedControlValues(s)...)
		diags = append(diags, checkStanzaFieldPlacement(s, isSource)...)
		diags = append(diags, checkPackageNameValidity(s, isSource)...)
		diags = append(diags, checkURLFields(s)...)
	}
	return diags
}

// ---------------------------------------------------------------------------
// Source stanza
// ---------------------------------------------------------------------------

func lintSourceStanza(s controlStanza, ctx LintContext) []Diag {
	var diags []Diag

	// Mandatory fields.
	for _, req := range []string{"source", "maintainer", "standards-version"} {
		if _, ok := s.fields[req]; !ok {
			diags = append(diags, Diag{
				Line: s.start, Col: 0, EndLine: s.start, EndCol: 0,
				Severity: SeverityError,
				Message:  fmt.Sprintf("source stanza is missing mandatory field %q", canonicalFieldName(req)),
			})
		}
	}

	// Recommended field.
	if _, ok := s.fields["section"]; !ok {
		diags = append(diags, Diag{
			Line: s.start, Col: 0, EndLine: s.start, EndCol: 0,
			Severity: SeverityWarning,
			Message:  "source stanza is missing recommended field \"Section\"",
		})
	}

	// Standards-Version format.
	if sv, ok := s.fields["standards-version"]; ok {
		if !standardsVersionRe.MatchString(sv) {
			ln := s.fieldLine["standards-version"]
			diags = append(diags, Diag{
				Line: ln, Col: 0, EndLine: ln, EndCol: 0,
				Severity: SeverityWarning,
				Message:  fmt.Sprintf("Standards-Version %q should match X.Y.Z or X.Y.Z.W", sv),
				Code:     "control-standards-version-format",
				Source:   "control",
			})
		} else if isOutdatedStandardsVersion(sv) {
			ln := s.fieldLine["standards-version"]
			diags = append(diags, Diag{
				Line: ln, Col: 0, EndLine: ln, EndCol: 0,
				Severity: SeverityInfo,
				Message:  fmt.Sprintf("Standards-Version %q is outdated; consider updating to %s or later", sv, minStandardsVersion),
				Code:     "control-standards-version-outdated",
				Source:   "control",
			})
		}
	}

	// Ubuntu Maintainer check.
	diags = append(diags, checkUbuntuMaintainer(s, ctx)...)

	return diags
}

// ---------------------------------------------------------------------------
// Binary stanza
// ---------------------------------------------------------------------------

func lintBinaryStanza(s controlStanza) []Diag {
	var diags []Diag
	for _, req := range []string{"package", "architecture", "description"} {
		if _, ok := s.fields[req]; !ok {
			diags = append(diags, Diag{
				Line: s.start, Col: 0, EndLine: s.start, EndCol: 0,
				Severity: SeverityError,
				Message:  fmt.Sprintf("binary stanza is missing mandatory field %q", canonicalFieldName(req)),
			})
		}
	}
	return diags
}

// ---------------------------------------------------------------------------
// Ubuntu Maintainer check
// ---------------------------------------------------------------------------

func checkUbuntuMaintainer(s controlStanza, ctx LintContext) []Diag {
	maintainer, ok := s.fields["maintainer"]
	if !ok {
		return nil // missing maintainer already reported
	}
	ln := s.fieldLine["maintainer"]
	isUbuntuMaint := maintainer == ubuntuMaintainer

	if ctx.IsUbuntu && !isUbuntuMaint {
		return []Diag{{
			Line: ln, Col: 0, EndLine: ln, EndCol: 0,
			Severity: SeverityWarning,
			Message: fmt.Sprintf(
				"Ubuntu package (version contains 'ubuntu') but Maintainer is not %q", ubuntuMaintainer,
			),
		}}
	}
	if !ctx.IsUbuntu && isUbuntuMaint {
		return []Diag{{
			Line: ln, Col: 0, EndLine: ln, EndCol: 0,
			Severity: SeverityWarning,
			Message: fmt.Sprintf(
				"non-Ubuntu package but Maintainer is %q; this is typically reserved for Ubuntu uploads", ubuntuMaintainer,
			),
		}}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Unknown field check
// ---------------------------------------------------------------------------

// xPrefixRe matches extension field names (XB-, XC-, XS-, X-).
var xPrefixRe = regexp.MustCompile(`(?i)^x[bcs]?-`)

func checkUnknownControlFields(s controlStanza) []Diag {
	var diags []Diag
	for lower, lineNum := range s.fieldLine {
		// Extension fields are always valid.
		if xPrefixRe.MatchString(lower) {
			continue
		}
		if LookupField(lower) == nil {
			diags = append(diags, Diag{
				Line: lineNum, Col: 0, EndLine: lineNum, EndCol: 0,
				Severity: SeverityWarning,
				Message:  fmt.Sprintf("unknown control field %q", canonicalFieldName(lower)),
			})
		}
	}
	return diags
}

// ---------------------------------------------------------------------------
// Enumerated value check
// ---------------------------------------------------------------------------

// fieldsWithEnumeratedValues lists the control fields whose Values list is a
// closed set that should be validated.
var fieldsWithEnumeratedValues = []string{
	"priority", "multi-arch", "rules-requires-root", "essential",
}

func checkEnumeratedControlValues(s controlStanza) []Diag {
	var diags []Diag

	// Section: strip optional "area/" prefix before checking.
	if sec, ok := s.fields["section"]; ok {
		ln := s.fieldLine["section"]
		normalised := sec
		if idx := strings.LastIndex(sec, "/"); idx >= 0 {
			normalised = sec[idx+1:]
		}
		if !stringInSlice(normalised, knownSections) {
			diags = append(diags, Diag{
				Line: ln, Col: 0, EndLine: ln, EndCol: 0,
				Severity: SeverityWarning,
				Message:  fmt.Sprintf("unknown section %q", sec),
			})
		}
	}

	// Architecture: space-separated tokens; skip negated forms (!amd64) and
	// wildcard patterns (linux-any, kfreebsd-any, …).
	if arch, ok := s.fields["architecture"]; ok {
		ln := s.fieldLine["architecture"]
		for _, tok := range strings.Fields(arch) {
			if strings.HasPrefix(tok, "!") {
				continue // negated form — skip
			}
			if strings.HasSuffix(tok, "-any") {
				continue // wildcard — skip
			}
			if !stringInSlice(tok, knownArchitectures) {
				diags = append(diags, Diag{
					Line: ln, Col: 0, EndLine: ln, EndCol: 0,
					Severity: SeverityWarning,
					Message:  fmt.Sprintf("unknown architecture %q", tok),
				})
			}
		}
	}

	// Simple single-value enumerated fields.
	for _, fieldName := range fieldsWithEnumeratedValues {
		val, ok := s.fields[fieldName]
		if !ok {
			continue
		}
		f := LookupField(fieldName)
		if f == nil || len(f.Values) == 0 {
			continue
		}
		if !stringInSlice(strings.ToLower(val), f.Values) {
			ln := s.fieldLine[fieldName]
			diags = append(diags, Diag{
				Line: ln, Col: 0, EndLine: ln, EndCol: 0,
				Severity: SeverityWarning,
				Message: fmt.Sprintf(
					"invalid value %q for %s; expected one of: %s",
					val, canonicalFieldName(fieldName), strings.Join(f.Values, ", "),
				),
			})
		}
	}

	return diags
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func stringInSlice(s string, list []string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// canonicalFieldName returns the properly-cased field name for a lower-case
// key, falling back to the input if not found.
func canonicalFieldName(lower string) string {
	f := LookupField(lower)
	if f != nil {
		return f.Name
	}
	return lower
}

// ---------------------------------------------------------------------------
// Standards-Version currency
// ---------------------------------------------------------------------------

// minStandardsVersion is the minimum recommended Debian Policy version.
// As of Debian Policy 4.7.1 (2025), anything below 4.0.0 is very outdated.
const minStandardsVersion = "4.0.0"

// isOutdatedStandardsVersion reports whether sv is older than
// minStandardsVersion. Both are "X.Y.Z" or "X.Y.Z.W" strings compared
// component-wise.
func isOutdatedStandardsVersion(sv string) bool {
	return compareVersions(sv, minStandardsVersion) < 0
}

// compareVersions compares two "X.Y.Z" or "X.Y.Z.W" version strings
// component-wise. Returns -1, 0, or 1.
func compareVersions(a, b string) int {
	ai := splitVersionParts(a)
	bi := splitVersionParts(b)
	for i := 0; i < len(ai) && i < len(bi); i++ {
		if ai[i] < bi[i] {
			return -1
		}
		if ai[i] > bi[i] {
			return 1
		}
	}
	if len(ai) < len(bi) {
		return -1
	}
	if len(ai) > len(bi) {
		return 1
	}
	return 0
}

func splitVersionParts(v string) []int {
	parts := strings.Split(v, ".")
	out := make([]int, len(parts))
	for i, p := range parts {
		n := 0
		for _, c := range p {
			if c >= '0' && c <= '9' {
				n = n*10 + int(c-'0')
			}
		}
		out[i] = n
	}
	return out
}

// ---------------------------------------------------------------------------
// Stanza-type field placement
// ---------------------------------------------------------------------------

// sourceOnlyFields are fields that only make sense in the source stanza.
var sourceOnlyFields = map[string]bool{
	"source": true, "maintainer": true, "uploaders": true,
	"standards-version": true,
	"build-depends":     true, "build-depends-indep": true,
	"build-conflicts": true, "build-conflicts-indep": true,
	"rules-requires-root": true, "testsuite": true,
}

// binaryOnlyFields are fields that only make sense in binary stanzas.
var binaryOnlyFields = map[string]bool{
	"package": true, "architecture": true, "multi-arch": true,
	"depends": true, "recommends": true, "suggests": true,
	"enhances": true, "pre-depends": true,
	"breaks": true, "conflicts": true, "replaces": true,
	"provides": true, "description": true,
	"essential": true, "installed-size": true,
	"built-using": true, "package-type": true,
	"build-profiles": true, "tag": true,
}

func checkStanzaFieldPlacement(s controlStanza, isSource bool) []Diag {
	var diags []Diag
	for lower, lineNum := range s.fieldLine {
		if xPrefixRe.MatchString(lower) {
			continue
		}
		if isSource && binaryOnlyFields[lower] {
			diags = append(diags, Diag{
				Line: lineNum, Col: 0, EndLine: lineNum, EndCol: 0,
				Severity: SeverityWarning,
				Message:  fmt.Sprintf("field %q is typically used in binary (Package) stanzas, not in the source stanza", canonicalFieldName(lower)),
				Code:     "control-field-wrong-stanza",
				Source:   "control",
			})
		}
		if !isSource && sourceOnlyFields[lower] {
			diags = append(diags, Diag{
				Line: lineNum, Col: 0, EndLine: lineNum, EndCol: 0,
				Severity: SeverityWarning,
				Message:  fmt.Sprintf("field %q is typically used in the source stanza, not in binary stanzas", canonicalFieldName(lower)),
				Code:     "control-field-wrong-stanza",
				Source:   "control",
			})
		}
	}
	return diags
}

// ---------------------------------------------------------------------------
// Package / Source name validity
// ---------------------------------------------------------------------------

var packageNameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9+.\-]+$`)

func checkPackageNameValidity(s controlStanza, isSource bool) []Diag {
	if isSource {
		if name, ok := s.fields["source"]; ok {
			if !packageNameRe.MatchString(name) {
				ln := s.fieldLine["source"]
				return []Diag{{
					Line: ln, Col: 0, EndLine: ln, EndCol: 0,
					Severity: SeverityWarning,
					Message:  fmt.Sprintf("source name %q is not a valid Debian package name (must match [a-z0-9][a-z0-9+.-]+)", name),
					Code:     "control-invalid-package-name",
					Source:   "control",
				}}
			}
		}
	} else {
		if name, ok := s.fields["package"]; ok {
			if !packageNameRe.MatchString(name) {
				ln := s.fieldLine["package"]
				return []Diag{{
					Line: ln, Col: 0, EndLine: ln, EndCol: 0,
					Severity: SeverityWarning,
					Message:  fmt.Sprintf("package name %q is not a valid Debian package name (must match [a-z0-9][a-z0-9+.-]+)", name),
					Code:     "control-invalid-package-name",
					Source:   "control",
				}}
			}
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// URL field validation
// ---------------------------------------------------------------------------

// urlFields lists control fields whose values should be http(s) URLs.
var urlFields = map[string]bool{
	"homepage":    true,
	"vcs-browser": true,
}

func checkURLFields(s controlStanza) []Diag {
	var diags []Diag
	for lower, lineNum := range s.fieldLine {
		if !urlFields[lower] {
			continue
		}
		val := strings.TrimSpace(s.fields[lower])
		if val == "" {
			continue
		}
		if !strings.HasPrefix(val, "https://") && !strings.HasPrefix(val, "http://") {
			diags = append(diags, Diag{
				Line: lineNum, Col: 0, EndLine: lineNum, EndCol: 0,
				Severity: SeverityWarning,
				Message:  fmt.Sprintf("%s should be an http(s) URL, got %q", canonicalFieldName(lower), val),
				Code:     "control-invalid-url",
				Source:   "control",
			})
		}
	}
	return diags
}
