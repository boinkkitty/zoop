import { describe, expect, it } from "vitest";
import { graceLeft, islandLabel, scrubber } from "./rail";

describe("an Island's label", () => {
  it("says Today, Yesterday, then the weekday and day of the month", () => {
    const today = "2026-11-26"; // a Thursday
    expect(islandLabel("2026-11-26", today)).toBe("Today");
    expect(islandLabel("2026-11-25", today)).toBe("Yesterday");
    expect(islandLabel("2026-11-23", today)).toBe("Mon 23");
    expect(islandLabel("2026-10-31", today)).toBe("Sat 31");
  });

  it("counts Yesterday across a month and year boundary", () => {
    expect(islandLabel("2026-12-31", "2027-01-01")).toBe("Yesterday");
  });
});

describe("the Grace Window on yesterday's Island", () => {
  // 02:00 on 27 Nov in Tokyo.
  const graceUntil = "2026-11-26T17:00:00Z";
  const at = (iso: string) => Date.parse(iso);

  it("counts the whole minutes left for a Free User", () => {
    const free = { plan: "free" as const, graceUntil };
    expect(graceLeft(free, at("2026-11-26T16:12:00Z"))).toBe(48);
    // 30 seconds left is still "1 min left", never "0 min left".
    expect(graceLeft(free, at("2026-11-26T16:59:30Z"))).toBe(1);
  });

  it("is over at 02:00, and never shows for Premium Users or outside the window", () => {
    expect(
      graceLeft({ plan: "free", graceUntil }, at("2026-11-26T17:00:00Z")),
    ).toBeNull();
    expect(
      graceLeft({ plan: "premium", graceUntil }, at("2026-11-26T16:12:00Z")),
    ).toBeNull();
    expect(
      graceLeft({ plan: "free", graceUntil: null }, at("2026-11-26T16:12:00Z")),
    ).toBeNull();
  });
});

// history lists every month from the first Day's to today's, newest first, as GET /days/summary does.
function history(from: string, to: string, written = 0) {
  const months = [];
  let y = Number(to.slice(0, 4));
  let m = Number(to.slice(5, 7));
  for (;;) {
    const month = `${y}-${String(m).padStart(2, "0")}`;
    months.push({ month, writtenDays: written });
    if (month === from) return months;
    m -= 1;
    if (m === 0) [y, m] = [y - 1, 12];
  }
}

const labels = (s: ReturnType<typeof scrubber>) => s.slots.map((x) => x.label);

describe("the rail scrubber", () => {
  it("shows months for this year and last, then one segment per older year", () => {
    const s = scrubber(history("2024-01", "2026-11"), "2026-11-26", 600);
    expect(labels(s)).toEqual([
      ...["11", "10", "09", "08", "07", "06", "05", "04", "03", "02", "01"].map(
        (m) => `2026-${m}`,
      ),
      ...[
        "12",
        "11",
        "10",
        "09",
        "08",
        "07",
        "06",
        "05",
        "04",
        "03",
        "02",
        "01",
      ].map((m) => `2025-${m}`),
      "2024",
    ]);
    expect(s.slot).toBe(18); // never taller than 18px
  });

  it("shrinks its slots so the whole history fits without scrolling", () => {
    expect(
      scrubber(history("2024-01", "2026-11"), "2026-11-26", 240).slot,
    ).toBe(10);
  });

  it("merges the oldest years into decades when even years can't fit", () => {
    // 50 years in 300px: 23 months and 48 years would be 4px each.
    const s = scrubber(history("1977-01", "2026-11"), "2026-11-26", 300);
    expect(labels(s).slice(23)).toEqual([
      ...Array.from({ length: 15 }, (_, i) => String(2024 - i)),
      "2000s",
      "1990s",
      "1980s",
      "1970s",
    ]);
    expect(s.slot).toBeCloseTo(300 / 42);
  });

  it("sizes each tick by how much of its time was written", () => {
    const months = history("2024-01", "2026-11");
    months[0] = { month: "2026-11", writtenDays: 15 }; // half of November
    for (const m of months.filter((m) => m.month.startsWith("2024"))) {
      m.writtenDays = m.month === "2024-02" ? 29 : 0; // a whole leap February
    }
    const s = scrubber(months, "2026-11-26", 600);
    expect(s.slots[0]?.written).toBe(0.5);
    expect(s.slots[1]?.written).toBe(0);
    expect(s.slots.at(-1)?.written).toBeCloseTo(29 / 366);
    expect(s.slots.at(-1)?.months.map((m) => m.month)).toHaveLength(12);
  });
});
