import type { Page } from "../api";
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
}: {
  page: Page;
  today: string;
  timezone: string;
  readOnly: boolean;
}) {
  const date = new Date(`${page.day}T00:00:00Z`);
  const stillOpen = !readOnly && page.locksAt && page.day !== today;
  return (
    <header className={styles.head}>
      <p className={styles.weekday}>
        {weekday.format(date)}
        {page.day === today && " · Today"}
        {readOnly && <span className={styles.badge}>Read only</span>}
        {stillOpen && (
          <span className={`${styles.badge} ${styles.open}`}>
            <i className={styles.dot} />
            Still open until {clockTime(page.locksAt as string, timezone)}
          </span>
        )}
      </p>
      <h1 className={styles.date}>{longDate.format(date)}</h1>
    </header>
  );
}

// clockTime is a moment as a 24-hour wall-clock time in the writer's timezone, like "02:00".
function clockTime(moment: string, timezone: string) {
  return new Intl.DateTimeFormat(undefined, {
    hour: "2-digit",
    minute: "2-digit",
    hourCycle: "h23",
    timeZone: timezone,
  }).format(new Date(moment));
}
