# debpack-lsp

A Language Server Protocol (LSP) server for Debian packaging files. Provides
completions, hover documentation, diagnostics, quick-fixes, document links,
folding, and symbols for files inside a `debian/` directory.

Licensed under [GPL-3.0-or-later](LICENSE).

## Features

### Completions
- **`debian/changelog`** — `LP: #` bug number completion from the local
  [lpad](https://github.com/BAMF0/lpad) cache, ranked by title similarity to
  the description already written on the line; `Fixes:` sub-bullet title
  completion
- **`debian/control`** — field names and enumerated values
- **`debian/rules`** — `dh_*` command names (scraped from installed man pages)
- **`debian/copyright`** — DEP-5 field names and known SPDX licence identifiers
- **`debian/patches/*.patch`** — DEP-3 header field names + a full DEP-3
  header snippet (when the client supports snippets)
- **`debian/watch`** — `opts=` option names

### Hover
- **`debian/changelog`** — bug title, status and importance for `LP: #`
  references; `Closes: #` references show a Debian BTS link
- **`debian/control`** — field documentation
- **`debian/rules`** — `dh_*` command synopsis, description, and flags (from
  rendered man pages, not boilerplate `--help` output)
- **`debian/copyright`** — DEP-5 field documentation
- **`debian/patches/*.patch`** — DEP-3 field documentation

### Diagnostics

All recognised files are checked for trailing whitespace, whitespace-only
blank lines, and CRLF line endings. Additional file-specific checks:

| File | Checks |
|---|---|
| `debian/changelog` | Entry structure, blank-line counts, urgency value, trailer format and spacing, body indentation |
| `debian/control` | Mandatory fields, `Standards-Version` format and currency, unknown fields, enumerated values, Ubuntu Maintainer consistency, stanza-type field placement, package-name validity, `Homepage`/`Vcs-*` URL schemes |
| `debian/copyright` | `Format:` presence and canonical DEP-5 URI, `Files:` stanza completeness, catch-all `Files: *` stanza, SPDX licence validation |
| `debian/watch` | Version declaration (`version=4` or `Version: 5`) |
| `debian/patches/*.patch` | `Description`/`Subject` required, `Origin` required unless `Author` present, `Origin` keyword, `Forwarded` value, `Last-Update` date format |
| `debian/rules` | Shebang `#!/usr/bin/make -f`, `%:` catch-all target, `dh $@` usage hint |

### Quick-fixes (code actions)

Diagnostics with machine-readable codes offer one-click quick-fixes:
- Remove trailing whitespace / clear whitespace-only lines
- Convert CRLF to LF
- Insert a DEP-3 patch header / DEP-5 `Format:` field / watch `version=4` declaration

### Document links

`LP: #N`, `Closes: #N`, `Homepage:`, `Vcs-*:`, `Bug-*:`, `Origin:`, and
`Forwarded:` values are rendered as clickable links to the relevant URL.

### Folding ranges

Control stanzas, copyright stanzas, changelog entries, and DEP-3 patch
header blocks (above the `---` diff marker) are foldable.

### Document symbols

Control stanzas (`Source:` / `Package:`), copyright stanzas (`Files:`),
changelog entries, and DEP-3 patch header fields appear in the editor's
outline/symbol view.

### Cross-file diagnostics

When the workspace root is known (via `RootURI`), the server reads sibling
`debian/` files to flag:
- Redundant or mismatched `debhelper-compat (= N)` vs `debian/compat`
- Missing `debian/patches/series` when `debian/source/format` declares
  `3.0 (quilt)`

### Formatting

`debian/changelog` can be formatted via `textDocument/formatting` (whole
document).

## Requirements

- **Go 1.25+** to build from source
- **lpad** *(optional)* — populates the bug cache at `~/.cache/lpad/<package>.json`;
  without it, bug completions and hover show a "not in local cache" message
- **man-db + debhelper** *(optional)* — the first run scrapes `dh_*` man pages
  for hover documentation; cached for 7 days at `~/.cache/debpack-lsp/dh-cmds.json`

## Installation

### From source

```sh
git clone https://github.com/BAMF0/debpack-lsp
cd debpack-lsp
make install          # installs binary to ~/.local/bin and Lua plugin
```

To install to a different directory:

```sh
make install INSTALL_DIR=/usr/local/bin
```

To install only the Lua plugin to a custom path:

```sh
make install-lua LUA_DIR=~/.config/nvim/lua
```

### With `go install`

```sh
go install github.com/BAMF0/debpack-lsp@latest
```

## Development

```sh
make build    # compile the binary
make test     # run all tests (with race detector)
make check    # vet + test
make fmt      # format all source files
make vet      # go vet
make lint     # golangci-lint (if installed)
make vuln     # govulncheck (if installed)
```

## Editor Setup

### Neovim

A companion Lua plugin lives at `lua/debpack-lsp.lua`. It uses `vim.lsp.start`
directly — **nvim-lspconfig is not required**.

**lazy.nvim**

```lua
{
  "BAMF0/debpack-lsp",
  build = "make install",
  config = true,
}
```

**packer.nvim**

```lua
use {
  "BAMF0/debpack-lsp",
  run    = "make install",
  config = function() require("debpack-lsp").setup() end,
}
```

**Manual (no plugin manager)**

Copy `lua/debpack-lsp.lua` somewhere on your `runtimepath`, then:

```lua
require("debpack-lsp").setup()
```

### Other editors

Any LSP-capable editor can connect to debpack-lsp over stdio. Point it at the
`debpack-lsp` binary with no arguments. The server advertises
`textDocumentSync`, `completionProvider`, `hoverProvider`,
`formattingProvider`, `codeActionProvider`, `documentLinkProvider`,
`documentSymbolProvider`, and `foldingRangeProvider` capabilities.

## Bug cache

Bug data is loaded from the lpad local cache at:

```
~/.cache/lpad/<source-package>.json
```

The cache is populated and refreshed by running `lpad sync` inside the package
directory. It is loaded automatically when debpack-lsp opens a
`debian/changelog`. Without it, bug references are still shown; they just
report "not found in local cache".

`Closes: #N` (the Debian BTS convention) is not yet backed by a BTS data
source; hovering shows a link to `https://bugs.debian.org/N` and completion
is disabled to avoid offering Launchpad bugs for Debian issues.

## Logging

Set `DEBPACK_LSP_LOG` to a file path to enable debug logging:

```sh
DEBPACK_LSP_LOG=/tmp/debpack-lsp.log nvim debian/control
```

## License

GPL-3.0-or-later. See [LICENSE](LICENSE).
