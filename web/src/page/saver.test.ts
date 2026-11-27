import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { Page } from "../api";
import { createPageSaver, type SaverDeps } from "./saver";
import { SCHEMA_VERSION } from "./schema";

const day = "2026-11-26";
const notebook = "my-journal";

const doc = (text: string) => ({
  type: "doc" as const,
  content: [{ type: "paragraph", content: [{ type: "text", text }] }],
});

const page = (version: number, d = doc("")): Page => ({
  day,
  notebook,
  doc: d,
  schemaVersion: SCHEMA_VERSION,
  version,
  updatedAt: version ? "2026-11-25T20:00:00Z" : null,
  editable: true,
  locksAt: "2026-11-26T17:00:00Z",
});

type Sent = {
  url: string;
  body: { doc: unknown; schemaVersion: number; baseVersion: number };
  keepalive: boolean;
};

// A fake server: records each PUT and answers with what the test queued, or a 200 at the next version.
function fakeServer() {
  const sent: Sent[] = [];
  const replies: (() => Promise<Response>)[] = [];
  const fetch = vi.fn(async (url: string, init: RequestInit) => {
    const body = JSON.parse(init.body as string);
    sent.push({ url, body, keepalive: !!init.keepalive });
    const reply = replies.shift();
    if (reply) return reply();
    return json(200, page(body.baseVersion + 1, body.doc));
  });
  // hold keeps the next request in flight until release(response) is called.
  function hold() {
    let release: (r: Response) => void = () => {};
    replies.push(() => new Promise<Response>((r) => (release = r)));
    return (r: Response) => release(r);
  }
  // failNext makes the next n requests fail as if the network were down.
  function failNext(n: number) {
    for (let i = 0; i < n; i++) {
      replies.push(() => Promise.reject(new TypeError("Failed to fetch")));
    }
  }
  return { sent, fetch, replies, hold, failNext };
}

function json(status: number, body: unknown) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

function memoryStorage(): Storage {
  const m = new Map<string, string>();
  return {
    getItem: (k) => m.get(k) ?? null,
    setItem: (k, v) => void m.set(k, v),
    removeItem: (k) => void m.delete(k),
    clear: () => m.clear(),
    key: (i) => [...m.keys()][i] ?? null,
    get length() {
      return m.size;
    },
  };
}

let server: ReturnType<typeof fakeServer>;
let storage: Storage;

function saver(start = page(0), deps: Partial<SaverDeps> = {}) {
  return createPageSaver({
    day,
    notebook,
    page: start,
    fetch: server.fetch as unknown as typeof fetch,
    storage,
    ...deps,
  });
}

beforeEach(() => {
  vi.useFakeTimers();
  server = fakeServer();
  storage = memoryStorage();
});

afterEach(() => {
  vi.useRealTimers();
});

describe("autosave", () => {
  it("saves once the writer has been idle for 1s", async () => {
    const s = saver();
    s.edit(doc("R"));
    await vi.advanceTimersByTimeAsync(500);
    s.edit(doc("Ra"));
    await vi.advanceTimersByTimeAsync(999);
    expect(server.sent).toHaveLength(0);

    await vi.advanceTimersByTimeAsync(1);
    expect(server.sent).toEqual([
      {
        url: "/api/v1/pages/2026-11-26/my-journal",
        body: { doc: doc("Ra"), schemaVersion: SCHEMA_VERSION, baseVersion: 0 },
        keepalive: false,
      },
    ]);
  });

  it("flushes a pending edit at once, and reports the saved Page", async () => {
    const onSaved = vi.fn();
    const s = saver(page(3), { onSaved });
    s.flush();
    expect(server.sent).toHaveLength(0);

    s.edit(doc("Off to the station"));
    s.flush();
    await vi.advanceTimersByTimeAsync(0);
    expect(server.sent.map((r) => r.body.baseVersion)).toEqual([3]);
    expect(onSaved).toHaveBeenCalledWith(page(4, doc("Off to the station")));

    await vi.advanceTimersByTimeAsync(10_000);
    expect(server.sent).toHaveLength(1);
  });

  it("saves at least every 5s while typing never pauses", async () => {
    const s = saver();
    let text = "";
    for (let t = 0; t < 12_000; t += 400) {
      text += "a";
      s.edit(doc(text));
      await vi.advanceTimersByTimeAsync(400);
    }
    // Typing started at 0 and never paused for 1s: saves at 5s and 10s, then the idle save after the last key.
    expect(server.sent.map((r) => r.body.doc)).toEqual([
      doc("a".repeat(13)),
      doc("a".repeat(26)),
    ]);
    expect(server.sent.map((r) => r.body.baseVersion)).toEqual([0, 1]);
    await vi.advanceTimersByTimeAsync(1000);
    expect(server.sent).toHaveLength(3);
    expect(server.sent[2]?.body).toMatchObject({
      doc: doc("a".repeat(30)),
      baseVersion: 2,
    });
  });
});

describe("the save queue", () => {
  it("keeps one save in flight per Page, and a newer edit replaces a queued one", async () => {
    const s = saver();
    const release = server.hold();
    s.edit(doc("one"));
    s.flush();
    await vi.advanceTimersByTimeAsync(0);
    expect(server.sent).toHaveLength(1);

    s.edit(doc("two"));
    s.flush();
    s.edit(doc("three"));
    await vi.advanceTimersByTimeAsync(6000);
    expect(server.sent).toHaveLength(1);

    release(json(200, page(1, doc("one"))));
    await vi.advanceTimersByTimeAsync(0);
    expect(server.sent.map((r) => r.body)).toEqual([
      { doc: doc("one"), schemaVersion: SCHEMA_VERSION, baseVersion: 0 },
      { doc: doc("three"), schemaVersion: SCHEMA_VERSION, baseVersion: 1 },
    ]);
  });

  it("retries failures with backoff, saying Not saved only once a save has failed", async () => {
    const s = saver();
    const statuses: unknown[] = [];
    s.subscribe((st) => statuses.push(st.notSaved));

    const release = server.hold();
    s.edit(doc("hello"));
    await vi.advanceTimersByTimeAsync(1000);
    expect(s.status().notSaved).toBeNull(); // in flight, and nothing is shown
    release(new Response(null, { status: 503 }));
    await vi.advanceTimersByTimeAsync(0);
    const t0 = Date.now();
    expect(s.status().notSaved).toEqual({ retryAt: t0 + 2000 });

    server.failNext(2);
    await vi.advanceTimersByTimeAsync(2000);
    expect(server.sent).toHaveLength(2);
    expect(s.status().notSaved).toEqual({ retryAt: t0 + 2000 + 4000 });
    await vi.advanceTimersByTimeAsync(4000);
    expect(server.sent).toHaveLength(3);
    expect(s.status().notSaved).toEqual({ retryAt: t0 + 6000 + 8000 });

    // Try now doesn't wait for the countdown; the save lands and the Notice clears.
    s.retryNow();
    await vi.advanceTimersByTimeAsync(0);
    expect(server.sent).toHaveLength(4);
    expect(server.sent.every((r) => r.body.baseVersion === 0)).toBe(true);
    expect(s.status().notSaved).toBeNull();
    expect(statuses.at(-1)).toBeNull();

    await vi.advanceTimersByTimeAsync(60_000);
    expect(server.sent).toHaveLength(4);
  });

  it("retries with a newer edit made while waiting, not the failed one", async () => {
    const s = saver();
    server.failNext(1);
    s.edit(doc("draft"));
    await vi.advanceTimersByTimeAsync(1000);
    s.edit(doc("draft, better"));
    await vi.advanceTimersByTimeAsync(2000);
    expect(server.sent.map((r) => r.body.doc)).toEqual([
      doc("draft"),
      doc("draft, better"),
    ]);
    expect(s.status().notSaved).toBeNull();
  });
});

describe("words kept on this device", () => {
  it("mirrors unsaved writing to storage until the server has it", async () => {
    const s = saver();
    server.failNext(1);
    s.edit(doc("Not lost"));
    // The tab closes here, mid-sentence: a new tab on the same Page gets the writing back.
    const reopened = saver(page(0), {
      fetch: vi.fn() as unknown as typeof fetch,
    });
    expect(reopened.restored).toEqual(doc("Not lost"));

    await vi.advanceTimersByTimeAsync(1000 + 2000);
    expect(server.sent).toHaveLength(2);
    expect(saver().restored).toBeNull();
  });

  it("re-sends kept writing on return, over the version it was based on", async () => {
    const first = saver(page(4));
    server.failNext(1);
    first.edit(doc("Offline thoughts"));
    await vi.advanceTimersByTimeAsync(1000);
    first.dispose();
    const before = server.sent.length;

    const back = saver(page(4));
    expect(back.restored).toEqual(doc("Offline thoughts"));
    await vi.advanceTimersByTimeAsync(0);
    expect(server.sent.slice(before).map((r) => r.body)).toEqual([
      {
        doc: doc("Offline thoughts"),
        schemaVersion: SCHEMA_VERSION,
        baseVersion: 4,
      },
    ]);
    expect(saver(page(5)).restored).toBeNull();
  });

  it("sends what's pending as a keepalive request when the tab unloads", async () => {
    const s = saver(page(2));
    s.flushOnUnload();
    expect(server.sent).toHaveLength(0);

    server.hold(); // an earlier save is still in flight
    s.edit(doc("first"));
    s.flush();
    s.edit(doc("last words"));
    s.flushOnUnload();
    expect(server.sent.at(-1)).toEqual({
      url: "/api/v1/pages/2026-11-26/my-journal",
      body: {
        doc: doc("last words"),
        schemaVersion: SCHEMA_VERSION,
        baseVersion: 2,
      },
      keepalive: true,
    });
  });

  it("never restores another account's writing", () => {
    saver(page(0), { owner: "ada@example.com" }).edit(doc("Ada's day"));
    expect(saver(page(0), { owner: "bob@example.com" }).restored).toBeNull();
    expect(saver(page(0), { owner: "ada@example.com" }).restored).toEqual(
      doc("Ada's day"),
    );
  });
});

const conflict = (current: Page) =>
  json(409, {
    error: {
      code: "version_conflict",
      message: "This Page changed on another device.",
    },
    page: current,
  });

describe("save conflicts", () => {
  it("quietly takes the server's version when it already holds this writing", async () => {
    // The unload request landed, but the kept copy still names the old version.
    const closedTab = vi.fn(() => new Promise<Response>(() => {}));
    saver(page(0), { fetch: closedTab }).edit(doc("Offline on the train"));
    // Postgres hands JSON back with its keys in its own order.
    const stored = {
      content: [
        {
          content: [{ text: "Offline on the train", type: "text" }],
          type: "paragraph",
        },
      ],
      type: "doc" as const,
    };
    server.replies.push(async () => conflict(page(1, stored)));

    const back = saver(page(1, doc("Offline on the train")));
    await vi.advanceTimersByTimeAsync(0);
    expect(back.status()).toEqual({
      notSaved: null,
      conflict: null,
      outdated: false,
    });
    expect(saver(page(1)).restored).toBeNull();

    back.edit(doc("Offline on the train, home now"));
    await vi.advanceTimersByTimeAsync(60_000);
    expect(server.sent.map((r) => r.body.baseVersion)).toEqual([0, 1]);
  });

  it("quietly takes a newer version from another device when there are no local edits", async () => {
    const onReplace = vi.fn();
    const s = saver(page(1, doc("Morning")), { onReplace });

    s.remoteChanged(page(1, doc("Morning")));
    expect(onReplace).not.toHaveBeenCalled();
    s.remoteChanged(page(3, doc("Morning, from my phone")));
    expect(onReplace).toHaveBeenCalledWith(doc("Morning, from my phone"));

    s.edit(doc("Morning, from my phone. Back on the laptop."));
    await vi.advanceTimersByTimeAsync(1000);
    expect(server.sent.at(-1)?.body.baseVersion).toBe(3);

    // With an edit waiting, a newer version isn't swapped in: the save's 409 decides.
    s.edit(doc("Unsaved thought"));
    s.remoteChanged(page(9, doc("Elsewhere")));
    expect(onReplace).toHaveBeenCalledTimes(1);
  });

  it("quietly takes the server's version on a conflict when nothing here changed", async () => {
    const onReplace = vi.fn();
    const s = saver(page(1, doc("Morning")), { onReplace });
    server.replies.push(async () => conflict(page(2, doc("Phone words"))));
    s.edit(doc("Morning!"));
    s.edit(doc("Morning")); // typed, then undone
    await vi.advanceTimersByTimeAsync(1000);

    expect(onReplace).toHaveBeenCalledWith(doc("Phone words"));
    expect(s.status().conflict).toBeNull();
    expect(saver(page(2)).restored).toBeNull();
  });

  it("stops saving when Zoop was updated, keeps the writing, and the reloaded tab sends it", async () => {
    const s = saver(page(1, doc("Before")));
    server.replies.push(async () =>
      json(409, {
        error: { code: "client_outdated", message: "Zoop was updated." },
      }),
    );
    s.edit(doc("Before the update"));
    await vi.advanceTimersByTimeAsync(1000);
    expect(s.status()).toEqual({
      notSaved: null,
      conflict: null,
      outdated: true,
    });

    s.edit(doc("Before the update, and after"));
    await vi.advanceTimersByTimeAsync(60_000);
    s.flushOnUnload();
    expect(server.sent).toHaveLength(1);

    const reloaded = saver(page(1, doc("Before")));
    expect(reloaded.restored).toEqual(doc("Before the update, and after"));
    await vi.advanceTimersByTimeAsync(0);
    expect(server.sent.at(-1)?.body).toEqual({
      doc: doc("Before the update, and after"),
      schemaVersion: SCHEMA_VERSION,
      baseVersion: 1,
    });
  });

  async function bothEdited() {
    const onReplace = vi.fn();
    const s = saver(page(1, doc("Morning")), { onReplace });
    const theirs = page(2, doc("Morning, from my phone"));
    server.replies.push(async () => conflict(theirs));
    s.edit(doc("Morning, from my laptop"));
    await vi.advanceTimersByTimeAsync(1000);
    return { s, theirs, onReplace };
  }

  it("asks which version to keep when both devices edited, and waits", async () => {
    const { s, theirs } = await bothEdited();
    expect(s.status().conflict).toEqual(theirs);
    expect(s.status().notSaved).toBeNull();

    s.edit(doc("Morning, from my laptop!"));
    await vi.advanceTimersByTimeAsync(60_000);
    expect(server.sent).toHaveLength(1);
    expect(saver(page(2)).restored).toEqual(doc("Morning, from my laptop!"));
  });

  it("Keep this version re-sends this device's writing over the other's", async () => {
    const { s, onReplace } = await bothEdited();
    s.keepMine();
    await vi.advanceTimersByTimeAsync(0);
    expect(server.sent.at(-1)?.body).toEqual({
      doc: doc("Morning, from my laptop"),
      schemaVersion: SCHEMA_VERSION,
      baseVersion: 2,
    });
    expect(s.status().conflict).toBeNull();
    expect(onReplace).not.toHaveBeenCalled();
  });

  it("Use the other swaps in the other device's version and drops this one", async () => {
    const { s, theirs, onReplace } = await bothEdited();
    s.takeTheirs();
    expect(onReplace).toHaveBeenCalledWith(theirs.doc);
    expect(s.status().conflict).toBeNull();
    expect(saver(page(2)).restored).toBeNull();

    s.edit(doc("Morning, from my phone. Then coffee."));
    await vi.advanceTimersByTimeAsync(1000);
    expect(server.sent.at(-1)?.body.baseVersion).toBe(2);
  });
});
