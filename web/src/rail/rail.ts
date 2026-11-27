// The Island rail's rules, kept free of React so they can be tested on their own.
// Days are calendar dates ("YYYY-MM-DD"), so they're handled in UTC: never shifted by the browser's zone.

const DAY_MS = 24 * 3600 * 1000;

const dayDate = (day: string) => new Date(`${day}T00:00:00Z`);

const weekdayDay = new Intl.DateTimeFormat("en-GB", {
  weekday: "short",
  day: "numeric",
  timeZone: "UTC",
});

// islandLabel is what an Island says before its Title: "Today", "Yesterday", then "Mon 23".
export function islandLabel(day: string, today: string): string {
  const daysAgo = Math.round(
    (dayDate(today).getTime() - dayDate(day).getTime()) / DAY_MS,
  );
  if (daysAgo === 0) return "Today";
  if (daysAgo === 1) return "Yesterday";
  return weekdayDay.format(dayDate(day));
}

// graceLeft is how many minutes (rounded up) yesterday stays open, while a Free
// User is in the Grace Window at now (ms); otherwise null.
export function graceLeft(
  me: { plan: "free" | "premium"; graceUntil: string | null },
  now: number,
): number | null {
  if (me.plan !== "free" || me.graceUntil === null) return null;
  const left = Date.parse(me.graceUntil) - now;
  return left > 0 ? Math.ceil(left / 60_000) : null;
}

export type MonthCount = { month: string; writtenDays: number };

// A scrubber slot: one month ("2026-11"), one older year ("2024") or, in a very
// long history, one decade ("1990s"). written is the share of its Days that were
// written (0..1), which sets the tick's width.
export type ScrubSlot = {
  label: string;
  unit: "month" | "year" | "decade";
  year: number; // the slot's newest year
  months: MonthCount[]; // newest first; a year's or decade's open the month grid
  written: number;
};

// Slots are at most MAX_SLOT px tall; below MIN_SLOT the oldest years merge into decades.
const MAX_SLOT = 18;
const MIN_SLOT = 6;

export const yearOf = (month: string) => Number(month.slice(0, 4));
const decadeOf = (year: number) => Math.floor(year / 10) * 10;

// daysInMonth counts the Days in a "YYYY-MM" month.
export const daysInMonth = (month: string) =>
  new Date(Date.UTC(yearOf(month), Number(month.slice(5, 7)), 0)).getUTCDate();

const daysInYears = (from: number, to: number) =>
  (Date.UTC(to, 0, 1) - Date.UTC(from, 0, 1)) / DAY_MS;

function slotOf(
  label: string,
  unit: ScrubSlot["unit"],
  year: number,
  months: MonthCount[],
): ScrubSlot {
  const days =
    unit === "month"
      ? daysInMonth(label)
      : unit === "year"
        ? daysInYears(year, year + 1)
        : daysInYears(decadeOf(year), decadeOf(year) + 10);
  const written = months.reduce((n, m) => n + m.writtenDays, 0);
  return { label, unit, year, months, written: written / days };
}

// scrubber lays out the whole history (newest first, as GET /days/summary gives it) in
// height px: months for this calendar year and last, then one slot per older year. It
// never scrolls: slots shrink to fit, and if they'd get too small the oldest years merge into decades.
export function scrubber(
  months: MonthCount[],
  today: string,
  height: number,
): { slots: ScrubSlot[]; slot: number } {
  const cut = yearOf(today) - 1;
  const recent = months
    .filter((m) => yearOf(m.month) >= cut)
    .map((m) => slotOf(m.month, "month", yearOf(m.month), [m]));

  const byYear = new Map<number, MonthCount[]>();
  for (const m of months) {
    const y = yearOf(m.month);
    if (y < cut) byYear.set(y, [...(byYear.get(y) ?? []), m]);
  }
  let older = [...byYear].map(([y, ms]) => slotOf(String(y), "year", y, ms));

  // Merge the oldest decade at a time until the slots are tall enough.
  while (height / (recent.length + older.length) < MIN_SLOT) {
    const oldest = older.filter((s) => s.unit === "year").at(-1);
    if (!oldest) break;
    const d = decadeOf(oldest.year);
    const inDecade = older.filter((s) => decadeOf(s.year) === d);
    const newest = Math.max(...inDecade.map((s) => s.year));
    const merged = slotOf(
      `${d}s`,
      "decade",
      newest,
      inDecade.flatMap((s) => s.months),
    );
    older = [...older.filter((s) => !inDecade.includes(s)), merged].sort(
      (a, b) => b.year - a.year,
    );
  }

  const slots = [...recent, ...older];
  return { slots, slot: Math.min(MAX_SLOT, height / slots.length) };
}
