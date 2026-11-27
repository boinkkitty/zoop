// The Playwright test, with each test arriving from its own client IP.
//
// Signups are rate limited per IP (ratelimit.SignupsPerIP: 5 an hour), and the
// suite signs up more often than that. The e2e server runs with TRUSTED_PROXIES=1,
// as behind Railway, so it takes the last X-Forwarded-For hop as the client IP: a
// random one per test keeps every test under the limit, even across runs against
// one database. (A server set up via ZOOP_E2E_BASE_URL needs TRUSTED_PROXIES=1 too.)
import { test as base } from "@playwright/test";

const octet = () => Math.floor(Math.random() * 254) + 1;

export const test = base.extend({
  extraHTTPHeaders: async ({ extraHTTPHeaders }, use) => {
    await use({
      ...extraHTTPHeaders,
      "X-Forwarded-For": `10.${octet()}.${octet()}.${octet()}`,
    });
  },
});

export type { BrowserContext, Locator, Page } from "@playwright/test";
export { expect } from "@playwright/test";
