// The autosave queue for one open Page. It has no React in it, so it's tested
// with fake timers and a fake fetch.
import type { Page } from "../api";
import { type Doc, SCHEMA_VERSION } from "./schema";

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
};

export type SaverOptions = SaverDeps & {
  day: string;
  notebook: string;
  page: Page;
};

export type SaverStatus = {
  // Set only after a save has failed: when the next retry goes out.
  notSaved: { retryAt: number } | null;
};

export type PageSaver = ReturnType<typeof createPageSaver>;

type Timer = ReturnType<typeof setTimeout> | undefined;

export function createPageSaver(o: SaverOptions) {
  const url = `/api/v1/pages/${o.day}/${o.notebook}`;
  let base = o.page.version;
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

  let status: SaverStatus = { notSaved: null };
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
    if (!pending) return;
    const doc = pending;
    pending = null;
    inFlight = true;
    let ok = false;
    try {
      const res = await put(doc, false);
      if (res.ok) {
        const saved = (await res.json()) as Page;
        base = saved.version;
        ok = true;
        if (pending) keep();
        else forget();
        o.onSaved?.(saved);
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

  function scheduleRetry() {
    sendAfter = false;
    stopTimers();
    retry = setTimeout(send, retryDelay);
    setStatus({ notSaved: { retryAt: Date.now() + retryDelay } });
    retryDelay = Math.min(retryDelay * 2, MAX_RETRY_MS);
  }

  if (kept) void send();

  return {
    // The writing kept on this device from an earlier visit, shown instead of the server's copy.
    restored: kept?.doc ?? null,
    // edit records the editor's latest doc: it's saved after 1s idle, or within 5s while typing continues.
    // While a failed save waits to retry, the retry carries the latest doc instead.
    edit(doc: Doc) {
      pending = doc;
      keep();
      if (retry) return;
      clearTimeout(idle);
      idle = setTimeout(send, IDLE_MS);
      maxWait ??= setTimeout(send, MAX_WAIT_MS);
    },
    // flush saves any pending edit now.
    flush() {
      void send();
    },
    // flushOnUnload sends what's pending as the page goes away, without waiting for a save in flight.
    // The doc stays kept on this device until a later visit sees it acknowledged.
    flushOnUnload() {
      stopTimers();
      if (pending) void put(pending, true).catch(() => {});
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
