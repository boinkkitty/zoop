// Error reporting for the React app. Like the Go side, it sends only an
// allowlist: the error, its stack and where it happened. Never anyone's
// writing, what they typed, search queries or their email.
import {
  type Breadcrumb,
  type BrowserOptions,
  breadcrumbsIntegration,
  type ErrorEvent,
  init,
} from "@sentry/react";

const allowedTags = new Set(["route"]);
const allowedContexts = new Set(["browser", "os", "device", "trace"]);

/** Rebuilds an event from allowlisted fields, so nothing else leaves the browser. */
export function scrubEvent(event: ErrorEvent): ErrorEvent {
  const out: ErrorEvent = {
    type: undefined,
    event_id: event.event_id,
    level: event.level,
    platform: event.platform,
    release: event.release,
    dist: event.dist,
    environment: event.environment,
    timestamp: event.timestamp,
    sdk: event.sdk,
    fingerprint: event.fingerprint,
    exception: event.exception,
    debug_meta: event.debug_meta, // ties the stack to uploaded source maps
  };
  if (event.user?.id !== undefined) out.user = { id: event.user.id };
  if (event.tags) {
    out.tags = Object.fromEntries(
      Object.entries(event.tags).filter(([k]) => allowedTags.has(k)),
    );
  }
  if (event.contexts) {
    out.contexts = Object.fromEntries(
      Object.entries(event.contexts).filter(([k]) => allowedContexts.has(k)),
    );
  }
  if (event.breadcrumbs) {
    out.breadcrumbs = event.breadcrumbs
      .map(keepBreadcrumb)
      .filter((b): b is Breadcrumb => b !== null);
  }
  return out;
}

/**
 * Keeps navigations and API calls, minus query strings (search queries live
 * there). Drops everything else: console and DOM breadcrumbs can carry what
 * was typed into the editor.
 */
export function keepBreadcrumb(crumb: Breadcrumb): Breadcrumb | null {
  const data = crumb.data ?? {};
  switch (crumb.category) {
    case "navigation":
      return {
        ...base(crumb),
        data: { from: stripQuery(data.from), to: stripQuery(data.to) },
      };
    case "fetch":
    case "xhr":
      return {
        ...base(crumb),
        data: {
          method: data.method,
          url: stripQuery(data.url),
          status_code: data.status_code,
        },
      };
    default:
      return null;
  }
}

function base(crumb: Breadcrumb): Breadcrumb {
  const out: Breadcrumb = { category: crumb.category };
  if (crumb.type !== undefined) out.type = crumb.type;
  if (crumb.level !== undefined) out.level = crumb.level;
  if (crumb.timestamp !== undefined) out.timestamp = crumb.timestamp;
  return out;
}

function stripQuery(url: unknown): string | undefined {
  if (typeof url !== "string") return undefined;
  return url.split(/[?#]/, 1)[0];
}

/** The client's options: no personal data, no replay, scrubbed events and breadcrumbs. */
export function sentryOptions(dsn: string): BrowserOptions {
  return {
    dsn,
    release: import.meta.env.VITE_SENTRY_RELEASE,
    environment: import.meta.env.MODE,
    // v11's replacement for sendDefaultPii: false. Collect none of it.
    dataCollection: {
      userInfo: false,
      cookies: false,
      httpHeaders: false,
      httpBodies: [],
      urlQueryParams: false,
      stackFrameVariables: false,
    },
    beforeSend: scrubEvent,
    beforeBreadcrumb: keepBreadcrumb,
    integrations: (defaults) => [
      // Console breadcrumbs come from the Console integration; HttpContext
      // adds the page URL and headers.
      ...defaults.filter(
        (i) => !["Breadcrumbs", "Console", "HttpContext"].includes(i.name),
      ),
      breadcrumbsIntegration({
        dom: false,
        fetch: true,
        xhr: true,
        history: true,
        sentry: false,
      }),
    ],
  };
}

/** Starts error reporting when the build was given a DSN; otherwise does nothing. */
export function initSentry() {
  const dsn = import.meta.env.VITE_SENTRY_DSN;
  if (dsn) init(sentryOptions(dsn));
}
