import type { Page } from "../api";
import { matches } from "../search/match";
import search from "../search/Search.module.css";
import styles from "./DayHead.module.css";

// A Day is a calendar date, so it's formatted in UTC: never shifted by the browser's zone.
const weekday = new Intl.DateTimeFormat(undefined, {
  weekday: "long",
  timeZone: "UTC",
});
const longDate = new Intl.DateTimeFormat(undefined, {
  day: "numeric",
  month: "long",
  year: "numeric",
  timeZone: "UTC",
});

// The date at the top of the sheet, with the Day's status pill on the weekday line:
// "Read only" once it's closed, "Still open until 02:00" on yesterday in the Grace Window.
export function DayHead({
  page,
  today,
  timezone,
  readOnly,
  searchWords = [],
}: {
  page: Page;
  today: string;
  timezone: string;
  readOnly: boolean;
  searchWords?: readonly string[]; // marked in the date, when the Page was opened from search
}) {
  const date = new Date(`${page.day}T00:00:00Z`);
  const stillOpen = !readOnly && page.locksAt && page.day !== today;
  return (
    <header className={styles.head}>
      <p className={styles.weekday}>
        <MarkedDate text={weekday.format(date)} words={searchWords} />
        {page.day === today && " · Today"}
        {readOnly && <span className={styles.badge}>Read only</span>}
        {stillOpen && (
          <span className={`${styles.badge} ${styles.open}`}>
            <i className={styles.dot} />
            Still open until {clockTime(page.locksAt as string, timezone)}
          </span>
        )}
      </p>
      <h1 className={styles.date}>
        <MarkedDate text={longDate.format(date)} words={searchWords} />
      </h1>
    </header>
  );
}

// MarkedDate is text with the words a search matched marked, like the matches on the Page.
function MarkedDate({
  text,
  words,
}: {
  text: string;
  words: readonly string[];
}) {
  const out: React.ReactNode[] = [];
  let at = 0;
  for (const { from, to } of matches(text, words)) {
    out.push(text.slice(at, from));
    out.push(
      <mark key={from} className={search.dateMatch}>
        {text.slice(from, to)}
      </mark>,
    );
    at = to;
  }
  out.push(text.slice(at));
  return out;
}

// clockTime is a moment as a 24-hour wall-clock time in the writer's timezone, like "02:00".
export function clockTime(moment: string, timezone: string) {
  return new Intl.DateTimeFormat(undefined, {
    hour: "2-digit",
    minute: "2-digit",
    hourCycle: "h23",
    timeZone: timezone,
  }).format(new Date(moment));
}
