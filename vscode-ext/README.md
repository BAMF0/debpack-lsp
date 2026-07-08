# debpack-lsp VS Code Extension

Language server for Debian packaging files (`debian/control`, `changelog`,
`rules`, `copyright`, `patches`, `watch`).

## Features

- **Completions** — field names, `dh_*` commands, bug numbers, DEP-3/DEP-5
  snippets, changelog suite/urgency, and more
- **Hover** — field documentation, `dh_*` man-page descriptions, bug cards
- **Diagnostics** — lint rules for all file types (trailing whitespace, CRLF,
  missing fields, unknown values, stanza placement, SPDX licences, etc.)
- **Quick-fixes** — one-click fixes for trailing whitespace, CRLF→LF, missing
  DEP-3/DEP-5 headers, watch version
- **Document links** — clickable `LP: #N`, `Closes: #N`, `Homepage`, `Vcs-*`
- **Folding** — stanzas, changelog entries, DEP-3 header blocks
- **Symbols** — control/copyright/changelog/patch outline
- **Snippets** — DEP-3 header, changelog entry, control stanzas, DEP-5
  copyright, watch template, rules template, override targets
- **Formatting** — changelog whole-document formatting

## Installation

### From .vsix

1. Download the latest `.vsix` from [releases](https://github.com/BAMF0/debpack-lsp/releases)
2. In VS Code: `Ctrl+Shift+P` → "Extensions: Install from VSIX" → select the file

### From source

```sh
git clone https://github.com/BAMF0/debpack-lsp
cd debpack-lsp
make vscode-package
```

The `.vsix` is produced in `vscode-ext/`.

## Binary discovery

The extension finds the `debpack-lsp` binary in this order:

1. The `debpack-lsp.binaryPath` setting (if set and exists)
2. The `$PATH` environment variable
3. A binary bundled inside the extension (for release builds)

If no binary is found, a warning is shown with install instructions.

## Settings

| Setting | Default | Description |
|---|---|---|
| `debpack-lsp.binaryPath` | `""` | Explicit path to the `debpack-lsp` binary. If empty, searches `$PATH` and bundled binary. |
| `debpack-lsp.logFile` | `""` | Path for debug logging (sets `DEBPACK_LSP_LOG`). |

## License

GPL-3.0-or-later
