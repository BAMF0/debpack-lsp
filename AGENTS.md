# AGENTS.md — debpack-lsp

Everything an AI agent needs to work efficiently on this codebase without
reading every file from scratch.

---

## Project overview

`debpack-lsp` is a Language Server Protocol (LSP) server for the files inside
a Debian source package's `debian/` directory.  It provides completions, hover,
diagnostics, and formatting over stdio using the LSP 3.16 protocol.

**Module path**: `github.com/BAMF0/debpack-lsp`
The old path `github.com/yourusername/debpack-lsp` is dead — using it breaks
the build. Every import must use the `BAMF0` path.

---

## Architecture — package roles

```
main            Thin entrypoint. Parses --version flag, calls server.New(version).Run().
server/         LSP wiring. All protocol handlers, the document store, async
                loading of bug and debhelper caches, workspace cache for
                cross-file diagnostics, snippet capability detection.
debpkg/         All Debian-file domain logic: file-type detection, lint rules,
                formatter, completion trigger detection, similarity ranking.
                MUST NOT import LSP types — it is LSP-free by design.
bugs/           Bug tracker backends. Currently only a Launchpad source that
                reads ~/.cache/lpad/<pkg>.json (no network calls). Exposes a
                Store with All()/ByID() under a RWMutex.
debhelper/      Scrapes installed dh_* man pages (via "man -P cat") once,
                caches for 7 days to ~/.cache/debpack-lsp/dh-cmds.json.
                Exposes a Store with All()/ByName().
internal/log    Logging only (ilog.Logf). No external dependencies.
```

---

## File map

### `debpkg/`

| File | Purpose |
|---|---|
| `filetype.go` | `FileType` enum; `FileTypeFromURI(uri)` — entry point for all file-type detection; `isInDebianPatches` helper for nested patch subdirs |
| `diag.go` | `Diag` (with `Code`/`Source` fields), `Severity`, `LintContext`; `Lint()` dispatcher (incl. `FileTypeRules`); universal pre-pass (trailing whitespace, whitespace-only blank lines, CRLF); `splitLines` (strips `\r`), `isBlank`, `countLeadingSpaces` helpers |
| `changelog.go` | `PackageFromChangelog`, `IsUbuntuChangelog`, `BugRefAtCursor` (LP: # only), `ClosesRefAtCursor` (Closes: # only), `BugNumberAtOffset`, `ClosesRefAtOffset`, `ContextBeforeBugRef`, `BugNumbersInText`, `ChangelogContextForBugRef`, `ChangelogFixesBulletAtLine` |
| `similarity.go` | `Tokenize` (lowercase, split non-alphanum, drop <3 chars and stop words); `TitleSimilarity` (token substring match count) |
| `control.go` | `KnownControlFields`, `LookupField`, `FieldAtCursor`, `FieldNameFromLine`, `knownSections`, `knownArchitectures` |
| `copyright.go` | `KnownCopyrightFields`, `LookupCopyrightField`, `knownSPDXLicenses`, `isKnownLicense` |
| `patches.go` | `KnownPatchFields` (incl. `Subject`, `Acked-by`), `LookupPatchField`, `HasDep3Header` |
| `watch.go` | `KnownWatchOptions`, `IsInOpts` |
| `lint_changelog.go` | All changelog lint rules: header format, urgency, blank-line structure, trailer format, body indentation, contributor blocks |
| `lint_control.go` | All control lint rules: mandatory fields, `Standards-Version` format + currency, unknown fields (X-/XB-/XC-/XS- exempted), enumerated value checking, Ubuntu Maintainer consistency, stanza-type field placement, package-name validity, URL field validation |
| `lint_copyright.go` | DEP-5 rules: `Format:` presence/URI, stanza completeness, catch-all `Files: *` stanza, SPDX licence validation |
| `lint_watch.go` | Watch file version check — valid versions are 4 and 5 only |
| `lint_patch.go` | DEP-3 rules: `Description`/`Subject` required, `Origin` validation, `Forwarded` value, `Last-Update` format, line length |
| `lint_rules.go` | `debian/rules` lint: shebang `#!/usr/bin/make -f`, `%:` catch-all target, `dh $@` usage hint |
| `format_changelog.go` | `FormatChangelog(text) string` — idempotent changelog formatter |
| `debian_test.go` | Tests for `FileTypeFromURI`, `PackageFromChangelog`, `BugRefAtCursor`, `BugNumberAtOffset`, `ClosesRefAtCursor`, `ClosesRefAtOffset` |
| `format_changelog_test.go` | 10 black-box tests for `FormatChangelog` |
| `diag_test.go` | Tests for CRLF detection, trailing whitespace, `splitLines` `\r` stripping |
| `patches_test.go` | Tests for `HasDep3Header` |
| `lint_extra_test.go` | Tests for `lintRules`, control stanza placement, package-name validity, URL fields, Standards-Version currency, SPDX licence validation |
| `lint_test.go` | Pre-existing lint tests (control, copyright, watch, changelog, patch) |

### `server/`

| File | Purpose |
|---|---|
| `server.go` | `Server` struct (incl. `snippetsSupported`, `rootPath`, `workspace`); `New(version)`, `Run()`; handler registration; `maybeLoadBugs`, `maybeReloadBugsOnChange`; `stripFileScheme`, `strPtr` helpers |
| `documents.go` | Thread-safe `documentStore` (open/close/get/applyChanges); `applyRangeChange`; `lineColToOffset` (uses `colToByteOffset` for multi-byte safety); its own `splitLines` (different semantics — see below); `stripTrailingNewline` |
| `completion.go` | All completion handlers; `changelogCompletions` dispatcher; `changelogNumberCompletions`; `changelogTitleCompletions`; `controlCompletions`; `rulesCompletions`; `copyrightCompletions`; `patchCompletions` (snippet-aware); `watchCompletions`; `colToByteOffset`, `lineUpToCursor`, `fullLineAt`, `stringsToCompletions` |
| `hover.go` | All hover handlers; `hoverBugRef` (routes `Closes:#` to Debian BTS message); `bugMarkdown`, `dhMarkdown`; `lineColToByteOffset` (multi-byte safe), `wordAtCol` |
| `diagnostics.go` | `publishDiagnostics` (lint + cross-file checks → `textDocument/publishDiagnostics`); `clearDiagnostics`; surfaces `Diag.Code` as `IntegerOrString` |
| `format.go` | `format` handler — fetches text, calls `debpkg.FormatChangelog`, returns whole-document `TextEdit` if text changed |
| `folding.go` | `foldingRange` handler; `stanzaFolds` (control/copyright), `changelogFolds`, `patchFolds`; `isChangelogHeader`, `trimTrailingBlanks` |
| `symbols.go` | `documentSymbol` handler; `controlSymbols`, `copyrightSymbols`, `changelogSymbols`, `patchSymbols`; `stanzaBounds`, `stanzaRange` |
| `links.go` | `documentLink` handler; `bugRefLinks` (LP/Closes → clickable URLs), `urlFieldLinks` (Homepage/Vcs-*/Bug-*/Origin/Forwarded) |
| `codeaction.go` | `codeAction` handler; `quickFix` (trailing whitespace, blank-line, CRLF→LF, insert DEP-3 header, DEP-5 Format, watch version); `boolPtr` |
| `workspace.go` | `workspaceCache` (lazily reads sibling `debian/` files); `crossFileDiagnostics`; `checkDebhelperCompat`, `checkQuiltSeries`; `extractDebhelperCompatVersion` |
| `documents_test.go` | Tests for `applyRangeChange`, `splitLines`, `lineColToOffset`, `documentStore` (incl. multi-byte and CRLF edge cases) |
| `integration_test.go` | Integration tests exercising the full handler pipeline via a fake `glsp.Context` |
| `lsp_features_test.go` | Tests for folding, links, symbols, and quick-fix helpers |
| `position_test.go` | Tests for `colToByteOffset` and `wordAtCol` (multi-byte safety) |
| `workspace_test.go` | Tests for the sibling-file cache and cross-file checks |

### `bugs/`

| File | Purpose |
|---|---|
| `source.go` | `Bug`, `BugSource` interface, `Store` (Load/All/ByID), `NewStore` |
| `launchpad.go` | `launchpadSource` — reads `~/.cache/lpad/<pkg>.json`; no network |
| `bts.go` | Stub for future Debian BTS support |
| `bugs_test.go` | Tests for `Store`, `Load`, `ByID`, concurrent access |

### `debhelper/`

| File | Purpose |
|---|---|
| `store.go` | `Command` (Name/Synopsis/Description/Flags); `Store` (Load/All/ByName); man-page scraper (`scrapeManpage`, `parseManpage`, `joinWrapped`); `--help` flag fallback (`extractHelpInfo`, `parseFlags`); cache with `schema_version` |
| `store_test.go` | Fixture-based parser tests, `joinWrapped`, `parseFlags`, `collapseWS`, store round-trip, live man-page integration test |

### Root

| File | Purpose |
|---|---|
| `main.go` | Entrypoint; `--version` flag; calls `server.New(version).Run()` |
| `Makefile` | `build`, `install` (+ `install-lua`), `test` (race), `fmt`, `vet`, `lint`, `vuln`, `check`, `clean`, `uninstall` |
| `lua/debpack-lsp.lua` | Neovim companion plugin; uses `vim.lsp.start` directly (no nvim-lspconfig needed) |
| `go.mod` | Module `github.com/BAMF0/debpack-lsp`; Go 1.25; main dependency: `github.com/tliron/glsp v0.2.2` |
| `LICENSE` | GPL-3.0-or-later full text |
| `.github/workflows/ci.yml` | CI: gofmt check, go vet, go test -race, govulncheck |

---

## Critical design decisions

### `debpkg` is LSP-free

`debpkg` must never import `github.com/tliron/glsp` or any LSP package.
`Diag.Severity` values (1–4) are defined to match `protocol.DiagnosticSeverity`
exactly so `server/diagnostics.go` can cast directly: `protocol.DiagnosticSeverity(d.Severity)`.

### Two `splitLines` functions — do not confuse them

- `debpkg.splitLines(text string) []string` — strips a single trailing `\n`
  before splitting so there is no phantom empty final element. Used by all
  linters and the formatter.
- `server/documents.splitLines(s string) []string` — does NOT strip the trailing
  newline; each element includes its `\n`. Used only by `applyRangeChange`.

### All coordinates are 0-indexed

LSP uses 0-indexed line and character offsets throughout. `Diag` fields
(`Line`, `Col`, `EndLine`, `EndCol`) follow the same convention.

### Cross-file state via `LintContext`

`debpkg.Lint(text, ft, ctx LintContext)` receives a `LintContext` struct rather
than individual flags.  Currently it carries only `IsUbuntu bool`, which is
derived from the changelog version string containing `"ubuntu"` (case-insensitive)
and used to enforce the Ubuntu canonical Maintainer check in `lint_control.go`.
Add new cross-file state here without touching the `Lint` signature.

Cross-file diagnostics that need sibling `debian/` files are handled in the
server layer (not via `LintContext`) using `workspaceCache` — see
`server/workspace.go`.

### `Closes: #` is not backed by a BTS backend

`BugRefAtCursor` only matches `LP: #` (Launchpad). `ClosesRefAtCursor` /
`ClosesRefAtOffset` match `Closes: #` for hover routing, but completion is
disabled for `Closes: #` to avoid offering Launchpad bugs for Debian issues.
Hovering a `Closes: #N` reference shows a link to `https://bugs.debian.org/N`
and a "Debian BTS support not yet implemented" message.

### Ubuntu canonical maintainer

Defined as the constant `ubuntuMaintainer` in `debpkg/lint_control.go`:

```
Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>
```

### Bug store is loaded asynchronously

`maybeLoadBugs` is called on every `didOpen`.  It detects the source package
name from the changelog, then fires `go s.bugs.Load(pkg)`.  Completion
requests that arrive before the load finishes return an empty list — this is
intentional and not a bug.

`maybeReloadBugsOnChange` is called on every `didChange` of a changelog file.
If the package name on line 0 has changed (e.g. the user renamed the source
package), it re-triggers `go s.bugs.Load(pkg)`.

### Workspace cache for cross-file diagnostics

`workspaceCache` (`server/workspace.go`) is initialised in `initialize` from
`RootURI`/`RootPath`.  It lazily reads sibling `debian/` files (e.g.
`debian/compat`, `debian/source/format`, `debian/patches/series`) from disk
and caches them.  `crossFileDiagnostics` is called from `publishDiagnostics`
after the per-file lint pass to append cross-file checks
(`checkDebhelperCompat`, `checkQuiltSeries`).

### Snippet support detection

`snippetsSupported` is set in `initialize` from the client's
`textDocument.completion.completionItem.snippetSupport` capability.  When
false, `patchSnippetItems` emits a plain-text fallback (no `${1:...}`
placeholders) instead of an LTP snippet-format item.

### `bugs.Store.All()` has random iteration order

The internal map is `map[int]*Bug`.  Always sort the result of `All()` before
presenting it to the user.  Use `sort.SliceStable` (not `sort.Slice`) when
applying a secondary tiebreaker so equal-scored bugs keep recency order.

### `strPtr` helper

Defined once in `server/server.go`.  Do not redeclare it in other server files.

---

## How each LSP capability works

### Diagnostics

`publishDiagnostics` is called on every `didOpen` and `didChange`.
`clearDiagnostics` is called on `didClose`.

Pipeline:
1. `debpkg.FileTypeFromURI(uri)` → file type
2. `debpkg.Lint(text, ft, LintContext{IsUbuntu: s.isUbuntu})` →
   universal pre-pass (all types) + file-type-specific rules → `[]Diag`
3. Cast each `Diag` to `protocol.Diagnostic` and push via
   `ctx.Notify("textDocument/publishDiagnostics", ...)`

### Completions

Trigger characters: `#`, ` `, `-`, `_`

Dispatcher in `server/completion.go → completion()`:

```
FileTypeChangelog  → changelogCompletions(lineUpTo, fullText, lineNum, col)
FileTypeControl    → controlCompletions(fullText, lineUpTo, lineNum)
FileTypeRules      → rulesCompletions(lineUpTo)
FileTypeCopyright  → copyrightCompletions(lineUpTo)
FileTypePatch      → patchCompletions(lineUpTo, fullText, snippetsSupported)
FileTypeWatch      → watchCompletions(lineUpTo)
```

#### Changelog completions — three-branch priority order

1. **LP: # number completion** (`changelogNumberCompletions`) — fires when
   `debpkg.BugRefAtCursor(lineUpTo)` returns a non-empty prefix (`"LP: #"` only).
   Ranks bugs by `TitleSimilarity` using
   `ChangelogContextForBugRef` as the query (current-line text before trigger
   + nearest parent `  * ...` bullet text). Filters bugs already referenced in
   the file via `BugNumbersInText(fullText)`.

2. **Title completion** (`changelogTitleCompletions`) — fires when
   `debpkg.ChangelogFixesBulletAtLine(fullText, lineNum)` returns ok=true.
   This requires: (a) the current line is a `    - ` sub-bullet, AND (b)
   scanning backwards finds a `  * ...` parent bullet whose text contains
   `"fixes:"` (case-insensitive). Emits each bug as `Label=title` with a
   `TextEdit` that replaces `[lineNum, startCol → col]` with
   `"title (LP: #N)"`. Also filtered by `BugNumbersInText`.

3. **nil** — no completions otherwise.

Branch 1 takes strict priority: even on a Fixes: sub-bullet, if the cursor
is inside `LP: #`, branch 1 fires and branch 2 is never reached.

#### Control completions

- Cursor before `:` on a line → field name completion (`controlFieldNameItems`)
- Cursor after `:` → look up field by name (`LookupField`), complete known values

#### Rules completions

Triggered when the current token starts with `"dh"`.  Completes against
`debhelper.Store.All()` sorted alphabetically.

### Hover

Dispatcher in `server/hover.go → hover()`:

```
FileTypeChangelog → hoverBugRef    — shows bug card (status, importance)
FileTypeControl   → hoverControlField — shows field description + known values
FileTypeRules     → hoverDhCommand — shows dh_* synopsis + flags
FileTypeCopyright → hoverCopyrightField
FileTypePatch     → hoverPatchField
```

### Formatting

Registered as `TextDocumentFormatting` in `server/server.go`.
Handler in `server/format.go`.

Currently only `FileTypeChangelog` is handled; all other file types return nil.

`debpkg.FormatChangelog(text string) string` is idempotent.  The server returns
a single whole-document `TextEdit` if the formatted text differs from the
original, nil otherwise.

### Code actions

`codeAction` handler in `server/codeaction.go`.  Diagnostics carry a
`Code` string (set in `debpkg/diag.go`) so quick-fixes can be keyed to a
specific problem.  Available quick-fixes:
- `trailing-whitespace` → remove trailing whitespace
- `blank-line-whitespace` → clear the line
- `crlf-line-ending` → remove `\r`
- `dep3-missing-description` / `dep3-missing-origin` → insert DEP-3 header
- `dep5-missing-format` → insert `Format:` field
- `watch-missing-version` → insert `version=4`

The overlap check uses range overlap (`diagStart <= reqEnd && diagEnd >= reqStart`),
not start-line containment, so diagnostics that span the requested range are
eligible even if their start line is outside it.

### Document links

`documentLink` handler in `server/links.go`.  Scans for:
- `LP: #N` → `https://bugs.launchpad.net/bugs/N`
- `Closes: #N` → `https://bugs.debian.org/N`
- `Homepage:`, `Vcs-Browser:`, `Bug-*:`, `Origin:`, `Forwarded:` → the URL value

### Document symbols

`documentSymbol` handler in `server/symbols.go`.  Returns hierarchical
`[]DocumentSymbol`:
- Control: each stanza (`Source:` / `Package:`) as a `Module` symbol
- Copyright: each `Files:` stanza as a `File` symbol
- Changelog: each version entry as an `Event` symbol
- Patch: each DEP-3 header field as a `Field` symbol

### Folding ranges

`foldingRange` handler in `server/folding.go`.  Foldable:
- Control/copyright: blank-line-separated stanzas
- Changelog: each version entry (header → next header / EOF)
- Patch: the DEP-3 header block (everything before the `---` diff marker)

Trailing blank lines are trimmed from fold ranges via `trimTrailingBlanks`.

---

## Changelog bullet hierarchy

Debian changelogs use a three-level bullet structure.  The formatter and
`ChangelogFixesBulletAtLine` both depend on understanding these levels.

```
  * Level 1 — top-level bullet (2 spaces + * + space; contIndent = 4)
    - Level 2 — sub-bullet    (4 spaces + - + space; contIndent = 6)
      + Level 3 — third-level (6 spaces + + + space; contIndent = 8)
        Continuation of level 3 (8 spaces)
    Continuation of level 1 (4 spaces)
```

### Bullet character classification in `FormatChangelog`

| Source char | Source indent before char | Canonical output |
|---|---|---|
| `*` | any | `  * text` (level 1) |
| `+` | ≤ 2 spaces | `  * text` (level 1 — non-standard variant, normalised to `*`) |
| `+` | ≥ 3 spaces | `      + text` (level 3 — preserved as `+`) |
| `-` | any | `    - text` (level 2) |

`bulletLineRe` (from `lint_changelog.go`) captures the leading spaces in `m[1]`
and the bullet character in `m[2]`, so `len(m[1])` gives the source indent.

---

## How to add a new file type

1. Add `FileType<Name> FileType = iota` in `debpkg/filetype.go`.
2. Add a detection case in `FileTypeFromURI` and a `String()` case.
3. Create `debpkg/lint_<name>.go` with a `lint<Name>(text string) []Diag` function.
4. Wire it into the `switch` in `debpkg/diag.go → Lint()`.
5. Create `debpkg/<name>.go` if field/keyword data is needed.
6. Add a completion branch in `server/completion.go → completion()`.
7. Add a hover branch in `server/hover.go → hover()`.
8. Add the glob pattern to `lua/debpack-lsp.lua → DEBIAN_PATTERNS`.

## How to add a new lint rule

1. Open the appropriate `debpkg/lint_<ft>.go`.
2. Write a helper that returns `[]Diag`.
3. Call it from the file's top-level `lint<FileType>` function and append results.
4. Coordinates are 0-indexed.  `SeverityError = 1`, `SeverityWarning = 2`.
5. Add a test case in `debpkg/debian_test.go` or a dedicated `_test.go`.

## How to extend the formatter

1. Write `debpkg.Format<FileType>(text string) string` in a new file
   `debpkg/format_<ft>.go`.  Make it idempotent.
2. Add a test file `debpkg/format_<ft>_test.go`.  Always include an idempotency
   test (apply twice, compare).
3. Add a `case debpkg.FileType<Name>:` to the switch in `server/format.go`.

## How to add a new bug source (tracker backend)

1. Create `bugs/<tracker>.go` implementing `bugs.BugSource`:
   - `Name() string`
   - `Prefix() string` (e.g. `"Closes: #"`)
   - `Bugs(pkg string) ([]Bug, error)`
   - `BugByID(pkg string, id int) (*Bug, error)`
2. Register it in `bugs.NewStore()` alongside the existing `newLaunchpadSource()`.

---

## Testing

```
go test ./...           # run all tests
go test -race ./...      # run with race detector (make test)
go test ./debpkg/ -v    # verbose debpkg tests only
go test ./server/ -v    # verbose server tests only
go test ./debhelper/ -v # verbose debhelper tests only
```

Tests live in `debpkg/`, `server/`, `debhelper/`, and `bugs/`.

When writing tests for `debpkg`, use package `debpkg_test` (black-box) or
`debpkg` (white-box, needed to access unexported helpers).  The formatter
tests in `format_changelog_test.go` use the `debpkg` package (white-box).

Server tests (`server/`) use package `server` (white-box).  Integration tests
in `server/integration_test.go` exercise the full handler pipeline
(initialize → didOpen → completion/hover/codeAction → didChange → assert
diagnostics) via a fake `glsp.Context`.  Unit tests in
`server/documents_test.go` cover `applyRangeChange` (incremental sync),
`splitLines`, and `lineColToOffset` including multi-byte edge cases.
`server/position_test.go` tests the `colToByteOffset` helper.
`server/workspace_test.go` tests the sibling-file cache and cross-file checks.
`server/lsp_features_test.go` tests folding, links, symbols, and quick-fix
helpers.

Debhelper tests (`debhelper/store_test.go`) include fixture-based parser
tests for the man-page scraper and a live integration test (skipped when
`man` is unavailable).

---

## Build and install

```
make build              # produces ./debpack-lsp binary
make install            # installs binary + Lua plugin (override: INSTALL_DIR=/usr/local/bin)
make install-lua        # installs only the Neovim Lua plugin (override: LUA_DIR=...)
make test               # go test -race ./...
make fmt                # gofmt -s -w .
make vet                # go vet ./...
make lint               # golangci-lint (if installed)
make vuln               # govulncheck (if installed)
make check              # vet + test
make clean              # removes ./debpack-lsp
```

Version is injected at build time via `-ldflags "-X main.version=$(VERSION)"`.
`VERSION` comes from `git describe --tags --always --dirty` or falls back to `"dev"`.

---

## Caches

| Cache | Location | TTL | Written by |
|---|---|---|---|
| Launchpad bugs | `~/.cache/lpad/<pkg>.json` | external (`lpad` tool) | `lpad sync` |
| debhelper commands | `~/.cache/debpack-lsp/dh-cmds.json` | 7 days | `debhelper.Store.Load()` |

The debhelper cache uses `schema_version` (currently 2); a mismatched version
is treated as stale, forcing a re-scrape from man pages.

The server never writes to the lpad cache.  If it is absent, bug completions
and hover return "not found in local cache" messages.

---

## Out of scope (deliberate non-features)

- **No network calls** — the server is fully offline; bug data comes from the
  lpad cache only.
- **No external lint libraries** — all rules are pure string/regex processing in Go.
- **No line rewrapping** in the formatter — too risky with LP: # refs, URLs, and
  package names.
- **No trailer date normalisation** — the trailer is dch-managed; the formatter
  passes it through verbatim.
- **No distribution/suite name validation** in changelog — out of scope.
- **No LP bug existence checks** — async bug-store warnings are out of scope.
- **No `Closes: #` completion** — the Debian BTS backend (`bugs/bts.go`) is a
  stub; completion is disabled to avoid offering Launchpad bugs for Debian issues.
- **No range/on-type formatting** — only whole-document formatting is registered.
- **No direct lint for `debian/compat`, `debian/source/format`,
  `debian/patches/series`** — they are read by the workspace cache for
  cross-file checks but not individually linted when opened directly.
- **`*.install`/`*.dirs`/`*.docs`/`*.links`/`*.manpages` are detected but
  receive only universal lint** — no file-specific completions, hover, or
  format support yet.
