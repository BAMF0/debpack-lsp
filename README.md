<div align="center">

# debpack-lsp

A Language Server Protocol (LSP) server for Debian packaging files.

Provides completions, hover documentation, diagnostics, quick-fixes,
document links, folding, symbols, snippets, and formatting for every file
inside a `debian/` source package directory.

[![CI](https://github.com/BAMF0/debpack-lsp/actions/workflows/ci.yml/badge.svg)](https://github.com/BAMF0/debpack-lsp/actions/workflows/ci.yml)
[![License: GPL-3.0-or-later](https://img.shields.io/badge/license-GPL--3.0--or--later-blue.svg)](LICENSE)

</div>

---

## Why debpack-lsp?

Debian packaging involves editing a dozen different file formats inside a
`debian/` directory — control files, changelogs, rules, copyright, patches,
watch files, and more. Each has its own syntax conventions, required fields,
and common mistakes.

**debpack-lsp** brings IDE-quality tooling to all of them:

- Catch missing mandatory fields, invalid values, and common packaging
  mistakes **as you type** — no need to run `lintian` after every change.
- Get **completions** for field names, `dh_*` commands, SPDX licences,
  bug numbers, and more.
- **Hover** any field or command to see documentation scraped from
  installed man pages.
- Insert **snippets** for entire DEP-3 patch headers, changelog entries,
  control stanzas, copyright files, watch files, and rules targets.
- **Quick-fix** trailing whitespace, CRLF endings, missing headers, and
  version declarations with one click.
- **Click** bug references (`LP: #N`, `Closes: #N`) and URL fields
  (`Homepage`, `Vcs-*`) to open them in a browser.
- **Cross-file checks** warn when `debian/compat` and
  `Build-Depends: debhelper-compat (= N)` disagree, or when
  `3.0 (quilt)` is declared but `patches/series` is missing.

The server is **fully offline** — no network calls, no external lint
libraries. Bug data comes from a local [lpad](https://github.com/BAMF0/lpad)
cache; `dh_*` documentation is scraped from installed man pages and cached
for 7 days.

---

## Supported file types

| File | Lint | Completion | Hover | Snippets | Symbols | Folding | Links |
|---|:---:|:---:|:---:|:---:|:---:|:---:|:---:|
| `debian/control` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `debian/changelog` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `debian/rules` | ✅ | ✅ | ✅ | ✅ | — | — | — |
| `debian/copyright` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `debian/patches/*.patch` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `debian/watch` | ✅ | ✅ | ✅ | ✅ | — | — | — |
| `debian/*.install` | ✅ | — | — | — | — | — | — |
| `debian/*.dirs` | ✅ | — | — | — | — | — | — |
| `debian/*.docs` | ✅ | — | — | — | — | — | — |
| `debian/*.links` | ✅ | — | — | — | — | — | — |
| `debian/*.manpages` | ✅ | — | — | — | — | — | — |
| `debian/patches/series` | ✅ | — | — | — | — | — | — |

All files also receive universal checks: trailing whitespace,
whitespace-only blank lines, and CRLF line-ending detection.

---

## Snippets

debpack-lsp offers snippet templates that expand into full structural
scaffolding with tab-stops for each value:

| Snippet | File | Trigger | Expands into |
|---|---|---|---|
| DEP-3 header | `patches/*.patch` | Blank line in empty file | `Description`, `Origin`, `Forwarded`, `Author`, `Bug`, `Last-Update` |
| Changelog entry | `changelog` | Type `entry` | Full entry header with package, version, suite, urgency, trailer |
| Control source stanza | `control` | Blank line in empty file | `Source`, `Section`, `Priority`, `Maintainer`, `Standards-Version`, `Homepage`, + binary package |
| Binary package stanza | `control` | Type `package` | `Package`, `Architecture`, `Depends`, `Description` |
| DEP-5 copyright header | `copyright` | Blank line in empty file | `Format`, `Upstream-Name`, `Upstream-Contact`, `Source`, `Files: *`, `Copyright`, `License` |
| Watch file template | `watch` | Blank line in empty file | `version=4` + URL line with version regex |
| Rules file template | `rules` | Blank line in empty file | `#!/usr/bin/make -f`, `%:` target, `dh $@` |
| Override target | `rules` | Type `override` | `override_dh_<target>:` with tab |

Debian substvars like `${shlibs:Depends}` are properly escaped in snippet
text so the LSP snippet parser doesn't misinterpret them as placeholders.

---

## Installation

### Neovim

A companion Lua plugin (`lua/debpack-lsp.lua`) uses `vim.lsp.start`
directly — **nvim-lspconfig is not required**.

**lazy.nvim:**

```lua
{
  "BAMF0/debpack-lsp",
  build = "make install",
  config = true,
}
```

**packer.nvim:**

```lua
use {
  "BAMF0/debpack-lsp",
  run = "make install",
  config = function() require("debpack-lsp").setup() end,
}
```

**Manual:** run `make install`, then add `require("debpack-lsp").setup()`
to your init.

> **Snippet expansion:** if you use [nvim-cmp](https://github.com/hrsh7th/nvim-cmp)
> with [LuaSnip](https://github.com/L3MON4D3/LuaSnip), the plugin
> automatically detects `cmp-nvim-lsp` and wires up snippet expansion.
> Without it, completions still work but `${1:...}` placeholders are
> inserted as plain text.

### VS Code

A VS Code extension is available as a `.vsix` package with bundled
binaries (no Go installation required):

```sh
git clone https://github.com/BAMF0/debpack-lsp
cd debpack-lsp
make vscode-package
code --install-extension vscode-ext/debpack-lsp-*.vsix
```

Or in VS Code: `Ctrl+Shift+P` → **Extensions: Install from VSIX** → select
the `.vsix` file.

The extension includes TextMate grammars for syntax highlighting and
finds the `debpack-lsp` binary in this order:
1. The `debpack-lsp.binaryPath` setting
2. The `$PATH` environment variable
3. A binary bundled inside the extension

### Other editors

Any LSP-capable editor can connect to `debpack-lsp` over stdio with no
arguments. The server advertises these capabilities:

`completion` · `hover` · `diagnostics` · `formatting` · `codeAction` ·
`documentLink` · `documentSymbol` · `foldingRange`

### From source (all platforms)

```sh
git clone https://github.com/BAMF0/debpack-lsp
cd debpack-lsp
make install          # binary → ~/.local/bin, Lua plugin → ~/.local/share/nvim
```

Override defaults:

```sh
make install INSTALL_DIR=/usr/local/bin LUA_DIR=~/.config/nvim/lua
```

### With `go install`

```sh
go install github.com/BAMF0/debpack-lsp@latest
```

---

## Diagnostics

### Per-file lint rules

| File | Checks |
|---|---|
| `debian/changelog` | Entry structure, blank-line counts, urgency value, trailer format and spacing, body indentation |
| `debian/control` | Mandatory fields, `Standards-Version` format + currency, unknown fields, enumerated values, Ubuntu Maintainer consistency, stanza-type field placement, package-name validity, `Homepage`/`Vcs-*` URL schemes |
| `debian/copyright` | `Format:` presence and canonical DEP-5 URI, `Files:` stanza completeness, catch-all `Files: *`, SPDX licence validation |
| `debian/watch` | Version declaration (`version=4` or `Version: 5`) |
| `debian/patches/*.patch` | `Description`/`Subject` required, `Origin` required unless `Author` present, `Origin` keyword, `Forwarded` value, `Last-Update` date format |
| `debian/rules` | Shebang `#!/usr/bin/make -f`, `%:` catch-all target, `dh $@` usage hint |
| `*.install` / `*.links` | Field count validation (exactly 2: source + dest) |
| `*.dirs` / `*.docs` / `*.manpages` | Single-path validation |

### Cross-file diagnostics

When the workspace root is known (via `RootURI`), the server reads sibling
`debian/` files to flag:

- **Redundant or mismatched** `debhelper-compat (= N)` vs `debian/compat`
- **Missing `patches/series`** when `source/format` declares `3.0 (quilt)`

### Quick-fixes

Diagnostics with machine-readable codes offer one-click fixes:

- Remove trailing whitespace / clear whitespace-only lines
- Convert CRLF → LF
- Insert DEP-3 patch header / DEP-5 `Format:` field / watch `version=4`

---

## Bug data

Bug completions and hover are backed by the
[lpad](https://github.com/BAMF0/lpad) local cache:

```
~/.cache/lpad/<source-package>.json
```

Populate it with `lpad sync` inside the package directory. The cache is
loaded automatically when a `debian/changelog` is opened. Without it, bug
references show "not found in local cache".

**`Closes: #N`** (the Debian BTS convention) is not yet backed by a BTS
data source. Hovering shows a link to `https://bugs.debian.org/N`; completion
is disabled to avoid offering Launchpad bugs for Debian issues.

---

## Caches

| Cache | Location | TTL | Written by |
|---|---|---|---|
| Launchpad bugs | `~/.cache/lpad/<pkg>.json` | external | `lpad sync` |
| debhelper commands | `~/.cache/debpack-lsp/dh-cmds.json` | 7 days | first server start |

The debhelper cache uses `schema_version` (currently 2); a mismatched
version forces a re-scrape from man pages.

---

## Configuration

### VS Code settings

| Setting | Default | Description |
|---|---|---|
| `debpack-lsp.binaryPath` | `""` | Path to the `debpack-lsp` binary. If empty, searches `$PATH` then bundled. |
| `debpack-lsp.logFile` | `""` | Debug log file path. Sets `DEBPACK_LSP_LOG` env var. |

### Debug logging

Set `DEBPACK_LSP_LOG` to a file path to enable debug logging:

```sh
DEBPACK_LSP_LOG=/tmp/debpack-lsp.log nvim debian/control
```

---

## Development

```sh
make build              # compile the binary
make test               # run all tests (with race detector)
make check              # vet + test
make fmt                # format all source files
make vet                # go vet
make lint               # golangci-lint (if installed)
make vuln               # govulncheck (if installed)
make vscode-package     # build the VS Code .vsix
```

### Architecture

```
main            Entrypoint. Parses --version, calls server.New(version).Run().
server/         LSP wiring: handlers, document store, workspace cache,
                snippet capability detection, cross-file diagnostics.
debpkg/         Domain logic: file-type detection, lint rules, formatter,
                completion triggers, similarity ranking. LSP-free by design.
bugs/           Bug tracker backends (Launchpad via lpad cache; BTS stub).
debhelper/      Man-page scraper for dh_* commands (cached 7 days).
internal/log    Logging only.
```

See [AGENTS.md](AGENTS.md) for a comprehensive guide to the codebase,
including file maps, design decisions, and how to add new file types or
lint rules.

### Requirements

- **Go 1.25+** to build from source
- **Node.js 20+** to build the VS Code extension
- **man-db + debhelper** *(optional)* — for `dh_*` hover documentation
- **lpad** *(optional)* — for Launchpad bug completions and hover

---

## License

[GPL-3.0-or-later](LICENSE). All Go source files carry
`SPDX-License-Identifier: GPL-3.0-or-later` headers.
