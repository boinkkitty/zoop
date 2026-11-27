import type { Breadcrumb, ErrorEvent } from "@sentry/react";
import { describe, expect, it } from "vitest";
import { keepBreadcrumb, scrubEvent, sentryOptions } from "./sentry";

const dirtyEvent = (): ErrorEvent => ({
  type: undefined,
  event_id: "abc123",
  level: "error",
  platform: "javascript",
  release: "r1",
  environment: "production",
  timestamp: 1764200000,
  message: "Dear diary, today I cried",
  transaction: "/search?q=dear+diary",
  exception: {
    values: [
      {
        type: "TypeError",
        value: "x is undefined",
        stacktrace: { frames: [{ filename: "/assets/app-abc.js", lineno: 1 }] },
      },
    ],
  },
  request: {
    url: "https://zoop.app/day/2026-11-26/my-journal?q=dear+diary",
    headers: { Cookie: "zoop_session=secret-cookie" },
    data: '{"text":"Dear diary"}',
  },
  user: { id: "user-1", email: "ada@example.com", ip_address: "203.0.113.9" },
  extra: { page: "Dear diary, today I cried" },
  tags: { route: "/day/$day/$notebook", title: "Dear diary" },
  contexts: {
    browser: { name: "Chrome", version: "142" },
    page: { text: "Dear diary" },
  },
  breadcrumbs: [
    { category: "console", message: "Dear diary" },
    { category: "navigation", data: { from: "/", to: "/search?q=dear+diary" } },
  ],
});

describe("scrubEvent", () => {
  it("keeps only the error and where it happened", () => {
    const out = scrubEvent(dirtyEvent());

    const raw = JSON.stringify(out);
    for (const secret of [
      "diary",
      "ada@example.com",
      "203.0.113.9",
      "secret-cookie",
    ]) {
      expect(raw).not.toContain(secret);
    }
    expect(out.request).toBeUndefined();
    expect(out.extra).toBeUndefined();
    expect(out.user).toEqual({ id: "user-1" });
    expect(out.tags).toEqual({ route: "/day/$day/$notebook" });
    expect(out.contexts).toEqual({
      browser: { name: "Chrome", version: "142" },
    });
    expect(out.exception?.values?.[0]?.value).toBe("x is undefined");
    expect(out.event_id).toBe("abc123");
    expect(out.breadcrumbs).toEqual([
      { category: "navigation", data: { from: "/", to: "/search" } },
    ]);
  });
});

describe("keepBreadcrumb", () => {
  it.each([
    "console",
    "ui.click",
    "ui.input",
    "ui.keypress",
    "sentry.event",
    undefined,
  ])("drops %s breadcrumbs, which can carry what was typed", (category) => {
    expect(keepBreadcrumb({ category, message: "Dear diary" })).toBeNull();
  });

  it("keeps API calls without their query strings", () => {
    const crumb: Breadcrumb = {
      category: "fetch",
      type: "http",
      data: {
        method: "GET",
        url: "/api/v1/search?q=dear+diary",
        status_code: 200,
        request_body_size: 42,
      },
    };
    expect(keepBreadcrumb(crumb)).toEqual({
      category: "fetch",
      type: "http",
      data: { method: "GET", url: "/api/v1/search", status_code: 200 },
    });
  });

  it("keeps navigations without their query strings", () => {
    expect(
      keepBreadcrumb({
        category: "navigation",
        message: "Dear diary",
        data: { from: "/day/2026-11-26?q=x", to: "/search?q=dear" },
      }),
    ).toEqual({
      category: "navigation",
      data: { from: "/day/2026-11-26", to: "/search" },
    });
  });
});

describe("sentryOptions", () => {
  const opts = sentryOptions("https://public@o0.ingest.sentry.io/0");

  // SDK v11 replaced sendDefaultPii with dataCollection: switch every kind off.
  it("collects no personal data and no replays", () => {
    expect(opts.dataCollection).toEqual({
      userInfo: false,
      cookies: false,
      httpHeaders: false,
      httpBodies: [],
      urlQueryParams: false,
      stackFrameVariables: false,
    });
    expect(opts.replaysSessionSampleRate ?? 0).toBe(0);
    expect(opts.replaysOnErrorSampleRate ?? 0).toBe(0);
  });

  it("scrubs every event and breadcrumb", () => {
    expect(opts.beforeSend).toBe(scrubEvent);
    expect(opts.beforeBreadcrumb).toBe(keepBreadcrumb);
  });

  it("leaves out replay and the console and DOM breadcrumb recorders", () => {
    const integrations = opts.integrations;
    if (typeof integrations !== "function") throw new Error("want a function");
    const names = integrations([]).map((i) => i.name);
    for (const banned of [
      "Replay",
      "Console",
      "CaptureConsole",
      "HttpContext",
    ]) {
      expect(names).not.toContain(banned);
    }
    expect(names).toContain("Breadcrumbs");
  });
});
