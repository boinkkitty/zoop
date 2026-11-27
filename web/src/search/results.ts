// The search results list's rules, kept free of React so they can be tested on their own.
import type { SearchResult, Segment } from "../api";
import { islandLabel } from "../rail/rail";
import { matches } from "./match";

export type MonthOfResults = {
  month: string; // "2026-11"
  results: (SearchResult & { index: number })[]; // index: its place in the whole list
};

// byMonth groups results, already newest first, under the rail's month headers.
export function byMonth(results: readonly SearchResult[]): MonthOfResults[] {
  const out: MonthOfResults[] = [];
  results.forEach((r, index) => {
    const month = r.day.slice(0, 7);
    if (out.at(-1)?.month !== month) out.push({ month, results: [] });
    out.at(-1)?.results.push({ ...r, index });
  });
  return out;
}

// When ↓ comes this close to the end of what's loaded, the next page is fetched.
const LOAD_AHEAD = 3;

// moveActive is where ↑ or ↓ takes the active result among count, and whether to load more.
export function moveActive(
  active: number,
  count: number,
  key: "ArrowUp" | "ArrowDown",
): { active: number; loadMore: boolean } {
  const next =
    key === "ArrowDown"
      ? Math.min(count - 1, active + 1)
      : Math.max(0, active - 1);
  return {
    active: next,
    loadMore: key === "ArrowDown" && next >= count - LOAD_AHEAD,
  };
}

// endOfResults is the line under the last result once every match is loaded.
export const endOfResults = (count: number) =>
  `${count === 1 ? "One Page" : `${count} Pages`} · that’s every match`;

const WEEKDAYS = [
  "Sunday",
  "Monday",
  "Tuesday",
  "Wednesday",
  "Thursday",
  "Friday",
  "Saturday",
];
const MONTHS = [
  "January",
  "February",
  "March",
  "April",
  "May",
  "June",
  "July",
  "August",
  "September",
  "October",
  "November",
  "December",
];

// resultLabel names a result's Day as its Island does ("Today", "Sun 22"), or, when one of
// words matched the date, as the full date with those words marked: "Sun, 22 November 2026".
// The weekday is matched in full, like the server's date words, but shown short.
export function resultLabel(
  day: string,
  today: string,
  words: readonly string[],
): Segment[] {
  const d = new Date(`${day}T00:00:00Z`);
  const weekday = WEEKDAYS[d.getUTCDay()] ?? "";
  const parts: { show: string; word?: string }[] = [
    { show: weekday.slice(0, 3), word: weekday },
    { show: ", " },
    { show: String(d.getUTCDate()) },
    { show: " " },
    { show: MONTHS[d.getUTCMonth()] ?? "" },
    { show: " " },
    { show: String(d.getUTCFullYear()) },
  ];
  const marked = parts.map(
    (p) => p.show.trim() !== "" && matches(p.word ?? p.show, words).length > 0,
  );
  if (!marked.some(Boolean))
    return [{ text: islandLabel(day, today), match: false }];
  const out: Segment[] = [];
  parts.forEach((p, i) => {
    const match = marked[i] ?? false;
    const last = out.at(-1);
    if (last && last.match === match) last.text += p.show;
    else out.push({ text: p.show, match });
  });
  return out;
}
