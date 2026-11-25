# Island rail and hover expansion

Type: prototype
Status: open
Blocked by: 05

## Question

How do the thin Day Islands on the left rail look and behave?
- What does an Island show at rest (date + Title)?
- How does the vertical expansion on hover animate, and what does it reveal (one row per non-empty Notebook)?
- How do faded empty Days look?
- How does the "still open" Grace Window marker look?
- How does the rail scroll?

Build it on top of the journal page prototype.

## Starting point (2026-11-25)

Much of the rail was settled while closing [Journal page look and feel](05-journal-page-prototype.md). See its Answer and T4 in [journal-page-v3-PROTOTYPE.html](../prototypes/journal-page-v3-PROTOTYPE.html):
- full-width black pills showing `date │ Title` with an ellipsis
- expansion that pushes neighbours down, after a 150ms hover delay
- lazy loading in batches
- grey stubs for empty Days

Still open here:
- rail order (newest or oldest first)
- jump-to-month
- the look of the Grace Window "still open" marker
- the drawer in a narrow window
- animation tuning
