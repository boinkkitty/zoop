# Island rail and hover expansion

Type: prototype
Status: resolved
Assignee: boinkkitty
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

## Answer

Settled with the user on 2026-11-25 over four rounds in [island-rail-PROTOTYPE.html](../prototypes/island-rail-PROTOTYPE.html). The winner is **R4 with scrubber S3**, which the prototype opens on by default. The other variants stay in the file for reference.

- **Order**: newest first. Today is at the top, and older Days load in batches as you scroll down. Month headers ("NOVEMBER 2026") stick to the top of the rail while you scroll through that month. No month-picker popover.
- **Island labels**: `Today`, `Yesterday`, then weekday + day number (`Mon 23`) under the month header, then `│ Title`. Empty Days stay grey pills with only the date.
- **Jump-to-month: a scrubber** down the rail's right edge (~22px wide), newest at the top.
  - It shows one tick per month for **this calendar year and last year**. Hover or drag shows a bubble naming the month ("February 2026"), and releasing or clicking jumps there.
  - Below those, **one segment per older year**. Hovering a year opens a small 12-month grid beside the scrubber, with a bar under each month showing how much was written. Click a month to jump.
  - A tick's width shows how much was written that month. Year labels (`’25`) sit at year boundaries and thin out so they never overlap. A small black dot marks the month currently in view.
  - **It never scrolls.** Slots are at most 18px and shrink so the whole history always fits the rail: ~11px at 25 years in a normal window, ~7px in a short one. The user tried a scrolling version and ruled it out: it becomes a second scrollbar inside the rail, the wheel scrolls the wrong thing, and it hides the oldest years. If it ever ran out of room (~40–50 years), the fallback is to merge the oldest years into decades, never to scroll.
- **Jumping loads only the target**: the rail shows a window starting at the target month and loads newer Days as you scroll up and older Days as you scroll down. It never loads the Days in between. At 25 years of history, a jump in the prototype kept 80 of 9,453 Days in the DOM, so no virtualisation library is needed in v1.
- **Grace Window marker**: a pulsing green dot at the right end of yesterday's Island. The expanded Island adds "Still open until 02:00 · 48 min left". The ring and draining-bar markers lost.
- **Hover**: the 150ms delay already on the map. Motion is the **Spring** curve, `cubic-bezier(.32,1.25,.5,1)` over 0.45s, chosen in a side-by-side comparison against gentle spring, soft, snappy and none. Expansion pushes the Islands below it down, and collapsing is instant.
- **Narrow window**: **D1 slide-over**. A menu button at the top-left of the page slides the whole sidebar in over the sheet, with a light scrim. Choosing a Day, clicking the scrim or pressing Esc closes it. The D2 mini rail lost.

**What this asks of the API** (passed on to [API contract and autosave](04-api-contract-and-autosave.md)):
- Fetch a page of Days **starting from any date, in either direction**, not only "the next batch after the last one loaded".
- A **per-month summary across the whole history** (written Days per month), so the scrubber's tick widths and the year grid's bars render without loading the Days themselves.
