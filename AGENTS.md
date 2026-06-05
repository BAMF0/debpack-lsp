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
main            Thin entrypoint. Parses --version flag, calls server.New().Run().
server/         LSP wiring. All protocol handlers, the document store, async
                loading of bug and debhelper caches.
debpkg/         All Debian-file domain logic: file-type detection, lint rules,
                formatter, completion trigger detection, similarity ranking.
                MUST NOT import LSP types — it is LSP-free by design.
bugs/           Bug tracker backends. Currently only a Launchpad source that
                reads ~/.cache/lpad/<pkg>.json (no network calls). Exposes a
                Store with All()/ByID() under a RWMutex.
debhelper/      Scrapes installed dh_* binaries from $PATH once, caches for
                7 days to ~/.cache/debpack-lsp/dh-cmds.json. Exposes a Store
                with All()/ByName().
internal/log    Logging only (ilog.Logf). No external dependencies.
```

---

## File map

### `debpkg/`

| File | Purpose |
|---|---|
| `filetype.go` | `FileType` enum; `FileTypeFromURI(uri)` — entry point for all file-type detection |
| `diag.go` | `Diag`, `Severity`, `LintContext`; `Lint()` dispatcher; universal pre-pass (trailing whitespace, whitespace-only blank lines); `splitLines`, `isBlank`, `countLeadingSpaces` helpers |
| `changelog.go` | `PackageFromChangelog`, `IsUbuntuChangelog`, `BugRefAtCursor`, `BugNumberAtOffset`, `ContextBeforeBugRef`, `BugNumbersInText`, `ChangelogContextForBugRef`, `ChangelogFixesBulletAtLine` |
| `similarity.go` | `Tokenize` (lowercase, split non-alphanum, drop <3 chars and stop words); `TitleSimilarity` (token substring match count) |
| `control.go` | `KnownControlFields`, `LookupField`, `FieldAtCursor`, `FieldNameFromLine`, `knownSections`, `knownArchitectures` |
| `copyright.go` | `KnownCopyrightFields`, `LookupCopyrightField`, `knownSPDXLicenses` |
| `patches.go` | `KnownPatchFields`, `LookupPatchField` |
| `watch.go` | `KnownWatchOptions`, `IsInOpts` |
| `lint_changelog.go` | All changelog lint rules: header format, urgency, blank-line structure, trailer format, body indentation, contributor blocks |
| `lint_control.go` | All control lint rules: mandatory fields, `Standards-Version` format, unknown fields (X-/XB-/XC-/XS- exempted), enumerated value checking, Ubuntu Maintainer consistency |
| `lint_copyright.go` | DEP-5 rules: `Format:` presence/URI, stanza completeness, catch-all `Files: *` |
| `lint_watch.go` | Watch file version check — valid versions are 4 and 5 only |
| `lint_patch.go` | DEP-3 rules: `Description`/`Subject` required, `Origin` validation, `Forwarded` value, `Last-Update` format, line length |
| `format_changelog.go` | `FormatChangelog(text) string` — idempotent changelog formatter |
| `debian_test.go` | Tests for `FileTypeFromURI`, `PackageFromChangelog`, `BugRefAtCursor`, `BugNumberAtOffset` |
| `format_changelog_test.go` | 10 black-box tests for `FormatChangelog` |

### `server/`

| File | Purpose |
|---|---|
| `server.go` | `Server` struct; `New()`, `Run()`; handler registration; `maybeLoadBugs`; `strPtr` helper |
| `documents.go` | Thread-safe `documentStore` (open/close/get/applyChanges); `applyRangeChange`; its own `splitLines` (different semantics — see below) |
| `completion.go` | All completion handlers; `changelogCompletions` dispatcher; `changelogNumberCompletions`; `changelogTitleCompletions`; `controlCompletions`; `rulesCompletions`; `copyrightCompletions`; `patchCompletions`; `watchCompletions`; `lineUpToCursor`, `fullLineAt`, `stringsToCompletions` |
| `hover.go` | All hover handlers; `bugMarkdown`, `dhMarkdown`; `lineColToByteOffset`, `wordAtCol` |
| `diagnostics.go` | `publishDiagnostics` (lint → `textDocument/publishDiagnostics`); `clearDiagnostics` |
| `format.go` | `format` handler — fetches text, calls `debpkg.FormatChangelog`, returns whole-document `TextEdit` if text changed |

### `bugs/`

| File | Purpose |
|---|---|
| `source.go` | `Bug`, `BugSource` interface, `Store` (Load/All/ByID), `NewStore` |
| `launchpad.go` | `launchpadSource` — reads `~/.cache/lpad/<pkg>.json`; no network |
| `bts.go` | Stub for future Debian BTS support |

### Root

| File | Purpose |
|---|---|
| `main.go` | Entrypoint; `--version` flag; calls `server.New().Run()` |
| `Makefile` | `build`, `install` (`INSTALL_DIR` default `~/.local/bin`), `test`, `clean`, `uninstall` |
| `lua/debpack-lsp.lua` | Neovim companion plugin; uses `vim.lsp.start` directly (no nvim-lspconfig needed) |
| `go.mod` | Module `github.com/BAMF0/debpack-lsp`; Go 1.25; main dependency: `github.com/tliron/glsp v0.2.2` |

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
FileTypePatch      → patchCompletions(lineUpTo)
FileTypeWatch      → watchCompletions(lineUpTo)
```

#### Changelog completions — three-branch priority order

1. **LP: # number completion** (`changelogNumberCompletions`) — fires when
   `debpkg.BugRefAtCursor(lineUpTo)` returns a non-empty prefix (`"LP: #"` or
   `"Closes: #"`). Ranks bugs by `TitleSimilarity` using
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
go test ./debpkg/ -v    # verbose debpkg tests only
```

Tests currently live in `debpkg/`.  There are no server-level tests.

When writing tests for `debpkg`, use package `debpkg_test` (black-box) or
`debpkg` (white-box, needed to access unexported helpers).  The formatter
tests in `format_changelog_test.go` use the `debpkg` package (white-box).

---

## Build and install

```
make build              # produces ./debpack-lsp binary
make install            # installs to ~/.local/bin (override: INSTALL_DIR=/usr/local/bin make install)
make test               # go test ./...
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
