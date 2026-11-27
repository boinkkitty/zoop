import { useEffect, useState } from "react";
import type { Island as IslandData, Me } from "../api";
import { clockTime } from "../page/DayHead";
import styles from "./Rail.module.css";
import { graceLeft, islandLabel } from "./rail";

const notebookNames: Record<string, string> = {
  "my-journal": "My Journal",
  "i-learned": "I Learned",
};

// One Day in the rail: `date │ Title`, expanding on hover to a glimpse of each written Notebook.
// Yesterday's Island carries the Grace Window marker while the Day is still open.
export function Island({
  island,
  me,
  selected,
  onOpen,
}: {
  island: IslandData;
  me: Me;
  selected: boolean;
  onOpen: (day: string) => void;
}) {
  const label = islandLabel(island.day, me.today);
  const minutesLeft = useGraceLeft(label === "Yesterday" ? me : null);
  // In Notebook order, not the JSON's (Go writes map keys sorted).
  const glimpses = Object.keys(notebookNames)
    .map((n) => [n, island.glimpses[n] ?? ""] as const)
    .filter(([, g]) => g);
  const cls = [
    styles.slot,
    island.written ? "" : styles.empty,
    selected ? styles.selected : "",
  ].join(" ");

  return (
    <li className={cls}>
      <button
        type="button"
        className={styles.island}
        onClick={() => onOpen(island.day)}
        aria-current={selected ? "date" : undefined}
        data-day={island.day}
      >
        <span className={styles.row}>
          <span className={styles.label}>{label}</span>
          {island.written && island.title && (
            <>
              <span className={styles.sep} />
              <span className={styles.title}>{island.title}</span>
            </>
          )}
          {minutesLeft !== null && (
            <span className={styles.graceDot} title="Still open" />
          )}
        </span>
        {(glimpses.length > 0 || minutesLeft !== null) && (
          <span className={styles.more}>
            <span>
              {minutesLeft !== null && me.graceUntil && (
                <span className={styles.stillOpen}>
                  Still open until {clockTime(me.graceUntil, me.timezone)} ·{" "}
                  {minutesLeft} min left
                </span>
              )}
              {glimpses.map(([notebook, g]) => (
                <span key={notebook} className={styles.glimpse}>
                  <b>{notebookNames[notebook] ?? notebook}</b>
                  {g}
                </span>
              ))}
            </span>
          </span>
        )}
      </button>
    </li>
  );
}

// useGraceLeft is the minutes left in the Grace Window, kept current while it lasts,
// so the marker disappears at 02:00 without a reload. null (and no timer) otherwise.
function useGraceLeft(me: Me | null) {
  const [now, setNow] = useState(() => Date.now());
  const left = me ? graceLeft(me, now) : null;
  const open = left !== null;
  useEffect(() => {
    if (!open) return;
    const t = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(t);
  }, [open]);
  return left;
}
