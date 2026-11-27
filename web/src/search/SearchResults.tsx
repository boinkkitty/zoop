import { useEffect, useRef } from "react";
import type { SearchResult, Segment } from "../api";
import { NOTEBOOKS } from "../page/notebooks";
import styles from "../rail/Rail.module.css";
import { queryWords } from "./match";
import { byMonth, endOfResults, resultLabel } from "./results";
import own from "./Search.module.css";

const monthName = new Intl.DateTimeFormat("en-GB", {
  month: "long",
  year: "numeric",
  timeZone: "UTC",
});
const monthLabel = (month: string) =>
  monthName.format(new Date(`${month}-01T00:00:00Z`));

const notebookName = (slug: string) =>
  NOTEBOOKS.find((n) => n.slug === slug)?.name ?? slug;

// A snippet's lines run on with " · ", like the rail's glimpses.
const flat = (snippet: Segment[]) =>
  snippet.map((s) => ({ ...s, text: s.text.replace(/\n+/g, " · ") }));

export function Marked({ segments }: { segments: Segment[] }) {
  return segments.map((s, i) =>
    // biome-ignore lint/suspicious/noArrayIndexKey: segments never reorder
    s.match ? <mark key={i}>{s.text}</mark> : <span key={i}>{s.text}</span>,
  );
}

// The results that replace the Island rail while there's a query, grouped under the
// rail's month headers, newest first, loading more as the end scrolls into view.
export function SearchResults({
  q,
  results,
  today,
  active,
  opened,
  firstLoad,
  stale,
  hasMore,
  onLoadMore,
  onOpen,
}: {
  q: string;
  results: SearchResult[];
  today: string;
  active: number;
  opened: string | null; // "day/notebook" of the result opened from search, while it's open
  firstLoad: boolean;
  stale: boolean;
  hasMore: boolean;
  onLoadMore: () => void;
  onOpen: (index: number) => void;
}) {
  const scroller = useRef<HTMLDivElement>(null);
  const end = useRef<HTMLDivElement>(null);
  const words = queryWords(q);

  useEffect(() => {
    const root = scroller.current;
    if (!root || !end.current || !hasMore) return;
    const io = new IntersectionObserver(
      (es) => {
        if (es.some((e) => e.isIntersecting)) onLoadMore();
      },
      { root, rootMargin: "150px" },
    );
    io.observe(end.current);
    return () => io.disconnect();
  }, [hasMore, onLoadMore]);

  // Keep the active result in view as ↑ / ↓ move it.
  useEffect(() => {
    scroller.current
      ?.querySelector(`[data-index="${active}"]`)
      ?.scrollIntoView({ block: "nearest" });
  }, [active]);

  let body: React.ReactNode;
  if (firstLoad) {
    body = (
      <>
        <div className={own.placeholder} />
        <div className={own.placeholder} />
        <div className={own.placeholder} />
        <div className={own.placeholder} />
      </>
    );
  } else if (results.length === 0) {
    body = (
      <p className={own.message}>
        <b>No Pages match “{q.trim()}”</b>
        Search finds whole words and the start of words, in both Notebooks.
        <br />
        Dates count as words too: “november”, “nov 2025”, “thursday”.
      </p>
    );
  } else {
    body = (
      <>
        <ul
          className={`${own.list} ${stale ? own.stale : ""}`}
          aria-label="Search results"
        >
          {byMonth(results).map((m) => (
            <li key={m.month}>
              <h2 className={`${styles.month} ${own.month}`}>
                {monthLabel(m.month)}
              </h2>
              <ul className={own.list}>
                {m.results.map((r) => {
                  const open = opened === `${r.day}/${r.notebook}`;
                  return (
                    <li key={`${r.day}/${r.notebook}`}>
                      <button
                        type="button"
                        className={[
                          own.result,
                          r.index === active ? own.active : "",
                          open ? own.open : "",
                        ].join(" ")}
                        data-index={r.index}
                        data-day={r.day}
                        aria-current={open ? "page" : undefined}
                        onClick={() => onOpen(r.index)}
                      >
                        <span className={own.head}>
                          <span className={own.day}>
                            <Marked
                              segments={resultLabel(r.day, today, words)}
                            />
                          </span>
                          <span className={own.notebook}>
                            {notebookName(r.notebook)}
                          </span>
                        </span>
                        {r.title && (
                          <span className={own.title}>{r.title}</span>
                        )}
                        <span className={own.snippet}>
                          <Marked segments={flat(r.snippet)} />
                        </span>
                      </button>
                    </li>
                  );
                })}
              </ul>
            </li>
          ))}
        </ul>
        <div ref={end}>
          {hasMore ? (
            <div className={own.placeholder} />
          ) : (
            <p className={own.end}>{endOfResults(results.length)}</p>
          )}
        </div>
      </>
    );
  }

  return (
    <div className={own.results} ref={scroller}>
      <p className={own.caption}>Newest first</p>
      {body}
    </div>
  );
}
