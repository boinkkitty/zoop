// The autosave queue for one open Page. It has no React in it, so it's tested
// with fake timers and a fake fetch.
import type { Page } from "../api";
import { type Doc, SCHEMA_VERSION } from "./schema";

export const IDLE_MS = 1000;
export const MAX_WAIT_MS = 5000;

export type SaverDeps = {
  fetch: typeof fetch;
  storage: Storage;
  onSaved?: (page: Page) => void;
};

export type SaverOptions = SaverDeps & {
  day: string;
  notebook: string;
  page: Page;
};

export type PageSaver = ReturnType<typeof createPageSaver>;

export function createPageSaver(o: SaverOptions) {
  const url = `/api/v1/pages/${o.day}/${o.notebook}`;
  let base = o.page.version;
  let pending: Doc | null = null;
  let idle: ReturnType<typeof setTimeout> | undefined;
  let maxWait: ReturnType<typeof setTimeout> | undefined;

  function stopTimers() {
    clearTimeout(idle);
    clearTimeout(maxWait);
    idle = maxWait = undefined;
  }

  async function send() {
    stopTimers();
    if (!pending) return;
    const doc = pending;
    pending = null;
    const res = await o.fetch(url, {
      method: "PUT",
      credentials: "same-origin",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        doc,
        schemaVersion: SCHEMA_VERSION,
        baseVersion: base,
      }),
      keepalive: false,
    });
    if (res.ok) {
      const saved = (await res.json()) as Page;
      base = saved.version;
      o.onSaved?.(saved);
    }
  }

  return {
    // edit records the editor's latest doc: it's saved after 1s idle, or within 5s while typing continues.
    edit(doc: Doc) {
      pending = doc;
      clearTimeout(idle);
      idle = setTimeout(send, IDLE_MS);
      maxWait ??= setTimeout(send, MAX_WAIT_MS);
    },
    // flush saves any pending edit now.
    flush() {
      void send();
    },
    dispose() {
      void send();
    },
  };
}
