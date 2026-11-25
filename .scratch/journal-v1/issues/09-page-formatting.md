# Formatting on Pages

Type: grilling
Status: open
Blocked by:

## Question

The user wants formatting on Pages, at least bold, italic and bullet points, which overturns the map's settled "Pages are plain text". Decide:
- **The v1 set.** Suggested: bold, italic, strikethrough, highlight, bullets, numbered lists, checklists, quote. Candidates to rule in or out: underline, divider line, links, headings. Photos are proposed as out of scope for v1 because of storage and upload cost.
- **The storage format.** Markdown as the source of truth (readable, editor-independent, easy to export and search) or the editor's JSON document.
- **How formatting is applied.** Keyboard shortcuts (⌘B/⌘I), Markdown typing shortcuts (`- `, `1. `, `[] `, `> `), and/or a pop-up bar on text selection. No permanent toolbar, to keep the minimal look.
- **What gets indexed for search and shown in Island glimpses and the Title.** These need the plain-text form, with formatting stripped.

Context: the session that closed [Journal page look and feel](05-journal-page-prototype.md) raised the options. The prototype for this ticket can build on [journal-page-v3-PROTOTYPE.html](../prototypes/journal-page-v3-PROTOTYPE.html).
