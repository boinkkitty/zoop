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
import type { Me, SearchResult } from "../api";
import { useSearch } from "../search/queries";
import { moveActive } from "../search/results";
import search from "../search/Search.module.css";
import { SearchResults } from "../search/SearchResults";
import { isMac, isSearchShortcut } from "../shortcuts/shortcuts";
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

// The left sidebar: Today, Settings, search, a divider, and the Island rail with its scrubber.
// While there's a query, search results take the rail's place. The query is kept by
// the caller, so it outlives the narrow-window drawer closing. onChoose runs on choosing
// a Day or a result, which closes that drawer.
export function Sidebar({
  me,
  day,
  notebook,
  query: q,
  onQueryChange: setQ,
  onOpenResult,
  onOpenSettings,
  onChoose = () => {},
}: {
  me: Me;
  day: string;
  notebook: string;
  query: string;
  onQueryChange: (q: string) => void;
  onOpenResult: (r: SearchResult, q: string) => void;
  onOpenSettings: () => void;
  onChoose?: () => void;
}) {
  const navigate = useNavigate();
  const found = useSearch(q);
  const [active, setActive] = useState(0);
  // The result last opened from search: it keeps a bar while its Page is open.
  const [opened, setOpened] = useState<string | null>(null);
  const searching = q.trim() !== "";
  const box = useRef<HTMLInputElement>(null);
  // A new query's results start with the first one active.
  const resultsFor = found.stale ? null : found.settled;
  useEffect(() => {
    if (resultsFor !== null) setActive(0);
  }, [resultsFor]);
  const thisMonth = me.today.slice(0, 7);
  // The month the rail starts from: today's, or the one the scrubber jumped to.
  const [anchor, setAnchor] = useState(thisMonth);
  const railRef = useRef<HTMLElement>(null);
  const [inView, setInView] = useState(thisMonth);

  const open = (d: string) => {
    onChoose();
    return navigate({
      to: "/day/$day/$notebook",
      params: { day: d, notebook },
    });
  };

  const toToday = () => {
    setAnchor(thisMonth);
    railRef.current?.scrollTo({ top: 0 });
    void open(me.today);
  };

  const jump = useCallback((month: string) => {
    setAnchor(month);
    railRef.current?.scrollTo({ top: 0 });
  }, []);

  // ⌘K (Ctrl+K off a Mac) focuses search from anywhere.
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (!isSearchShortcut(e)) return;
      e.preventDefault();
      box.current?.focus();
      box.current?.select();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);

  const openResult = (index: number) => {
    const r = found.results[index];
    if (!r) return;
    setActive(index);
    setOpened(`${r.day}/${r.notebook}`);
    box.current?.blur();
    onChoose();
    onOpenResult(r, found.settled);
  };

  // Clearing the query brings the rail back, at the opened Day.
  const clear = () => {
    setQ("");
    setAnchor(day.slice(0, 7));
  };

  const onKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    const n = found.results.length;
    if ((e.key === "ArrowDown" || e.key === "ArrowUp") && n > 0) {
      e.preventDefault();
      const next = moveActive(active, n, e.key);
      setActive(next.active);
      if (next.loadMore) found.loadMore();
    } else if (e.key === "Enter" && n > 0) {
      e.preventDefault();
      openResult(active);
    } else if (e.key === "Escape") {
      e.preventDefault();
      e.stopPropagation(); // not the opened Page's highlights
      if (q) clear();
      else box.current?.blur();
    }
  };

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
        <label className={`${search.box} ${q ? search.has : ""}`}>
          <SearchIcon />
          <input
            ref={box}
            type="search"
            aria-label="Search"
            placeholder="Search"
            autoComplete="off"
            spellCheck={false}
            value={q}
            onChange={(e) => setQ(e.target.value)}
            onKeyDown={onKeyDown}
          />
          {searching && found.loading && <span className={search.spin} />}
          {q ? (
            <button
              type="button"
              className={search.clear}
              aria-label="Clear search"
              onClick={() => {
                clear();
                box.current?.focus();
              }}
            >
              ×
            </button>
          ) : (
            <kbd>{isMac() ? "⌘K" : "Ctrl K"}</kbd>
          )}
        </label>
      </nav>
      <div className={styles.divider} />
      <div className={styles.railWrap} data-coach="rail">
        {searching ? (
          <SearchResults
            q={found.settled}
            results={found.results}
            today={me.today}
            active={active}
            opened={opened === `${day}/${notebook}` ? opened : null}
            firstLoad={found.firstLoad}
            stale={found.stale}
            hasMore={found.hasMore}
            onLoadMore={found.loadMore}
            onOpen={openResult}
          />
        ) : (
          <>
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
          </>
        )}
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
  // Until the first Written Day, the rail's end explains what Islands are.
  const { data: summary } = useQuery(summaryQuery);
  const noWrittenDay = summary?.months.every((m) => m.writtenDays === 0);
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

  // Coming back from search (or a jump), the open Day starts in the middle of the rail.
  const revealed = useRef(false);
  useLayoutEffect(() => {
    const rail = railRef.current;
    if (!rail || revealed.current || months.length === 0) return;
    revealed.current = true;
    const open = rail.querySelector<HTMLElement>('[aria-current="date"]');
    const slot = open?.closest("li");
    if (slot) rail.scrollTop = slot.offsetTop - rail.clientHeight / 2;
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
          <>
            <p className={styles.railEnd}>
              You started here ·{" "}
              {me.firstDay === me.today
                ? "today"
                : new Date(`${me.firstDay}T00:00:00Z`).toLocaleDateString(
                    "en-GB",
                    {
                      day: "numeric",
                      month: "long",
                      year: "numeric",
                      timeZone: "UTC",
                    },
                  )}
            </p>
            {noWrittenDay && (
              <p className={styles.railHint}>
                Every Day gets an Island here. Its Title is the first line you
                write.
              </p>
            )}
          </>
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

function SearchIcon() {
  return (
    <svg
      width="12"
      height="12"
      viewBox="0 0 16 16"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      aria-hidden="true"
      style={{ flex: "none" }}
    >
      <circle cx="7" cy="7" r="5" />
      <path d="m11 11 4 4" />
    </svg>
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
