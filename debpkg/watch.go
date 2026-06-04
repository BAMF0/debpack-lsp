package debpkg

import "strings"

// WatchField describes a known option or keyword in debian/watch.
type WatchField struct {
	Name        string
	Description string
}

// KnownWatchOptions lists uscan options used in the version=4 opts= line.
var KnownWatchOptions = []WatchField{
	{Name: "repacksuffix", Description: "Suffix appended to version when repacking (e.g. +dfsg)."},
	{Name: "repack", Description: "Repack the upstream tarball."},
	{Name: "compression", Description: "Compression format for repacked tarball (xz, gz, bz2, lzma)."},
	{Name: "component", Description: "Name used for a component tarball."},
	{Name: "pgpmode", Description: "PGP signature verification mode (auto, mangle, next, previous, self, none)."},
	{Name: "filenamemangle", Description: "Perl expression to mangle the downloaded filename."},
	{Name: "uversionmangle", Description: "Perl expression to mangle the upstream version."},
	{Name: "dversionmangle", Description: "Perl expression to mangle the Debian version for comparison."},
	{Name: "versionmangle", Description: "Shorthand for both uversionmangle and dversionmangle."},
	{Name: "oversionmangle", Description: "Perl expression to mangle the version used in orig tarball name."},
	{Name: "pagemangle", Description: "Perl expression to mangle the downloaded HTML page before scanning."},
	{Name: "downloadurlmangle", Description: "Perl expression to mangle the download URL."},
	{Name: "mode", Description: "Watch mode: http (default), ftp, git, svn."},
	{Name: "pretty", Description: "Format string for the version from git tags."},
	{Name: "gitmode", Description: "Git fetch mode: shallow (default) or full."},
	{Name: "gitexport", Description: "How to export git repo: default, git-archive."},
	{Name: "gittagmangler", Description: "Perl expression to mangle git tag names to version strings."},
	{Name: "passive", Description: "Use passive FTP mode."},
	{Name: "searchmode", Description: "HTML search mode: html (default) or plain."},
	{Name: "decompress", Description: "Decompress the downloaded file before checking."},
	{Name: "unzipopt", Description: "Options to pass to unzip."},
}

// IsInOpts returns true if lineUpToCursor is within an opts=... clause.
func IsInOpts(lineUpToCursor string) bool {
	lower := strings.ToLower(lineUpToCursor)
	return strings.Contains(lower, "opts=")
}
