import { useInfiniteQuery, useQuery } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import {
  Fragment,
  useCallback,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
} from "react";
import type { Me } from "../api";
import { Island } from "./Island";
import { railQuery, summaryQuery } from "./queries";
import styles from "./Rail.module.css";
import { Scrubber } from "./Scrubber";

const monthName = new Intl.DateTimeFormat("en-GB", {
  month: "long",
  year: "numeric",
  timeZone: "UTC",
});
const monthLabel = (month: string) =>
  monthName.format(new Date(`${month}-01T00:00:00Z`));

// The left sidebar: Today, a divider, and the Island rail with its scrubber.
export function Sidebar({
  me,
  day,
  notebook,
  onOpenSettings,
}: {
  me: Me;
  day: string;
  notebook: string;
  onOpenSettings: () => void;
}) {
  const navigate = useNavigate();
  const thisMonth = me.today.slice(0, 7);
  // The month the rail starts from: today's, or the one the scrubber jumped to.
  const [anchor, setAnchor] = useState(thisMonth);
  const railRef = useRef<HTMLElement>(null);
  const [inView, setInView] = useState(thisMonth);

  const open = (d: string) =>
    navigate({ to: "/day/$day/$notebook", params: { day: d, notebook } });

  const toToday = () => {
    setAnchor(thisMonth);
    railRef.current?.scrollTo({ top: 0 });
    void open(me.today);
  };

  const jump = useCallback((month: string) => {
    setAnchor(month);
    railRef.current?.scrollTo({ top: 0 });
  }, []);

  return (
    <aside className={styles.sidebar}>
      <nav className={styles.nav} aria-label="Journal">
        <button
          type="button"
          className={`${styles.navItem} ${day === me.today ? styles.on : ""}`}
          onClick={toToday}
        >
          <SunIcon />
          Today
        </button>
        <button
          type="button"
          className={styles.navItem}
          onClick={onOpenSettings}
        >
          <GearIcon />
          Settings
        </button>
      </nav>
      <div className={styles.divider} />
      <div className={styles.railWrap}>
        <Rail
          key={anchor}
          anchor={anchor}
          me={me}
          day={day}
          railRef={railRef}
          onOpen={open}
          onMonthInView={setInView}
        />
        <ScrubberFromSummary me={me} inView={inView} onJump={jump} />
      </div>
    </aside>
  );
}

function ScrubberFromSummary({
  me,
  inView,
  onJump,
}: {
  me: Me;
  inView: string;
  onJump: (month: string) => void;
}) {
  const { data } = useQuery(summaryQuery);
  if (!data) return null;
  return (
    <Scrubber
      months={data.months}
      today={me.today}
      inView={inView}
      onJump={onJump}
    />
  );
}

function Rail({
  anchor,
  me,
  day,
  railRef,
  onOpen,
  onMonthInView,
}: {
  anchor: string;
  me: Me;
  day: string;
  railRef: React.RefObject<HTMLElement | null>;
  onOpen: (day: string) => void;
  onMonthInView: (month: string) => void;
}) {
  const q = useInfiniteQuery(railQuery(anchor));
  const top = useRef<HTMLDivElement>(null);
  const bottom = useRef<HTMLDivElement>(null);
  // Newer months load above what's on screen: keep it from jumping down.
  const heightBefore = useRef<number | null>(null);

  const {
    fetchNextPage,
    fetchPreviousPage,
    hasNextPage,
    hasPreviousPage,
    isFetching,
  } = q;
  useEffect(() => {
    const rail = railRef.current;
    if (!rail) return;
    const io = new IntersectionObserver(
      (entries) => {
        if (isFetching) return;
        for (const e of entries) {
          if (!e.isIntersecting) continue;
          if (e.target === bottom.current && hasNextPage) void fetchNextPage();
          if (e.target === top.current && hasPreviousPage) {
            heightBefore.current = rail.scrollHeight;
            void fetchPreviousPage();
          }
        }
      },
      { root: rail, rootMargin: "300px" },
    );
    if (top.current) io.observe(top.current);
    if (bottom.current) io.observe(bottom.current);
    return () => io.disconnect();
  }, [
    railRef,
    isFetching,
    hasNextPage,
    hasPreviousPage,
    fetchNextPage,
    fetchPreviousPage,
  ]);

  const months = q.data?.pages ?? [];
  useLayoutEffect(() => {
    const rail = railRef.current;
    if (!rail || heightBefore.current === null || months.length === 0) return;
    rail.scrollTop += rail.scrollHeight - heightBefore.current;
    heightBefore.current = null;
  }, [months, railRef]);

  // The month whose sticky header is at the top, for the scrubber's marker.
  const onScroll = () => {
    const rail = railRef.current;
    if (!rail) return;
    let current = months[0]?.month;
    for (const h of rail.querySelectorAll<HTMLElement>("[data-month]")) {
      if (h.offsetTop > rail.scrollTop + 12) break; // .rail is the offsetParent
      current = h.dataset.month;
    }
    if (current) onMonthInView(current);
  };
  const firstMonth = months[0]?.month;
  useEffect(() => {
    if (firstMonth) onMonthInView(firstMonth);
  }, [firstMonth, onMonthInView]);

  return (
    <nav
      className={styles.rail}
      ref={railRef}
      onScroll={onScroll}
      aria-label="Days"
    >
      <div ref={top}>{q.hasPreviousPage && <Shimmer />}</div>
      {months.map((m) => (
        <Fragment key={m.month}>
          <h2 className={styles.month} data-month={m.month}>
            {monthLabel(m.month)}
          </h2>
          <ul className={styles.islands}>
            {m.days.map((isl) => (
              <Island
                key={isl.day}
                island={isl}
                me={me}
                selected={isl.day === day}
                onOpen={onOpen}
              />
            ))}
          </ul>
        </Fragment>
      ))}
      <div ref={bottom}>
        {q.isPending || q.hasNextPage ? (
          <Shimmer />
        ) : (
          <p className={styles.railEnd}>
            You started here ·{" "}
            {new Date(`${me.firstDay}T00:00:00Z`).toLocaleDateString("en-GB", {
              day: "numeric",
              month: "long",
              year: "numeric",
              timeZone: "UTC",
            })}
          </p>
        )}
      </div>
    </nav>
  );
}

function Shimmer() {
  return (
    <>
      <div className={styles.placeholder} />
      <div className={styles.placeholder} />
      <div className={styles.placeholder} />
    </>
  );
}

function SunIcon() {
  return (
    <svg
      width="15"
      height="15"
      viewBox="0 0 16 16"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.5"
      strokeLinecap="round"
      aria-hidden="true"
    >
      <circle cx="8" cy="8" r="3" />
      <path d="M8 1.5v1.5M8 13v1.5M1.5 8H3M13 8h1.5M3.4 3.4l1 1M11.6 11.6l1 1M3.4 12.6l1-1M11.6 4.4l1-1" />
    </svg>
  );
}

function GearIcon() {
  return (
    <svg
      width="15"
      height="15"
      viewBox="0 0 16 16"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.5"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <circle cx="8" cy="8" r="2.2" />
      <path d="M8 1.5v1.6M8 12.9v1.6M1.5 8h1.6M12.9 8h1.6M3.4 3.4l1.1 1.1M11.5 11.5l1.1 1.1M3.4 12.6l1.1-1.1M11.5 4.5l1.1-1.1" />
      <circle cx="8" cy="8" r="4.6" />
    </svg>
  );
}
