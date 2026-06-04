// Package debian provides file-type detection and parsing utilities for
// files found inside a debian/ source package directory.
package debian

import (
	"path"
	"strings"
)

// FileType identifies which debian/ file is being edited.
type FileType int

const (
	FileTypeUnknown FileType = iota
	FileTypeControl
	FileTypeChangelog
	FileTypeRules
	FileTypeWatch
	FileTypeCopyright
	FileTypePatch // debian/patches/*
	FileTypeInstall
	FileTypeDirs
	FileTypeDocs
	FileTypeLinks
	FileTypeManpages
)

// FileTypeFromURI maps a document URI to a FileType.
// It matches on the filename within a debian/ directory.
func FileTypeFromURI(uri string) FileType {
	// Strip scheme (file://)
	p := uri
	if idx := strings.Index(p, "://"); idx >= 0 {
		p = p[idx+3:]
	}

	// Normalise to forward slashes and lower-case the base name for matching.
	p = path.Clean(p)
	base := path.Base(p)
	dir := path.Base(path.Dir(p))
	grandParent := path.Base(path.Dir(path.Dir(p)))

	switch {
	case base == "control" && dir == "debian":
		return FileTypeControl
	case base == "changelog" && dir == "debian":
		return FileTypeChangelog
	case base == "rules" && dir == "debian":
		return FileTypeRules
	case base == "watch" && dir == "debian":
		return FileTypeWatch
	case base == "copyright" && dir == "debian":
		return FileTypeCopyright
	case dir == "patches" && (grandParent == "debian"):
		return FileTypePatch
	case strings.HasSuffix(base, ".install") && dir == "debian":
		return FileTypeInstall
	case strings.HasSuffix(base, ".dirs") && dir == "debian":
		return FileTypeDirs
	case strings.HasSuffix(base, ".docs") && dir == "debian":
		return FileTypeDocs
	case strings.HasSuffix(base, ".links") && dir == "debian":
		return FileTypeLinks
	case strings.HasSuffix(base, ".manpages") && dir == "debian":
		return FileTypeManpages
	}
	return FileTypeUnknown
}

func (ft FileType) String() string {
	switch ft {
	case FileTypeControl:
		return "control"
	case FileTypeChangelog:
		return "changelog"
	case FileTypeRules:
		return "rules"
	case FileTypeWatch:
		return "watch"
	case FileTypeCopyright:
		return "copyright"
	case FileTypePatch:
		return "patch"
	default:
		return "unknown"
	}
}
