import { useQuery } from "@tanstack/react-query";
import { type ReactNode, useCallback, useLayoutEffect, useState } from "react";
import { createPortal } from "react-dom";
import { summaryQuery } from "../rail/queries";
import { keyLabels } from "../shortcuts/shortcuts";
import styles from "./CoachMarks.module.css";
import { endCoachMarks, showCoachMarks } from "./firstRun";

// Where each coach mark points: an element carrying data-coach="<target>", and the
// side of the mark its arrow is on.
type Mark = {
  target: string;
  side: "top" | "left" | "right";
  title: string;
  body: ReactNode;
};

const marks: Mark[] = [
  {
    target: "page",
    side: "top",
    title: "Today’s Page",
    body: "Just start writing. It saves as you type. The first line becomes the Day’s Title and turns bold.",
  },
  {
    target: "notebooks",
    side: "right",
    title: "Two Notebooks",
    body: (
      <>
        <b>My Journal</b> for your day, <b>I Learned</b> for one thing you found
        out. Each has its own Page every Day.
      </>
    ),
  },
  {
    target: "rail",
    side: "left",
    title: "Your Days",
    body: (
      <>
        Each Day gets an Island here, with its Title. You can write until 02:00,
        then the Day closes. <kbd>{keyLabels(["Mod", "/"]).join("")}</kbd> shows
        formatting shortcuts.
      </>
    ),
  },
];

// useCoachMarks is the first-run walkthrough's state: which mark shows (null for none),
// and ending it, which Skip, Got it and typing all do for good in this browser.
export function useCoachMarks() {
  const { data } = useQuery(summaryQuery);
  const [ended, setEnded] = useState(false);
  const [step, setStep] = useState(0);
  const shown =
    !ended &&
    data !== undefined &&
    showCoachMarks({ months: data.months, storage: localStorage });

  const end = useCallback(() => {
    endCoachMarks(localStorage);
    setEnded(true);
  }, []);
  const next = () => (step === marks.length - 1 ? end() : setStep(step + 1));

  return { step: shown ? step : null, next, end };
}

export function CoachMarks({
  step,
  target,
  onNext,
  onSkip,
}: {
  step: number | null;
  target?: (mark: string) => string; // lets a narrow window point step 3 elsewhere
  onNext: () => void;
  onSkip: () => void;
}) {
  const mark = step === null ? null : (marks[step] ?? null);
  const at = usePlacement(mark, target);
  if (!mark || step === null || !at) return null;
  const last = step === marks.length - 1;
  const id = `coach-${mark.target}`;

  return createPortal(
    <div
      role="dialog"
      aria-modal="false"
      aria-labelledby={id}
      className={`${styles.mark} ${styles[at.side]}`}
      style={{ left: at.x, top: at.y }}
    >
      <b id={id} className={styles.title}>
        {mark.title}
      </b>
      {mark.body}
      <div className={styles.foot}>
        <span>
          {step + 1} of {marks.length}
        </span>
        <span>
          <button type="button" className={styles.skip} onClick={onSkip}>
            Skip
          </button>
          <button type="button" className={styles.next} onClick={onNext}>
            {last ? "Got it" : "Next"}
          </button>
        </span>
      </div>
    </div>,
    document.body,
  );
}

const WIDTH = 250;
const GAP = 12;

// usePlacement finds the mark's target and places the mark beside it, following resizes.
function usePlacement(
  mark: Mark | null,
  target: (mark: string) => string = (m) => m,
) {
  const [at, setAt] = useState<{
    x: number;
    y: number;
    side: Mark["side"];
  } | null>(null);
  const name = mark ? target(mark.target) : null;
  const side = mark?.side ?? "top";

  useLayoutEffect(() => {
    if (!name) {
      setAt(null);
      return;
    }
    const place = () => {
      const el = document.querySelector(`[data-coach="${name}"]`);
      const next = el ? placeBeside(el.getBoundingClientRect(), side) : null;
      // Only on a real move: the observer below sees the mark's own renders too.
      setAt((was) =>
        was &&
        next &&
        was.x === next.x &&
        was.y === next.y &&
        was.side === next.side
          ? was
          : next,
      );
    };
    place();
    window.addEventListener("resize", place);
    // Targets may mount after the mark (the rail loads its months).
    const mo = new MutationObserver(place);
    mo.observe(document.body, { childList: true, subtree: true });
    return () => {
      window.removeEventListener("resize", place);
      mo.disconnect();
    };
  }, [name, side]);

  return at;
}

function placeBeside(r: DOMRect, side: Mark["side"]) {
  const clampX = (x: number) =>
    Math.round(Math.max(GAP, Math.min(x, window.innerWidth - WIDTH - GAP)));
  // No room to the right (a narrow window): the mark goes below its target.
  const s =
    side === "left" && r.right + GAP + WIDTH > window.innerWidth ? "top" : side;
  if (s === "top")
    return {
      x: clampX(r.left + Math.min(40, r.width / 4)),
      y: Math.round(r.top + Math.min(48, r.height + GAP)),
      side: s,
    };
  if (s === "right")
    return {
      x: clampX(r.left - WIDTH - GAP),
      y: Math.round(r.top + 64),
      side: s,
    };
  return { x: clampX(r.right + GAP), y: Math.round(r.top + 8), side: s };
}
