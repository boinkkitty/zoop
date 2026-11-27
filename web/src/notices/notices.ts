// Notices: short messages about the app, the account or the open Page, shown in
// the notice pill. These are the rules for which show and in what order.

// Most urgent first.
const URGENCY = [
  "update",
  "conflict",
  "notSaved",
  "dayClosed",
  "timezone",
  "verifyEmail",
] as const;

export type NoticeId = (typeof URGENCY)[number];

// The others clear only when their cause is gone or answered.
const DISMISSIBLE: ReadonlySet<NoticeId> = new Set([
  "dayClosed",
  "verifyEmail",
]);

export const canDismiss = (id: NoticeId) => DISMISSIBLE.has(id);

// noticeStack orders the active Notices most urgent first, leaving out dismissed ones.
export function noticeStack<N extends { id: NoticeId }>(
  active: readonly N[],
  dismissed: ReadonlySet<NoticeId>,
): N[] {
  return active
    .filter((n) => !(canDismiss(n.id) && dismissed.has(n.id)))
    .sort((a, b) => URGENCY.indexOf(a.id) - URGENCY.indexOf(b.id));
}
