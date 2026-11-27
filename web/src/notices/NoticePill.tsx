import { useState } from "react";
import styles from "./NoticePill.module.css";
import { canDismiss, type NoticeId, noticeStack } from "./notices";

export type Notice = {
  id: NoticeId;
  text: string;
  actions: { label: string; onClick: () => void }[];
};

// The notice pill: one black pill centred over the sheet's top edge. It shows the
// most urgent Notice and "+N"; hovering or focusing it expands it into the whole stack.
export function NoticePill({ notices }: { notices: Notice[] }) {
  const [dismissed, setDismissed] = useState<ReadonlySet<NoticeId>>(new Set());
  const stack = noticeStack(notices, dismissed);
  if (stack.length === 0) return null;
  const dismiss = (id: NoticeId) => setDismissed(new Set(dismissed).add(id));

  return (
    <section className={styles.pill} aria-label="Notices" aria-live="polite">
      {stack.map((n, i) => (
        <div key={n.id} className={i === 0 ? styles.top : styles.rest}>
          <p className={styles.text}>{n.text}</p>
          {n.actions.map((a) => (
            <button
              key={a.label}
              type="button"
              className={styles.action}
              onClick={a.onClick}
            >
              {a.label}
            </button>
          ))}
          {canDismiss(n.id) && (
            <button
              type="button"
              className={styles.close}
              aria-label="Dismiss"
              onClick={() => dismiss(n.id)}
            >
              ×
            </button>
          )}
          {i === 0 && stack.length > 1 && (
            <span className={styles.more}>+{stack.length - 1}</span>
          )}
        </div>
      ))}
    </section>
  );
}
