# Journal page look and feel

Type: prototype
Status: resolved
Blocked by:

## Question

What should the main writing surface look like so it feels like a physical journal and not a generic app? Cover:
- the large white lined page
- the typeface(s)
- the bold first-line Title
- the Notebook tab divider on the side (all one colour, room for more than two)
- the date header
- how the page looks when it is read-only (a past Day)

Make a rough clickable prototype to react to.

## Answer

Settled with the user on 2026-11-25 over three rounds. The winner is **T4** in [prototypes/journal-page-v3-PROTOTYPE.html](../prototypes/journal-page-v3-PROTOTYPE.html) (it opens on T4 by default). Earlier rounds: [round 1, three variants](../prototypes/journal-page-PROTOTYPE.html), where the single sheet on a desk won; [round 2](../prototypes/journal-page-v2-PROTOTYPE.html).

**Look**: minimal, modern, Apple-style. Use the system font (SF), a light grey background (`#f5f5f7`) and one white sheet with rounded corners (~18px) and a soft shadow. The sheet is up to ~860px wide and fills the window's height, with generous padding.
- **No ruled lines** and no margin line. This replaces the "lined page" in the map's original Feel note.
- **Date header**: a small grey weekday line ("Wednesday · Today") and the date in semibold at about 19px. The user found anything larger too big. Status labels sit on the weekday line as small pills: "Read only" on past Days, "Still open until 02:00" on yesterday during the Grace Window.
- **Title**: the first line of My Journal is bold and a little larger (~21px). An empty Page shows a grey "Title" placeholder. The body text is ~17px with a line height of 1.65.
- **Read-only past Day**: the text is slightly muted, there's no caret, and the "Read only" label shows.

**Notebooks**: an inset index column inside the sheet's right edge (T2), ~52px wide behind a hairline. Labels are small, uppercase and vertical. The active Notebook is black with a 2px black edge mark, and a faint "+" leaves room for more.

**Left sidebar** (300px), from top to bottom:
1. Placeholder nav rows: Home, Filler 1, Filler 2. The real items are not decided yet.
2. The search field.
3. A hairline divider.
4. The Island rail, which scrolls on its own.

**Islands** (the first pass at the rail; the Island rail ticket owns the details):
- Each Island is a black pill as wide as the rail, showing `date │ Title` with a faint vertical divider. The Title truncates with "…", which needs `min-width: 0` on the flex child.
- Empty Days are light-grey pills showing only the date.
- On hover, after a **150ms delay**, the Island expands **in place and smoothly pushes the Islands below it down**. Collapsing is instant.
- Days load lazily in batches (40 in the prototype), with shimmering placeholder pills near the bottom.

**Still open after this ticket**:
- Inline formatting, which became its own ticket: Formatting on Pages.
- What the sidebar nav items really are.
- Rail order (newest first or oldest first).
