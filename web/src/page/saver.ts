// The autosave queue for one open Page. It has no React in it, so it's tested
// with fake timers and a fake fetch.
import type { Page } from "../api";
import { type Doc, SCHEMA_VERSION, sameDoc } from "./schema";

export const IDLE_MS = 1000;
export const MAX_WAIT_MS = 5000;
const FIRST_RETRY_MS = 2000;
const MAX_RETRY_MS = 60_000;

export type SaverDeps = {
  fetch: typeof fetch;
  storage: Storage;
  // Whose writing this is, so kept writing is never shown to another account on this browser.
  owner?: string;
  onSaved?: (page: Page) => void;
  // The editor must show this doc instead of its own (it must not echo it back as an edit).
  onReplace?: (doc: Doc) => void;
};

export type SaverOptions = SaverDeps & {
  day: string;
  notebook: string;
  page: Page;
};

export type SaverStatus = {
  // Set only after a save has failed: when the next retry goes out.
  notSaved: { retryAt: number } | null;
  // Both this device and another changed the Page: the other device's version.
  conflict: Page | null;
  // The server is on a newer schema: saving stops until reload.
  outdated: boolean;
  // The Day has closed: edits are no longer taken. What was written by then is still sent.
  locked: boolean;
};

export type PageSaver = ReturnType<typeof createPageSaver>;

type Timer = ReturnType<typeof setTimeout> | undefined;

export function createPageSaver(o: SaverOptions) {
  const url = `/api/v1/pages/${o.day}/${o.notebook}`;
  let base = o.page.version;
  // The doc the server holds at base, as far as this device knows.
  let confirmed: Doc = o.page.doc;
  let pending: Doc | null = null;
  let idle: Timer;
  let maxWait: Timer;
  let retry: Timer;
  let retryDelay = FIRST_RETRY_MS;

  // The unacknowledged doc is kept on this device until the server has it.
  const owner = o.owner ?? "";
  const mirrorKey = `zoop:unsaved:${o.day}/${o.notebook}`;
  function keep() {
    if (!pending) return;
    const kept: Kept = { owner, doc: pending, baseVersion: base };
    try {
      o.storage.setItem(mirrorKey, JSON.stringify(kept));
    } catch {
      // Storage full or blocked: the save itself still goes ahead.
    }
  }
  function forget() {
    o.storage.removeItem(mirrorKey);
  }
  const kept = readKept(o.storage, mirrorKey, owner);
  if (kept) {
    pending = kept.doc;
    base = kept.baseVersion;
  }

  let status: SaverStatus = {
    notSaved: null,
    conflict: null,
    outdated: false,
    locked: false,
  };
  const listeners = new Set<(s: SaverStatus) => void>();
  function setStatus(next: Partial<SaverStatus>) {
    status = { ...status, ...next };
    for (const l of listeners) l(status);
  }

  function stopTimers() {
    clearTimeout(idle);
    clearTimeout(maxWait);
    clearTimeout(retry);
    idle = maxWait = retry = undefined;
  }

  function put(doc: Doc, keepalive: boolean) {
    return o.fetch(url, {
      method: "PUT",
      credentials: "same-origin",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        doc,
        schemaVersion: SCHEMA_VERSION,
        baseVersion: base,
      }),
      keepalive,
    });
  }

  // One save in flight per Page: a send asked for meanwhile waits for it, then sends the latest doc.
  let inFlight = false;
  let sendAfter = false;

  async function send() {
    stopTimers();
    if (inFlight) {
      sendAfter = true;
      return;
    }
    if (!pending || blocked()) return;
    const doc = pending;
    pending = null;
    inFlight = true;
    let ok = false;
    try {
      const res = await put(doc, false);
      if (res.ok) {
        acknowledged((await res.json()) as Page);
        ok = true;
      } else if (res.status === 409) {
        ok = refused(doc, await res.json());
      } else if (res.status === 403) {
        ok = closedByServer(doc, await res.json());
      }
    } catch {
      // A network failure: retried below like a server error.
    } finally {
      inFlight = false;
    }
    if (ok) {
      retryDelay = FIRST_RETRY_MS;
      if (status.notSaved) setStatus({ notSaved: null });
    } else {
      pending ??= doc; // a newer edit made meanwhile replaces the failed one
      scheduleRetry();
      return;
    }
    if (sendAfter) {
      sendAfter = false;
      await send();
    }
  }

  function acknowledged(saved: Page) {
    base = saved.version;
    confirmed = saved.doc;
    if (pending) keep();
    else forget();
    o.onSaved?.(saved);
  }

  // refused handles a 409, returning whether it's settled (rather than to be retried).
  function refused(
    sent: Doc,
    body: { error?: { code?: string }; page?: Page },
  ) {
    const mine = pending ?? sent;
    if (body.error?.code === "client_outdated") {
      // Keep the writing on this device for the reloaded, upgraded tab to send.
      pending = mine;
      keep();
      setStatus({ outdated: true });
      return true;
    }
    const current = body.page;
    if (body.error?.code !== "version_conflict" || !current) return false;
    if (sameDoc(mine, current.doc)) {
      pending = null;
      acknowledged(current);
      return true;
    }
    if (sameDoc(mine, confirmed)) {
      // Nothing changed here after all: take the other version quietly.
      pending = null;
      adopt(current);
      return true;
    }
    // Both sides have edits: keep ours on this device and let the writer choose.
    pending = mine;
    keep();
    setStatus({ conflict: current });
    return true;
  }

  function adopt(theirs: Page) {
    forget();
    acknowledged(theirs);
    o.onReplace?.(theirs.doc);
  }

  // The server refused a save because the Day closed (it arrived past the leeway): nothing more can be sent.
  let closed = false;
  function closedByServer(sent: Doc, body: { error?: { code?: string } }) {
    if (body.error?.code !== "page_locked") return false;
    pending ??= sent;
    keep(); // the words stay on this device rather than vanish
    closed = true;
    setStatus({ locked: true });
    return true;
  }

  // Saving waits while the writer chooses a version, or until a reload after an update.
  const blocked = () => status.conflict !== null || status.outdated || closed;

  function scheduleRetry() {
    sendAfter = false;
    stopTimers();
    retry = setTimeout(send, retryDelay);
    setStatus({ notSaved: { retryAt: Date.now() + retryDelay } });
    retryDelay = Math.min(retryDelay * 2, MAX_RETRY_MS);
  }

  // At locksAt the editor goes read only: what's pending is sent at once, within the server's leeway.
  function lock() {
    void send();
    setStatus({ locked: true });
  }
  const locksIn = o.page.locksAt
    ? Date.parse(o.page.locksAt) - Date.now()
    : Number.POSITIVE_INFINITY;
  let lockTimer: Timer;
  if (!o.page.editable || locksIn <= 0) status.locked = true;
  else if (Number.isFinite(locksIn)) lockTimer = setTimeout(lock, locksIn);

  if (kept) void send();

  return {
    // The writing kept on this device from an earlier visit, shown instead of the server's copy.
    restored: kept?.doc ?? null,
    // edit records the editor's latest doc: it's saved after 1s idle, or within 5s while typing continues.
    // While a failed save waits to retry, the retry carries the latest doc instead.
    edit(doc: Doc) {
      if (status.locked) return;
      pending = doc;
      keep();
      if (retry || blocked()) return;
      clearTimeout(idle);
      idle = setTimeout(send, IDLE_MS);
      maxWait ??= setTimeout(send, MAX_WAIT_MS);
    },
    // keepMine is "Keep this version": this device's writing goes over the other version.
    keepMine() {
      const theirs = status.conflict;
      if (!theirs) return;
      base = theirs.version;
      keep();
      setStatus({ conflict: null });
      void send();
    },
    // takeTheirs is "Use the other": the other version replaces this device's writing.
    takeTheirs() {
      const theirs = status.conflict;
      if (!theirs) return;
      pending = null;
      setStatus({ conflict: null });
      adopt(theirs);
    },
    // remoteChanged is told the server's latest copy (refetched on focus). With no local edits,
    // a newer version is taken quietly; otherwise the next save's conflict decides.
    remoteChanged(latest: Page) {
      if (pending || inFlight || latest.version <= base) return;
      adopt(latest);
    },
    // flush saves any pending edit now.
    flush() {
      void send();
    },
    // flushOnUnload sends what's pending as the page goes away, without waiting for a save in flight.
    // The doc stays kept on this device until a later visit sees it acknowledged.
    flushOnUnload() {
      stopTimers();
      if (pending && !blocked()) void put(pending, true).catch(() => {});
    },
    // retryNow is the Not saved Notice's "Try now".
    retryNow() {
      void send();
    },
    status: () => status,
    subscribe(fn: (s: SaverStatus) => void) {
      listeners.add(fn);
      return () => void listeners.delete(fn);
    },
    // dispose saves what's pending as the editor goes away.
    dispose() {
      clearTimeout(lockTimer);
      void send();
    },
  };
}

type Kept = { owner: string; doc: Doc; baseVersion: number };

function readKept(storage: Storage, key: string, owner: string): Kept | null {
  try {
    const raw = storage.getItem(key);
    if (!raw) return null;
    const kept = JSON.parse(raw) as Kept;
    return kept.owner === owner && kept.doc ? kept : null;
  } catch {
    return null;
  }
}
