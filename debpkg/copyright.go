package debpkg

import "strings"

// CopyrightField describes a known field in a DEP-5 debian/copyright file.
type CopyrightField struct {
	Name        string
	Description string
	Values      []string
}

// KnownCopyrightFields lists DEP-5 fields.
var KnownCopyrightFields = []CopyrightField{
	// Header stanza
	{Name: "Format", Description: "URI of the DEP-5 specification (must be the first field).", Values: []string{"https://www.debian.org/doc/packaging-manuals/copyright-format/1.0/"}},
	{Name: "Upstream-Name", Description: "The name given to the package by its upstream authors."},
	{Name: "Upstream-Contact", Description: "Preferred address(es) to reach the upstream project."},
	{Name: "Source", Description: "Where to download the upstream source (URL or explanation)."},
	{Name: "Disclaimer", Description: "Disclaimer text for non-free or non-DFSG material."},
	{Name: "Comment", Description: "Any additional information about the package's copyright."},
	// Files stanzas
	{Name: "Files", Description: "Whitespace-separated list of path patterns (globs)."},
	{Name: "Copyright", Description: "One or more copyright statements."},
	{Name: "License", Description: "SPDX license shortname or 'LicenseRef-...' for non-SPDX.", Values: knownSPDXLicenses},
}

var knownSPDXLicenses = []string{
	"Apache-2.0", "Artistic", "Artistic-1.0", "Artistic-2.0",
	"BSD-2-Clause", "BSD-3-Clause", "BSD-4-Clause",
	"BSL-1.0", "CC-BY-3.0", "CC-BY-4.0", "CC-BY-SA-3.0", "CC-BY-SA-4.0",
	"CC-BY-NC-3.0", "CC-BY-NC-4.0", "CC-BY-NC-SA-4.0",
	"CC0-1.0", "CDDL-1.0", "CDDL-1.1", "CPL-1.0",
	"EPL-1.0", "EPL-2.0", "Expat", "MIT",
	"FSFAP", "FTL", "GFDL-1.1", "GFDL-1.2", "GFDL-1.3",
	"GPL-1.0", "GPL-1.0-only", "GPL-1.0-or-later",
	"GPL-2.0", "GPL-2.0-only", "GPL-2.0-or-later",
	"GPL-2+", "GPL-3.0", "GPL-3.0-only", "GPL-3.0-or-later",
	"GPL-3+", "ISC", "LGPL-2.0", "LGPL-2.0-only", "LGPL-2.0-or-later",
	"LGPL-2.1", "LGPL-2.1-only", "LGPL-2.1-or-later", "LGPL-2+",
	"LGPL-3.0", "LGPL-3.0-only", "LGPL-3.0-or-later", "LGPL-3+",
	"MPL-1.0", "MPL-1.1", "MPL-2.0",
	"OFL-1.0", "OFL-1.1", "OpenSSL", "Perl", "PSF-2.0",
	"Python-2.0", "Ruby", "SSPL-1.0",
	"UPL-1.0", "Unlicense", "Vim", "WTFPL",
	"X11", "Zlib", "ZPL-2.0", "ZPL-2.1",
	"public-domain",
}

// LookupCopyrightField returns the CopyrightField for name (case-insensitive).
func LookupCopyrightField(name string) *CopyrightField {
	lower := strings.ToLower(name)
	for i := range KnownCopyrightFields {
		if strings.ToLower(KnownCopyrightFields[i].Name) == lower {
			return &KnownCopyrightFields[i]
		}
	}
	return nil
}
