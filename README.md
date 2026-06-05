# debpack-lsp

A Language Server Protocol (LSP) server for Debian packaging files. Provides
completions, hover documentation, and diagnostics for files inside a `debian/`
directory.

## Features

### Completions
- **`debian/changelog`** — `LP: #` / `Closes: #` bug number completion from
  the local [lpad](https://github.com/BAMF0/lpad) cache, ranked by title similarity to the description already
  written on the line
- **`debian/control`** — field names and enumerated values
- **`debian/rules`** — `dh_*` command names (scraped from installed man pages)
- **`debian/copyright`** — DEP-5 field names and known SPDX licence identifiers
- **`debian/patches/*.patch`** — DEP-3 header field names
- **`debian/watch`** — `opts=` option names

### Hover
- **`debian/changelog`** — bug title, status and importance for `LP: #` / `Closes: #` references
- **`debian/control`** — field documentation
- **`debian/rules`** — `dh_*` command synopsis and flags
- **`debian/copyright`** — DEP-5 field documentation
- **`debian/patches/*.patch`** — DEP-3 field documentation

### Diagnostics

All recognised files are checked for trailing whitespace and whitespace-only
blank lines. Additional file-specific checks:

| File | Checks |
|---|---|
| `debian/changelog` | Entry structure, blank-line counts, urgency value, trailer format and spacing, body indentation |
| `debian/control` | Mandatory fields, recommended `Section`, `Standards-Version` format, unknown field names, enumerated values, Ubuntu Maintainer consistency |
| `debian/copyright` | `Format:` presence and canonical DEP-5 URI, `Files:` stanza completeness, catch-all `Files: *` stanza |
| `debian/watch` | Version declaration (`version=4` or `Version: 5`) |
| `debian/patches/*.patch` | `Description`/`Subject` required, `Origin` required unless `Author` present, `Origin` keyword, `Forwarded` value, `Last-Update` date format |

## Requirements

- **Go 1.25+** to build from source
- **lpad** *(optional)* — populates the bug cache at `~/.cache/lpad/<package>.json`;
  without it, bug completions and hover show a "not in local cache" message

## Installation

### From source

```sh
git clone https://github.com/BAMF0/debpack-lsp
cd debpack-lsp
make install          # installs to ~/.local/bin
```

To install to a different directory:

```sh
make install INSTALL_DIR=/usr/local/bin
```

### With `go install`

```sh
go install github.com/BAMF0/debpack-lsp@latest
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
`textDocumentSync`, `completionProvider` and `hoverProvider` capabilities.

## Bug cache

Bug data is loaded from the lpad local cache at:

```
~/.cache/lpad/<source-package>.json
```

The cache is populated and refreshed by running `lpad sync` inside the package
directory. It is loaded automatically when debpack-lsp opens a
`debian/changelog`. Without it, bug references are still shown; they just
report "not found in local cache".

## Logging

Set `DEBPACK_LSP_LOG` to a file path to enable debug logging:

```sh
DEBPACK_LSP_LOG=/tmp/debpack-lsp.log nvim debian/control
```

## Status

Early development. Usable day-to-day for Debian and Ubuntu packaging work.
