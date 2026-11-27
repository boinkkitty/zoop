// The Playwright test, with each test arriving from its own client IP.
//
// Signups are rate limited per IP (ratelimit.SignupsPerIP: 5 an hour), and the
// suite signs up more often than that. The server takes the client IP from the
// last X-Forwarded-For hop (the one Railway's proxy appends), so a random one per
// test keeps every test under the limit, even across runs against one database.
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
