package debpkg

import "strings"

// PatchField describes a known DEP-3 header field for debian/patches/* files.
type PatchField struct {
	Name        string
	Description string
	Values      []string
}

// KnownPatchFields lists DEP-3 patch header fields.
var KnownPatchFields = []PatchField{
	{Name: "Description", Description: "Description of what the patch does and why."},
	{Name: "Author", Description: "Who wrote the patch (name + email)."},
	{Name: "From", Description: "Alias for Author."},
	{Name: "Origin", Description: "Where the patch came from.", Values: []string{"upstream", "backport", "vendor", "other"}},
	{Name: "Bug", Description: "Upstream bug URL this patch fixes."},
	{Name: "Bug-Debian", Description: "Debian BTS bug URL (https://bugs.debian.org/NNNNNN)."},
	{Name: "Bug-Ubuntu", Description: "Launchpad bug URL (https://bugs.launchpad.net/bugs/NNNNNN)."},
	{Name: "Forwarded", Description: "Whether/where the patch was forwarded upstream.", Values: []string{"no", "not-needed"}},
	{Name: "Applied-Upstream", Description: "Indicates the patch has been applied upstream."},
	{Name: "Reviewed-by", Description: "Reviewer(s) of the patch."},
	{Name: "Last-Update", Description: "Date of the last revision of the patch (YYYY-MM-DD)."},
}

// LookupPatchField returns the PatchField for name (case-insensitive).
func LookupPatchField(name string) *PatchField {
	lower := strings.ToLower(name)
	for i := range KnownPatchFields {
		if strings.ToLower(KnownPatchFields[i].Name) == lower {
			return &KnownPatchFields[i]
		}
	}
	return nil
}
