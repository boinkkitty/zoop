import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { Page } from "../api";
import { createPageSaver, type SaverDeps } from "./saver";

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
  schemaVersion: 1,
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
  return { sent, fetch, replies };
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
        body: { doc: doc("Ra"), schemaVersion: 1, baseVersion: 0 },
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
