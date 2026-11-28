import { beforeEach, describe, expect, it } from "vitest";
import { endCoachMarks, showCoachMarks } from "./firstRun";

function memoryStorage(): Storage {
  const items = new Map<string, string>();
  return {
    get length() {
      return items.size;
    },
    clear: () => items.clear(),
    getItem: (k) => items.get(k) ?? null,
    key: (i) => [...items.keys()][i] ?? null,
    removeItem: (k) => items.delete(k),
    setItem: (k, v) => items.set(k, v),
  };
}

let storage: Storage;
beforeEach(() => {
  storage = memoryStorage();
});

const noWrittenDay = [{ month: "2026-11", writtenDays: 0 }];

describe("the first-run coach marks", () => {
  it("show for a user with no Written Day on a browser that hasn't ended them", () => {
    expect(showCoachMarks({ months: noWrittenDay, storage })).toBe(true);
  });

  it("never show for a user with a Written Day, even on a new browser", () => {
    const months = [
      { month: "2026-11", writtenDays: 0 },
      { month: "2026-10", writtenDays: 1 },
    ];
    expect(showCoachMarks({ months, storage })).toBe(false);
  });

  it("stay gone in this browser once ended, by Skip, Got it or typing", () => {
    endCoachMarks(storage);
    expect(showCoachMarks({ months: noWrittenDay, storage })).toBe(false);
  });
});
