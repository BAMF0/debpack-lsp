package debpkg

import "strings"

// ControlField describes a known field in debian/control.
type ControlField struct {
	Name        string
	Description string
	// Values lists known/allowed values; nil means free-form.
	Values []string
}

// KnownControlFields is the set of stanza fields recognised in debian/control.
var KnownControlFields = []ControlField{
	// Source stanza
	{Name: "Source", Description: "Source package name."},
	{Name: "Maintainer", Description: "Maintainer name and email address."},
	{Name: "Uploaders", Description: "Additional uploaders (name + email)."},
	{Name: "Section", Description: "Archive section.", Values: knownSections},
	{Name: "Priority", Description: "Package priority.", Values: []string{"required", "important", "standard", "optional", "extra"}},
	{Name: "Build-Depends", Description: "Packages required to build the source package."},
	{Name: "Build-Depends-Indep", Description: "Architecture-independent build dependencies."},
	{Name: "Build-Conflicts", Description: "Packages that must not be installed when building."},
	{Name: "Standards-Version", Description: "Debian Policy version this package complies with."},
	{Name: "Homepage", Description: "Upstream homepage URL."},
	{Name: "Vcs-Browser", Description: "URL to browse the VCS repository."},
	{Name: "Vcs-Git", Description: "Git repository URL."},
	{Name: "Vcs-Svn", Description: "Subversion repository URL."},
	{Name: "Vcs-Bzr", Description: "Bazaar repository URL."},
	{Name: "Vcs-Hg", Description: "Mercurial repository URL."},
	{Name: "Rules-Requires-Root", Description: "Whether debian/rules requires (fake)root.", Values: []string{"no", "binary-targets", "yes"}},
	{Name: "Testsuite", Description: "Name of test suite(s).", Values: []string{"autopkgtest", "autopkgtest-pkg-python", "autopkgtest-pkg-perl", "autopkgtest-pkg-go", "autopkgtest-pkg-ruby"}},
	// Binary stanza
	{Name: "Package", Description: "Binary package name."},
	{Name: "Architecture", Description: "Supported architectures.", Values: knownArchitectures},
	{Name: "Multi-Arch", Description: "Multi-arch qualifier.", Values: []string{"same", "foreign", "allowed", "no"}},
	{Name: "Depends", Description: "Runtime dependencies."},
	{Name: "Recommends", Description: "Recommended packages."},
	{Name: "Suggests", Description: "Suggested packages."},
	{Name: "Enhances", Description: "Packages that this package enhances."},
	{Name: "Pre-Depends", Description: "Dependencies that must be satisfied before unpacking."},
	{Name: "Breaks", Description: "Packages that this package breaks."},
	{Name: "Conflicts", Description: "Packages that conflict with this package."},
	{Name: "Replaces", Description: "Packages replaced by this package."},
	{Name: "Provides", Description: "Virtual packages provided by this package."},
	{Name: "Description", Description: "Short and long description of the package."},
	{Name: "Essential", Description: "Whether this is an essential package.", Values: []string{"yes", "no"}},
	{Name: "Installed-Size", Description: "Approximate installed size in KiB."},
	{Name: "Tag", Description: "Debtags for the package."},
	{Name: "XB-Maemo-Icon-26", Description: "Maemo application icon."},
}

var knownSections = []string{
	"admin", "cli-mono", "comm", "database", "debug", "devel",
	"doc", "editors", "education", "electronics", "embedded", "fonts",
	"games", "gnome", "gnu-r", "gnustep", "graphics", "hamradio",
	"haskell", "httpd", "interpreters", "introspection", "java",
	"javascript", "kde", "kernel", "libdevel", "libs", "lisp",
	"localization", "mail", "math", "metapackages", "misc", "net",
	"news", "ocaml", "oldlibs", "otherosfs", "perl", "php", "python",
	"ruby", "rust", "science", "shells", "sound", "text", "tex",
	"utils", "vcs", "video", "web", "x11", "xfce", "zope",
}

var knownArchitectures = []string{
	"all", "any",
	"amd64", "arm64", "armel", "armhf", "i386",
	"mips64el", "mipsel", "ppc64el", "riscv64", "s390x",
	"alpha", "hppa", "ia64", "m68k", "powerpc", "sh4", "sparc64",
}

// FieldAtCursor returns the field name being completed on the current line
// (i.e. the text before the first ':'), or "" if the line doesn't look like
// a field definition.
func FieldAtCursor(lineUpToCursor string) string {
	if idx := strings.Index(lineUpToCursor, ":"); idx < 0 {
		// Cursor is still in the field name — return what's typed so far.
		return strings.TrimSpace(lineUpToCursor)
	}
	return ""
}

// FieldNameFromLine returns the field name of the current line (text before
// the first ':'), used to look up known values for the field being edited.
func FieldNameFromLine(line string) string {
	name, _, found := strings.Cut(line, ":")
	if !found {
		return ""
	}
	return strings.TrimSpace(name)
}

// LookupField returns the ControlField for name (case-insensitive), or nil.
func LookupField(name string) *ControlField {
	lower := strings.ToLower(name)
	for i := range KnownControlFields {
		if strings.ToLower(KnownControlFields[i].Name) == lower {
			return &KnownControlFields[i]
		}
	}
	return nil
}
