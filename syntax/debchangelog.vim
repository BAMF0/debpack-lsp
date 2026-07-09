" debpack-lsp custom syntax for debian/changelog.
" Highlights:
"   - Header: package name, version, suite, urgency
"   - Bullet markers (*, -, +) — color varies by level
"   - Labels: text ending with a colon — color varies by bullet level
"     * Label:  → Title    (top-level)
"       - Label: → Identifier (sub-bullet)
"         + Label: → Special  (third-level)
"   - LP: #N and Closes: #N bug references (entire ref including digits)
"     Works on bullet lines AND continuation lines.
"   - Trailer line (-- Name <email>  Date)
"   - Non-label bullet text and continuation lines: no highlight

if exists('b:current_syntax')
  finish
endif

syn case ignore

" --- Header line: package (version) suite; urgency=value ---
syn region debchangelogHeader start="^\S" end="$" oneline contains=debchangelogPackage,debchangelogVersion,debchangelogSuite,debchangelogUrgency
syn match debchangelogPackage  contained "^\S\+\ze\s\+("  display
syn match debchangelogVersion  contained "(.\{-})"  display
syn match debchangelogSuite    contained ")\s\+\zs\S\+\ze\s*;"  display
syn match debchangelogUrgency  contained "urgency=\zs\S\+"  display

" --- Trailer line: -- Name <email>  Date ---
syn match debchangelogTrailer  "^\s\+--\s.\+$"

" --- Bug references (standalone — matches on continuation lines) ---
syn match debchangelogLP      /\clp:\s*#\d\+\%(,\s*#\d\+\)*/
syn match debchangelogCloses  /\ccloses:\s*#\d\+\%(,\s*#\d\+\)*/

" --- Top-level bullet:   * text ---
syn region debchangelogEntryStar  start=/^\s\+[*]\s/ end=/$/ oneline contains=debchangelogBulletStar,debchangelogLabelStar,debchangelogBugStar
syn match debchangelogBulletStar  /[*]\s\+/  contained containedin=debchangelogEntryStar
syn match debchangelogLabelStar   /[A-Za-z0-9 /._-]\+\ze:/  contained containedin=debchangelogEntryStar

" --- Sub-bullet:    - text ---
syn region debchangelogEntryDash  start=/^\s\+[-]\s/ end=/$/ oneline contains=debchangelogBulletDash,debchangelogLabelDash,debchangelogBugDash
syn match debchangelogBulletDash  /[-]\s\+/  contained containedin=debchangelogEntryDash
syn match debchangelogLabelDash   /[A-Za-z0-9 /._-]\+\ze:/  contained containedin=debchangelogEntryDash

" --- Third-level:      + text ---
syn region debchangelogEntryPlus  start=/^\s\+[+]\s/ end=/$/ oneline contains=debchangelogBulletPlus,debchangelogLabelPlus,debchangelogBugPlus
syn match debchangelogBulletPlus  /[+]\s\+/  contained containedin=debchangelogEntryPlus
syn match debchangelogLabelPlus   /[A-Za-z0-9 /._-]\+\ze:/  contained containedin=debchangelogEntryPlus

" --- Bug references (contained, defined AFTER labels so they win at LP: positions) ---
syn match debchangelogBugStar   /\clp:\s*#\d\+\%(,\s*#\d\+\)*/  contained containedin=debchangelogEntryStar
syn match debchangelogBugStar   /\ccloses:\s*#\d\+\%(,\s*#\d\+\)*/  contained containedin=debchangelogEntryStar
syn match debchangelogBugDash   /\clp:\s*#\d\+\%(,\s*#\d\+\)*/  contained containedin=debchangelogEntryDash
syn match debchangelogBugDash   /\ccloses:\s*#\d\+\%(,\s*#\d\+\)*/  contained containedin=debchangelogEntryDash
syn match debchangelogBugPlus   /\clp:\s*#\d\+\%(,\s*#\d\+\)*/  contained containedin=debchangelogEntryPlus
syn match debchangelogBugPlus   /\ccloses:\s*#\d\+\%(,\s*#\d\+\)*/  contained containedin=debchangelogEntryPlus

" --- Colors ---
hi def link debchangelogHeader      Normal
hi def link debchangelogPackage     Title
hi def link debchangelogVersion     Number
hi def link debchangelogSuite       String
hi def link debchangelogUrgency     Keyword
hi def link debchangelogTrailer     Comment

" Labels: distinct color per bullet level
hi def link debchangelogLabelStar   Title
hi def link debchangelogLabelDash   Identifier
hi def link debchangelogLabelPlus   Special

" Bullets: distinct color per level
hi def link debchangelogBulletStar  SpecialKey
hi def link debchangelogBulletDash  SpecialKey
hi def link debchangelogBulletPlus  SpecialKey

" Bug refs: all the same color (standalone + contained variants)
hi def link debchangelogLP          Statement
hi def link debchangelogCloses      Statement
hi def link debchangelogBugStar     Statement
hi def link debchangelogBugDash     Statement
hi def link debchangelogBugPlus     Statement

hi def link debchangelogEntryStar   Normal
hi def link debchangelogEntryDash   Normal
hi def link debchangelogEntryPlus   Normal

let b:current_syntax = 'debchangelog'
