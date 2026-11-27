import { useLayoutEffect, useRef, useState } from "react";
import styles from "./Rail.module.css";
import { daysInMonth, type MonthCount, type ScrubSlot, scrubber } from "./rail";

const monthYear = new Intl.DateTimeFormat("en-GB", {
  month: "long",
  year: "numeric",
  timeZone: "UTC",
});
const shortMonth = new Intl.DateTimeFormat("en-GB", {
  month: "short",
  timeZone: "UTC",
});
const asDate = (month: string) => new Date(`${month}-01T00:00:00Z`);

// The thin strip down the rail's right edge for jumping to any month. It never
// scrolls: the layout in rail.ts shrinks (and, for very long histories, merges) slots to fit.
export function Scrubber({
  months,
  today,
  inView,
  onJump,
}: {
  months: MonthCount[];
  today: string;
  inView: string;
  onJump: (month: string) => void;
}) {
  const ref = useRef<HTMLDivElement>(null);
  const [height, setHeight] = useState(0);
  const [hover, setHover] = useState<{ month: string; y: number } | null>(null);
  const [grid, setGrid] = useState<{ slot: ScrubSlot; y: number } | null>(null);

  useLayoutEffect(() => {
    const el = ref.current;
    if (!el) return;
    const ro = new ResizeObserver(() => setHeight(el.clientHeight));
    ro.observe(el);
    setHeight(el.clientHeight);
    return () => ro.disconnect();
  }, []);

  const { slots, slot } = scrubber(months, today, height);
  const inHistory = new Set(months.map((m) => m.month));

  // The month slot under a pointer at y px from the top, if it's a single month.
  const slotAt = (y: number) =>
    slots[Math.min(slots.length - 1, Math.max(0, Math.floor(y / slot)))];

  const onPointerMove = (e: React.PointerEvent) => {
    const y = e.clientY - e.currentTarget.getBoundingClientRect().top;
    const s = slotAt(y);
    if (!s) return;
    if (s.unit === "month") {
      setGrid(null);
      setHover({ month: s.label, y: (slots.indexOf(s) + 0.5) * slot });
    } else {
      setHover(null);
      setGrid({ slot: s, y: slots.indexOf(s) * slot });
    }
  };

  const onPointerUp = (e: React.PointerEvent) => {
    const y = e.clientY - e.currentTarget.getBoundingClientRect().top;
    const s = slotAt(y);
    if (s?.unit === "month") onJump(s.label);
  };

  // Where the "month in view" dot sits: on its month, or within its year or decade.
  const dotTop = (() => {
    const i = slots.findIndex((s) => s.months.some((m) => m.month === inView));
    const s = slots[i];
    if (!s) return null;
    const within =
      s.unit === "month"
        ? 0.5
        : (s.months.findIndex((m) => m.month === inView) + 0.5) /
          s.months.length;
    return (i + within) * slot;
  })();

  // Year labels ('25) at year boundaries, thinned so they never overlap.
  const yearLabels: { y: number; text: string }[] = [];
  let lastY = -Infinity;
  slots.forEach((s, i) => {
    const prev = slots[i - 1];
    if (s.unit !== "month" || (prev && prev.year === s.year)) return;
    const y = i * slot;
    if (y - lastY < 14) return;
    lastY = y;
    yearLabels.push({ y, text: `’${String(s.year).slice(2)}` });
  });

  return (
    <div className={styles.scrubWrap} onPointerLeave={() => setHover(null)}>
      <div
        ref={ref}
        className={styles.scrubber}
        onPointerMove={onPointerMove}
        onPointerDown={(e) => e.currentTarget.setPointerCapture(e.pointerId)}
        onPointerUp={onPointerUp}
        data-testid="scrubber"
      >
        {slots.map((s, i) => (
          <div
            key={s.label}
            className={s.unit === "month" ? styles.tick : styles.yearSlot}
            data-slot={s.label}
            style={{
              top: i * slot,
              height: slot,
              ...(s.unit === "month"
                ? { width: 3 + Math.round(s.written * 13) }
                : {}),
            }}
          />
        ))}
        {yearLabels.map((l) => (
          <span key={l.text} className={styles.yearLabel} style={{ top: l.y }}>
            {l.text}
          </span>
        ))}
        {dotTop !== null && (
          <span className={styles.now} style={{ top: dotTop }} />
        )}
      </div>
      {hover && (
        <span className={styles.bubble} style={{ top: hover.y }}>
          {monthYear.format(asDate(hover.month))}
        </span>
      )}
      {grid && (
        <div
          className={styles.grid}
          style={{ top: grid.y }}
          onPointerLeave={() => setGrid(null)}
        >
          {gridYears(grid.slot).map((y) => (
            <div key={y}>
              <b>{y}</b>
              <div className={styles.gridMonths}>
                {Array.from({ length: 12 }, (_, m) => {
                  const month = `${y}-${String(m + 1).padStart(2, "0")}`;
                  const count = grid.slot.months.find((x) => x.month === month);
                  return (
                    <button
                      key={month}
                      type="button"
                      disabled={!inHistory.has(month)}
                      onClick={() => {
                        setGrid(null);
                        onJump(month);
                      }}
                      aria-label={monthYear.format(asDate(month))}
                    >
                      {shortMonth.format(asDate(month))}
                      {count && (
                        <i
                          style={{
                            opacity: count.writtenDays / daysInMonth(month),
                          }}
                        />
                      )}
                    </button>
                  );
                })}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

// gridYears are the years a year or decade slot opens in its month grid, newest first.
function gridYears(s: ScrubSlot) {
  return [...new Set(s.months.map((m) => Number(m.month.slice(0, 4))))];
}
