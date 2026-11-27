import { describe, expect, it } from "vitest";
import type { Notebook } from "../page/notebooks";
import { byMonth, endOfResults, moveActive, resultLabel } from "./results";

const result = (day: string, notebook: Notebook = "my-journal") => ({
  day,
  notebook,
  title: "",
  snippet: [],
});

describe("the results list", () => {
  it("groups results under month headers, keeping their newest-first order", () => {
    const groups = byMonth([
      result("2026-11-26"),
      result("2026-11-26", "i-learned"),
      result("2026-11-02"),
      result("2025-11-30"),
    ]);
    expect(groups.map((g) => [g.month, g.results.map((r) => r.index)])).toEqual(
      [
        ["2026-11", [0, 1, 2]],
        ["2025-11", [3]],
      ],
    );
  });

  it("moves the active result with ↑ and ↓, stopping at either end", () => {
    expect(moveActive(0, 5, "ArrowDown")).toEqual({
      active: 1,
      loadMore: false,
    });
    expect(moveActive(0, 5, "ArrowUp")).toEqual({ active: 0, loadMore: false });
    expect(moveActive(4, 5, "ArrowDown").active).toBe(4);
  });

  it("asks for more as ↓ nears the end of what's loaded", () => {
    expect(moveActive(15, 20, "ArrowDown")).toEqual({
      active: 16,
      loadMore: false,
    });
    expect(moveActive(16, 20, "ArrowDown")).toEqual({
      active: 17,
      loadMore: true,
    });
  });

  it("ends by counting every match", () => {
    expect(endOfResults(1)).toBe("One Page · that’s every match");
    expect(endOfResults(23)).toBe("23 Pages · that’s every match");
  });
});

describe("a result's label", () => {
  const today = "2026-11-26"; // a Thursday

  it("names the Day as the rail does when the date didn't match", () => {
    expect(resultLabel("2026-11-26", today, ["rain"])).toEqual([
      { text: "Today", match: false },
    ]);
    expect(resultLabel("2026-11-25", today, ["rain"])).toEqual([
      { text: "Yesterday", match: false },
    ]);
    expect(resultLabel("2026-11-22", today, ["rain"])).toEqual([
      { text: "Sun 22", match: false },
    ]);
  });

  it("shows the full date with the matching words marked when the date matched", () => {
    expect(resultLabel("2026-11-22", today, ["nov", "mum"])).toEqual([
      { text: "Sun, 22 ", match: false },
      { text: "November", match: true },
      { text: " 2026", match: false },
    ]);
    // The weekday is shown short but matched in full, as the server's date words have it.
    expect(resultLabel("2026-11-26", today, ["thursday"])).toEqual([
      { text: "Thu", match: true },
      { text: ", 26 November 2026", match: false },
    ]);
  });
});
