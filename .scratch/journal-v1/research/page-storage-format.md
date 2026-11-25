# Research: how to store formatted Page content

Ticket: `issues/09-page-formatting.md` (the "storage format" question)
Researched: 2026-11-25, from first-party docs, engineering blogs, first-party source code (Automattic's Day One CLI) and forensic reverse-engineering of Apple Notes (Apple publishes no format docs). Items marked **(unverified)** were not confirmed on a primary page this session.

## Question

Zoop Pages (React + TipTap-or-Lexical editor, Go backend, Postgres) need bold, italic, underline, highlight, bullet / numbered / check lists nested up to 3 levels, and blockquote. They also need autosave, full-text search over plain text, export, and later edit history. Should the stored source of truth be Markdown, the editor's JSON document, or something else? How do Apple Notes, Google Docs and three note apps do it?

**Apps picked**: **Day One** (a journaling app, and it moved *from* Markdown storage *to* its own JSON, with the reasons written down), **Notion** (the best-documented block + annotated-span model, also on Postgres), and **Obsidian** (the purest "Markdown files are the source of truth" app, so it shows what Markdown storage costs). Bear, Evernote, Craft and Standard Notes were skipped: they are less documented, or (ENML) an XHTML variant that adds nothing these three don't cover.

## Comparison

| | Stored source of truth | Bold / italic | Underline | Highlight | Lists & checklists | Plain text for search | Export |
|---|---|---|---|---|---|---|---|
| **Apple Notes** | gzipped protobuf in `ZICNOTEDATA.ZDATA` (SQLite): one `note_text` string + `AttributeRun`s inside a CRDT ("topotext") [A1][A2][A3] | `font_weight` on the run (1 bold, 2 italic, 3 both) [A1][A3] | `underlined` flag on the run [A1] | `emphasis_style` 1-5 (purple, pink, orange, mint, blue) on the run [A3] | `ParagraphStyle.style_type` 100 dotted, 101 dashed, 102 numbered, 103 checkbox; `indent_amount`; `Checklist{uuid, done}`; `block_quote` [A1][A3] | `note_text` is already plain text, so reading it is just decoding [A3] | PDF, Markdown (macOS/iOS 26+), "Copy as Markdown" [A4] |
| **Google Docs** | revision log of changes (`InsertText`, `DeleteText`, `ApplyStyle` over a range), replayed to build the document; merged with OT [G1][G2] | `TextStyle.bold/italic` on a `TextRun` [G3][G4] | `TextStyle.underline` [G4] | `TextStyle.backgroundColor` [G4] | Paragraph has optional `Bullet{listId, nestingLevel}`; list presets include `BULLET_CHECKBOX` [G3][G5][G6] | text runs concatenated (the model is a plain string plus style ranges) [G3] | docx, odt, txt, **md**, pdf... [G7] |
| **Day One** | **`richTextJSON` (RTJ)**: an array of text nodes with inline + line attributes. Older entries: Markdown `body`. A feature flag says which one is canonical [D1][D2] | `bold`, `italic` attributes [D1] | **not supported** [D1][D3] | `highlightedColor` "RRGGBBAA" (typed `==x==` in Markdown) [D1][D3] | **flat lines** with `line.listStyle` (bulleted, numbered, checkbox), `indentLevel`, `checked`, `listIndex`, `identifier`; `line.quote` [D1] | Markdown `text` kept alongside RTJ in JSON export [D4]; the CLI plans search/embeddings off the markdown field [D5] | PDF, JSON (both forms), Plain Text/Markdown (lossy: "Export Plain Text strips all formatting") [D6] |
| **Notion** | one Postgres row per **block** (`type`, `properties`, `content[]` child ids, `parent`); text is an array of rich-text spans [N1][N2][N3] | span `annotations.bold/italic` [N2] | `annotations.underline` [N2] | `annotations.color` = `*_background` (e.g. `yellow_background`) [N2] | each item is a block (`bulleted_list_item`, `numbered_list_item`, `to_do{checked}`); nesting = child blocks; `quote` block [N1][N4] | each span has `plain_text`; blocks indexed into Elasticsearch by a "block → search document" function [N2][N5] | PDF, HTML, Markdown & CSV; callouts fall back to HTML [N6]. Their own Markdown dialect needs `<u>` and `<span color>` tags [N7] |
| **Obsidian** | **plain `.md` files in a folder** [O1] | `**` / `*` [O2] | **no Markdown syntax: raw HTML `<u>…</u>`** [O3] | `==…==` (non-standard; color via an emoji after `==`) [O2] | `- [ ]` / `- [x]`, nesting by indent [O2] | the file *is* the text, but search still sees the markup | the file is the export |

## Findings

### Apple Notes

- **Storage.** Notes live in `NoteStore.sqlite`. The note body is a zlib/gzip-compressed protobuf blob in `ZICNOTEDATA.ZDATA`; tables and drawings are in `ZMERGEABLEDATA` [A2]. The parser inflates it with gzip window bits (`Zlib::MAX_WBITS + 16`) [A3].
- **Model = attributed string.** The note is `note_text` (one plain string) plus `repeated AttributeRun attribute_run`. The runs are ordered and disjoint, and their lengths sum to the string length [A1][A2]. Each run carries `paragraph_style`, `font`, `font_weight`, `underlined`, `strikethrough`, `superscript`, `link`, `color`, `attachment_info` and `emphasis_style` [A3]. Ciofeca found this by making notes that changed one style at a time and checking that the run lengths added up to the text length [A1].
- **Underline / highlight.** Underline is a plain flag on the run [A1]. Highlight (the coloured "emphasis" styles added in recent iOS) is `emphasis_style` 1-5, which the parser renders as purple, pink, orange, mint and blue spans [A3].
- **Lists / quote.** These are paragraph-level attributes carried *on the runs*, not a tree: `style_type` (100 dotted, 101 dashed, 102 numbered, 103 checkbox), `indent_amount`, a `Checklist{uuid, done}` per checklist line, and `block_quote` [A1][A3]. Nesting is just an indent number.
- **Sync.** A CRDT that Apple calls "topotext". Field 3 of the String message holds sequence clocks, tombstones and children forming a DAG. Steve Dunham suspects it orders conflicts by sync order and is not a true CvRDT [A2]. A `RegisterLatest` object picks the latest version when edits arrive from several devices [A1].
- **Search.** The plain text is the `note_text` field itself, so no stripping is needed [A3]. Titles are stored in their own `ZTITLE1` column [A3]. (Spotlight / in-app index internals: unverified.)
- **Export.** PDF and Markdown export, "Copy as Markdown", and import from Markdown, TXT, RTF, HTML and ENEX [A4]. Markdown is an *interchange* format, not storage.
- **Lesson.** Apple keeps text and formatting apart: one string plus style runs. Plain text and search come for free, and formatting never pollutes the searchable text.

### Google Docs

- **Storage = operation log.** "A document is stored as a series of chronological changes ... we now compute the versions by playing forward the history of changes" [G1]. "All edits boil down to three basic types of changes: inserting text, deleting text, and applying styles to a range of text" [G2].
- **Styles are ranges.** `{ApplyStyle bold @10-20}` transformed against `{InsertText 'ABC' @15}` becomes `@10-23`. Different style types never conflict: a range can be red and italic at once [G2].
- **Lesson / regret.** The old Docs merged by *diffing versions*, which made John's bold plus Luiz's word replacement ambiguous ("fox dog", "dog", "dog fox"). Recording intent as operations fixed it [G1]. That only matters for real-time multi-writer editing.
- **API model (what Google exposes).** Body → `StructuralElement` → `Paragraph` (with `ParagraphStyle` and an optional `Bullet`) → `ParagraphElement`s → `TextRun` = "a contiguous string of text with all the same text style". Runs never cross paragraph boundaries. Indexes are in UTF-16 code units [G3].
- **Marks.** `TextStyle` has `bold`, `italic`, `underline`, `strikethrough` as booleans and **`backgroundColor` for highlight** [G4].
- **Lists.** A list is not a container. A paragraph *belongs* to a list through `Bullet{listId, nestingLevel}`, and the list's `nestingLevels[]` define the glyph per level [G4]. Presets include `NUMBERED_DECIMAL_ALPHA_ROMAN` (3 levels) [G5] and `BULLET_CHECKBOX` ("CHECKBOX bullet glyphs for all list nesting levels") [G6].
- **Export.** docx, odt, txt and `text/markdown` [G7].

### Day One (journaling, the closest analogue)

- **Moved off Markdown.** Automattic's (Day One's owner) RTJ spec: "Markdown provides many desired features and **was previously used as a storage format**. However, it does not support grouping photos into arbitrary collages or indenting an arbitrary paragraph independent of lists. RTJ addresses these gaps" [D1]. Its non-goals: "Full Markdown/HTML round-trip fidelity" [D1].
- **Two fields, one canonical.** Entries carry a Markdown `body` (older entries) and/or `richTextJSON`. "Only one of `body` or `richTextJSON` carries the canonical content at a given time". Feature flag `0x0010` means "content is stored as RTJ, not Markdown" [D2]. The public JSON export still includes both `richText` (RTJ as a string) and `text` (Markdown) per entry [D4].
- **RTJ shape** [D1]. `{meta:{version}, nodes|contents:[...]}`. A node is either `{text, attributes}` or `{embeddedObjects:[...]}`. Inline attributes: `bold`, `italic`, `strikethrough`, `inlineCode`, `highlightedColor` (RRGGBBAA), `linkURL`. **Line attributes** sit under `attributes.line`: `header`, `indentLevel`, `listStyle` ∈ {bulleted, numbered, checkbox}, `checked`, `listIndex`, `quote`, `codeBlock`, `identifier` ("e.g., checkbox IDs"). Only the first character's line attribute counts. Unknown attributes MUST be ignored, and new attributes don't bump `meta.version` [D1]. The spec says `nodes` but real exports use `contents` [D1][D4]. Even first-party formats drift.
- **Underline.** "At this time, underline is not supported". Highlight is typed as `==Highlight==`, and nested bullets come from indenting two spaces [D3].
- **Regret / pain.** The CLI's own architecture doc: "Converting between RTJson and Markdown is currently ad hoc and undertested ... the two formats have different expressive power (inline formatting, embedded objects, nested structure)" [D5]. When the CLI edits an entry that has meaningful rich text, it keeps the RTJ verbatim to avoid "corrupting formatted nodes" [D5]. Keeping two representations costs ongoing effort.
- **Storage practice.** Their local SQLite uses a "hybrid model": the full JSON blob for lossless round-trips plus extracted columns for querying [D5]. Server-side entries are end-to-end encrypted D1 blobs (decrypt, then gunzip, then JSON) [D2].
- **Export.** PDF, JSON (re-import), and Plain Text/Markdown that keeps `**bold**`, `*italic*`, `#` and `>` [D6].

### Notion

- **Block model.** "Everything you see in Notion is a block", with `id` (UUIDv4), `type`, `properties` (text lives in `title`), `content` (ordered child ids) and `parent` (for permissions) [N1]. Indentation is *structural*: indenting moves the block into the previous sibling's `content` [N1]. Changing type keeps properties: a to-do's `checked` survives a round trip through Heading [N1].
- **Rich text = annotated spans.** An array of `{type, text:{content, link}, annotations:{bold, italic, strikethrough, underline, code, color}, plain_text, href}`. Highlight is `color: "yellow_background"` and the like [N2]. Every span carries `plain_text`, "a convenient way ... to access unformatted text" [N2].
- **Lists.** `bulleted_list_item`, `numbered_list_item`, `to_do` (`checked`) and `quote` are block types. Nested items are `children` blocks [N4].
- **Persistence.** Edits are operations batched into transactions sent to `/saveTransactions`. After committing, the server schedules "version history snapshots and indexing block text for Quick Find" [N1]. History is snapshots, not the op log.
- **Lesson / regret.** One row per block "occupying a row in our database" produced "billions of new blocks". Postgres VACUUM stalled and TXID wraparound became "an existential threat", which forced app-level sharding [N3]. Search needs a whole separate pipeline whose "block → search document" code "encoded years of edge cases" [N5]. Notion's API Markdown dialect needs `<u>` "(the only underline)" and `<span color>` / `_bg` colors, because standard Markdown can't express its spans [N7]. Notion's own UI export also falls back to HTML for callouts [N6].

### Obsidian

- **Storage.** "Obsidian stores your notes as Markdown-formatted plain text files in a vault ... you can use other text editors" [O1]. The selling point is portability.
- **Marks.** `**bold**`, `*italic*`, `~~strike~~`, `==highlight==` with an optional colour emoji (`==🔴Important==`) [O2]. **Underline has no Markdown syntax: "you can use `<u>Example</u>`"**, and Markdown inside HTML is not rendered [O3].
- **Lists.** `- [ ]` / `- [x]` task lists, and any list type nests under any other by indenting [O2].
- **Lesson.** Pure Markdown works when users edit the *source* (Obsidian's audience). Underline and highlight come out as HTML and non-standard syntax, and every "Markdown" app parses them differently.

## What this means for Zoop

1. **Every WYSIWYG app stores structured text, not Markdown.** Apple Notes (runs), Google Docs (range ops → runs), Notion (spans in blocks) and Day One (RTJ) all do. Day One, the journaling app, explicitly *left* Markdown storage [D1]. Markdown is everyone's **export and interchange** format [A4][G7][N6][D6].
2. **Underline settles it.** Zoop's set includes underline. CommonMark has no underline: Obsidian and Notion's own Markdown fall back to `<u>` [O3][N7], and Day One drops underline entirely [D3]. Highlight (`==`) is non-standard too. Storing Markdown would mean Markdown-plus-HTML-plus-extensions, re-parsed on every load by both a JS and a Go parser that must agree.
3. **Your editor already produces a run model.** ProseMirror/TipTap JSON (marks on text nodes) and Lexical JSON (format bitmask on text nodes) are the same "plain text + style runs" idea as Apple, Google, Notion and Day One. Store what the editor emits, so there is no conversion step in the autosave path. (Editor JSON shapes: general knowledge, **unverified** this session.)
4. **Derive plain text once, on the server.** As in Notion's `plain_text` and Apple's `note_text`, the Go API walks the JSON on save. It writes a `plain_text` column (feeding the STORED tsvector, Island glimpses and Title) and rejects unknown node or mark types (an allowlist). The walker is a few dozen lines of Go.
5. **Edit history = snapshots, not ops.** Google's op log exists for real-time multi-writer OT [G1][G2]. Notion, which does have op transactions, still schedules *snapshots* for history [N1]. For a single-writer journal, write a revision row when autosave settles (e.g. coalesce to one per N minutes or per session).
6. **Version the document.** Day One's `meta.version` with "unknown attributes MUST be ignored" [D1] is the cheap insurance. Put `schemaVersion` in the stored doc so a later editor switch (TipTap ↔ Lexical) or new mark is a migration, not a guess.
7. **Don't do Notion's row-per-block.** Their sharding post shows the cost [N3]. Zoop needs no block addressing, so one JSONB document per Page is right.

## Takeaway

**Store the editor's JSON document (one `JSONB` per Page, with a `schemaVersion`) as the source of truth, plus a server-derived `plain_text` column for search, glimpses and Title. Generate Markdown only on export.**

- Markdown loses: no underline (it becomes `<u>`), non-standard highlight, and two parsers that must agree. Day One, the one journaling app here, abandoned Markdown storage and still pays for keeping two formats [D1][D5].
- Editor JSON *is* the attributed-text-runs pattern that Apple Notes, Google Docs, Notion and Day One all use, so you get the same benefits without inventing a format.
- Go validates and walks the JSON on save. Its node allowlist is paragraph, bulletList, orderedList, taskList (checked), listItem (depth ≤ 3) and blockquote; its mark allowlist is bold, italic, underline and highlight. It produces `plain_text` for `tsvector`, and it also writes Markdown export (underline exported as `<u>`, highlight as `==`, accepted as lossy, as everyone does).
- History is a `page_revisions` table of JSON snapshots taken when autosave settles. No op log, since Zoop has one writer.
- Hedge: pick TipTap vs Lexical *before* the first real Page is saved, because their JSON differs. If you want to stay editor-neutral, a small Day One-style RTJ of your own is the fallback, at the price of a converter on every load and save.

## Sources

- [A1] Ciofeca Forensics, "Apple Notes Revisited: Protobuf" (2020): https://ciofecaforensics.com/2020/09/18/apple-notes-revisited-protobuf/
- [A2] dunhamsteve/notesutils, `notes.md`: https://github.com/dunhamsteve/notesutils/blob/master/notes.md
- [A3] threeplanetssoftware/apple_cloud_notes_parser (`proto/notestore.proto`, `lib/ProtoPatches.rb`, `lib/AppleNote.rb`, `lib/AppleNoteStore.rb`; HEAD 2026-07-25): https://github.com/threeplanetssoftware/apple_cloud_notes_parser
- [A4] Apple Support, "Import, export, and print notes on Mac": https://support.apple.com/guide/notes/import-export-and-print-notes-not201900c07/mac
- [G1] Google Drive Blog, "What's different about the new Google Docs: Working together, even apart" (2010): https://drive.googleblog.com/2010/09/whats-different-about-new-google-docs_21.html
- [G2] Google Drive Blog, "... Conflict resolution" (2010): https://drive.googleblog.com/2010/09/whats-different-about-new-google-docs_22.html
- [G3] Google Docs API, "Structure of a Google Docs document": https://developers.google.com/workspace/docs/api/concepts/structure
- [G4] Google Docs API, `documents` resource (TextStyle, Bullet, NestingLevel): https://developers.google.com/workspace/docs/api/reference/rest/v1/documents
- [G5] Google Docs API, "Work with lists": https://developers.google.com/workspace/docs/api/how-tos/lists
- [G6] Google Docs API, Requests (BulletGlyphPreset `BULLET_CHECKBOX`): https://developers.google.com/workspace/docs/api/reference/rest/v1/documents/request
- [G7] Google Drive API, export MIME types: https://developers.google.com/workspace/drive/api/guides/ref-export-formats
- [D1] Automattic/DayOne-Cli, `docs/rtjson-format.md` (RTJSON spec v1, updated 2026-04-01): https://github.com/Automattic/DayOne-Cli/blob/main/docs/rtjson-format.md
- [D2] Automattic/DayOne-Cli, `docs/entry-body-format.md`: https://github.com/Automattic/DayOne-Cli/blob/main/docs/entry-body-format.md
- [D3] Day One, "Markdown Guide": https://dayoneapp.com/guides/tips-and-tutorials/markdown-guide/
- [D4] Example Day One JSON export (shows `richText` + `text` per entry; third-party gist of a first-party file): https://gist.github.com/extratone/2631adf503220a59b39ba1af5400f4be
- [D5] Automattic/DayOne-Cli, `docs/architecture.md` and `docs/entry-sql-schema.md`: https://github.com/Automattic/DayOne-Cli/blob/main/docs/architecture.md
- [D6] Day One, "Exporting entries": https://dayoneapp.com/guides/tips-and-tutorials/exporting-entries/
- [N1] Notion blog, "The data model behind Notion's flexibility" (2021): https://www.notion.com/blog/data-model-behind-notion
- [N2] Notion API, "Rich text": https://developers.notion.com/reference/rich-text
- [N3] Notion blog, "Herding elephants: Lessons learned from sharding Postgres at Notion": https://www.notion.com/blog/sharding-postgres-at-notion
- [N4] Notion API, "Block": https://developers.notion.com/reference/block
- [N5] Notion blog, "Rebuilding Notion's lexical search reindexer": https://www.notion.com/blog/rebuilding-notions-lexical-search-reindexer
- [N6] Notion Help, "Export your content": https://www.notion.com/help/export-your-content
- [N7] Notion API, "Notion-flavored Markdown format": https://developers.notion.com/guides/data-apis/notion-flavored-markdown
- [O1] Obsidian Help, "How Obsidian stores data": https://help.obsidian.md/data-storage
- [O2] Obsidian Help, "Basic formatting syntax": https://help.obsidian.md/syntax
- [O3] Obsidian Help, "HTML content" (Underline section): https://help.obsidian.md/html (source: https://github.com/obsidianmd/obsidian-help)
