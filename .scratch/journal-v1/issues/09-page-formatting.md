# Formatting on Pages

Type: grilling
Status: resolved
Assignee: boinkkitty
Blocked by:

## Question

The user wants formatting on Pages, at least bold, italic and bullet points, which overturns the map's settled "Pages are plain text". Decide:
- **The v1 set.** Suggested: bold, italic, strikethrough, highlight, bullets, numbered lists, checklists, quote. Candidates to rule in or out: underline, divider line, links, headings. Photos are proposed as out of scope for v1 because of storage and upload cost.
- **The storage format.** Markdown as the source of truth (readable, editor-independent, easy to export and search) or the editor's JSON document.
- **How formatting is applied.** Keyboard shortcuts (⌘B/⌘I), Markdown typing shortcuts (`- `, `1. `, `[] `, `> `), and/or a pop-up bar on text selection. No permanent toolbar, to keep the minimal look.
- **What gets indexed for search and shown in Island glimpses and the Title.** These need the plain-text form, with formatting stripped.

Context: the session that closed [Journal page look and feel](05-journal-page-prototype.md) raised the options. The prototype for this ticket can build on [journal-page-v3-PROTOTYPE.html](../prototypes/journal-page-v3-PROTOTYPE.html).

## Answer

Settled with the user on 2026-11-25 over three grilling rounds. The storage question was settled using [page-storage-format.md](../research/page-storage-format.md), which compares Apple Notes, Google Docs, Day One, Notion and Obsidian.

- **v1 set**: bold, italic, underline, highlight; bullet lists, numbered lists, checklists; quote. Lists nest up to **3 levels** with Tab / ⇧Tab, and list types can be mixed when nested. **Not in v1**: strikethrough, divider, links, headings, photos, fonts, colours and sizes. Divider, links and headings are parked to consider later.
- **Storage**: the source of truth is **TipTap (ProseMirror) JSON**, one JSONB document per Page with a `schemaVersion`. Markdown is **export-only**: underline becomes `<u>` and highlight becomes `==`, which is slightly lossy, as it is in every app compared.
  - Recorded as [ADR 0001](../../../docs/adr/0001-pages-stored-as-tiptap-json.md).
  - Why not Markdown: Markdown has no syntax for underline and no standard one for highlight. Every rich-text app compared stores plain text plus style ranges. Day One, the one journaling app, moved away from storing Markdown.
- **Server checks on save**: the Go server rejects any node or mark type outside the v1 set and any list nested deeper than 3 levels. It then derives a **plain-text column**: formatting is removed, list and checkbox markers are dropped, and each block becomes one line. Search (tsvector), Island glimpses and the Title read that column, never the JSON.
- **Edit history later**: whole JSON copies of the Page saved in their own table when autosave settles. A Page has one writer, so no change log is needed.
- **Applying formatting**: **keyboard shortcuts only**. There is no toolbar, no pop-up bar on selection, and typed Markdown (`- `, `**x**`) stays as literal characters.

  | Format | Shortcut |
  |---|---|
  | Bold / Italic / Underline | ⌘B / ⌘I / ⌘U |
  | Highlight | ⌘⇧H |
  | Bullet list / Numbered list / Checklist | ⌘⇧8 / ⌘⇧7 / ⌘⇧L |
  | Quote | ⌘⇧9 |
  | Indent / outdent in a list | Tab / ⇧Tab |
  | Toggle a checkbox | ⌘Enter, or click |

  Pressing a block shortcut a second time turns it back into a paragraph. Pressing Enter on an empty list item ends the list. On Windows and Linux, Ctrl takes the place of ⌘.
- **Discoverability**: **⌘/ opens a shortcut sheet** (a quiet overlay). The sheet is also reachable from a sidebar item and mentioned once in the first-run experience. No hints appear on the Page itself.
- **Title**: the first block of the *My Journal* Page is **always a plain paragraph**. Inline marks are allowed in it, block formats are not. The Island shows the Title in plain text.
- **Island hover glimpse**: plain text only, with no checkbox or list markers.
- **Pasting**: only formatting in the v1 set is kept, and everything else is dropped. Link text stays, without the link. ⌘⇧V pastes as plain text.
