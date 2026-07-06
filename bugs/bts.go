// SPDX-License-Identifier: GPL-3.0-or-later

package bugs

// btsBugSource is a stub for future Debian BTS integration.
// Implement this file to add support for Closes: #NNNNNN references.
//
// The Debian BTS exposes a SOAP API at https://bugs.debian.org/cgi-bin/soap.cgi
// and a plain-text status API at https://bugs.debian.org/cgi-bin/bugreport.cgi
// For caching, a similar ~/.cache/debpack-lsp/bts/<package>.json scheme
// is recommended.

// Uncomment and implement when ready:
//
// type btsBugSource struct{}
//
// func newBTSSource() *btsBugSource { return &btsBugSource{} }
// func (b *btsBugSource) Name() string   { return "bts" }
// func (b *btsBugSource) Prefix() string { return "Closes: #" }
// func (b *btsBugSource) Bugs(pkg string) ([]Bug, error)          { return nil, nil }
// func (b *btsBugSource) BugByID(pkg string, id int) (*Bug, error) { return nil, nil }
