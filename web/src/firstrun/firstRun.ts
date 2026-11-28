// The first-run coach marks' rules, kept free of React so they can be tested on their own.

const endedKey = "zoop.coachMarksEnded";

// showCoachMarks is whether to show them: only while the user has no Written Day
// (from GET /days/summary) and they haven't been ended in this browser.
export function showCoachMarks({
  months,
  storage,
}: {
  months: { month: string; writtenDays: number }[];
  storage: Storage;
}): boolean {
  return months.every((m) => m.writtenDays === 0) && !ended(storage);
}

// endCoachMarks ends them for good in this browser: Skip, Got it or typing.
export function endCoachMarks(storage: Storage) {
  try {
    storage.setItem(endedKey, "1");
  } catch {
    // Storage full or blocked: they still end until the next load.
  }
}

function ended(storage: Storage) {
  try {
    return storage.getItem(endedKey) !== null;
  } catch {
    return false;
  }
}
