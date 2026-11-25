# Search results UI

Type: prototype
Status: resolved
Assignee: boinkkitty
Blocked by:

## Question

How do search results display when the user types in the search box at the top of the left sidebar?
- Placement: a dropdown under the box, results replacing the Island rail, or a panel over the sheet.
- Each result shows the Day, the Notebook and the highlighted snippet. The data shape is fixed by [API contract and autosave](04-api-contract-and-autosave.md): 20 results newest first, snippet segments `{text, match}`, more by cursor.
- Keyboard: ⌘K or another shortcut to focus, arrow keys through results, Enter to open, Esc to clear and return to the rail.
- Empty query, no results, loading, and "load more" states.
- What happens on the opened Page: is the match highlighted or scrolled to?
- Narrow window: how search fits the slide-over drawer from [Island rail and hover expansion](06-island-rail-prototype.md).

## Answer

Settled with the user on 2026-11-25 in [search-results-PROTOTYPE.html](../prototypes/search-results-PROTOTYPE.html). The winner is **A with H2, blue matches, Title on, dates searched as text**, which the prototype opens on by default. B (dropdown), C (search page over the sheet) and D (⌘K spotlight) lost and stay in the file for reference.

- **Placement: results replace the Island rail** in the left sidebar while the query is non-empty, with the scrubber hidden. A small "Newest first" caption sits on top, and results are grouped under the rail's sticky month headers.
- **Each result**: the Day (`Today`, `Yesterday`, `Sun 22`) and the Notebook in small caps, then the **Title** in bold (My Journal only), then the snippet clamped to 3 lines with matches marked. When the query matches the date, the label shows the full date with the matching word marked ("Sun, 22 **November** 2026").
- **Results stay while you read.** Clicking one (or Enter) opens that Page and marks the row with a blue bar. The rail jumps to that Day in the background, loading only around it (per [Island rail and hover expansion](06-island-rail-prototype.md)).
- **Keyboard**: ⌘K focuses the search box (and opens the drawer in a narrow window). ↑ / ↓ move through results, and Enter opens one. **Esc clears** the query, and the rail comes back scrolled to the opened Day. A second Esc leaves the box.
- **States**:
  - **Empty query**: the rail as usual, with a `⌘K` hint in the box.
  - **First load**: shimmer rows.
  - **Loading a new query over old results**: a spinner in the box, and the old results dim.
  - **No results**: "No Pages match “…”", plus a hint that word starts and dates work.
  - **More**: infinite scroll by cursor. Arrowing near the end also loads more. At the end: "N Pages · that’s every match".
- **On the opened Page (H2)**: it scrolls so the first match is centred, with that match ringed. Every match stays highlighted, along with matching words in the date heading. A small "Matches for “…” · Clear · Esc" pill sits at the top of the sheet. Typing or Esc clears the highlights. The client marks matches itself using the same word-start rule. The flash-and-fade (H1) and scroll-only (H3) versions lost.
- **Match colour: blue** (`rgba(10,132,255,.17)`, ringed match `.38`), in results and on the Page, so search matches never look like the user's own yellow highlight formatting.
- **Dates are searched as text.** Each Page also matches on its Day's English date words: weekday, day number, month and year. So `nov`, `may`, `thursday`, `2025` and `nov mum` all work with the same AND + word-start rule, with no date parsing or filter syntax. The user rejected an interim version that turned date words into a filter: it made "may" and "march" ambiguous, and abbreviations needed a number next to them. If only the date matches, the snippet is the start of the Page with nothing marked.
- **Narrow window (D1 drawer)**: search lives in the drawer. Opening a result closes the drawer, like choosing a Day does. The results are still there when the drawer reopens.

**What this asks of other tickets:**
- [API contract and autosave](04-api-contract-and-autosave.md): each search result gains `title` (derived from `plain_text` like the rail's glimpses, empty for I Learned): `{day, notebook, title, snippet}`.
- [Data model](03-data-model.md): the search tsvector also covers the Day's date words. It is `to_tsvector('simple', zoop_day_words(day) || ' ' || coalesce(plain_text,''))`, where `zoop_day_words(date)` is a SQL function declared `IMMUTABLE` that returns `to_char(d, 'FMDay FMDD FMMonth YYYY')`. Without the `TM` prefix, `to_char` always gives English names, so declaring it immutable is safe.

No ADR: easy to reverse. No new glossary terms.
