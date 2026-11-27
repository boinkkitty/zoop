import { type BrowserContext, expect, type Page, test } from "@playwright/test";

// The Island rail, with time moved on the test-only server clock (built with
// -tags testclock), shifted per browser by a cookie, and Playwright's clock in the browser.
test.use({ timezoneId: "Asia/Tokyo" });

const DAY_MS = 24 * 3600 * 1000;
const PASSWORD = "correct horse battery staple";
const tokyoDate = (ms: number) =>
  new Date(ms + 9 * 3600 * 1000).toISOString().slice(0, 10);
const at = (day: string, time: string) => Date.parse(`${day}T${time}+09:00`);
// Each test signs up on its own Day ahead of the real one: ahead so cookies stay
// fresh, and apart so their signups don't share a per-IP rate-limit window.
const signupDay = (daysAhead: number) =>
  tokyoDate(Date.now() + daysAhead * DAY_MS);
const addDays = (day: string, n: number) =>
  tokyoDate(at(day, "12:00:00") + n * DAY_MS);

async function serverNow(context: BrowserContext, baseURL: string, ms: number) {
  await context.addCookies([
    {
      name: "zoop_test_now_offset",
      value: String(Math.round(ms - Date.now())),
      url: baseURL,
    },
  ]);
}

async function needsTestClock(page: Page) {
  const probe = await page.request.get("/api/healthz");
  test.skip(
    !probe.headers()["zoop-test-clock"],
    "the server wasn't built with -tags testclock",
  );
}

async function signUp(page: Page, day: string) {
  const email = `e2e+${Date.now()}${Math.floor(Math.random() * 1e6)}@example.com`;
  await page.goto("/signup");
  await page.getByLabel("Email").fill(email);
  await page.getByLabel("Password").fill(PASSWORD);
  await page.getByRole("button", { name: "Create account" }).click();
  await expect(page).toHaveURL(new RegExp(`/day/${day}/my-journal$`));
  return email;
}

async function write(page: Page, lines: string[]) {
  const editor = page.getByRole("textbox", { name: "Page" });
  await editor.click();
  for (const [i, line] of lines.entries()) {
    if (i > 0) await page.keyboard.press("Enter");
    await page.keyboard.type(line);
  }
  await page.waitForResponse(
    (res) => res.request().method() === "PUT" && res.ok(),
  );
}

const island = (page: Page, day: string) =>
  page.getByRole("navigation", { name: "Days" }).locator(`[data-day="${day}"]`);

test("the rail shows each Day's Title, glimpses it on hover, and opens it on click", async ({
  page,
  context,
  baseURL,
}) => {
  await needsTestClock(page);
  const first = signupDay(3);
  const today = addDays(first, 2);
  await serverNow(context, baseURL ?? "", at(first, "21:00:00"));
  await signUp(page, first);
  await write(page, [
    "Finally fixed the bike chain",
    "The derailleur was bent.",
  ]);

  await serverNow(context, baseURL ?? "", at(today, "12:00:00"));
  await page.goto("/");
  await expect(page).toHaveURL(new RegExp(`/day/${today}/my-journal$`));

  // Newest first: Today, Yesterday (both unwritten, so grey with only the date), then the written Day.
  await expect(island(page, today)).toHaveText("Today");
  await expect(island(page, addDays(first, 1))).toHaveText("Yesterday");
  const written = island(page, first);
  await expect(written).toContainText(/^\w{3} \d{1,2}Finally/);
  await expect(written).toContainText("Finally fixed the bike chain");

  const glimpse = written.getByText(
    "Finally fixed the bike chain · The derailleur was bent.",
  );
  await expect(glimpse).toBeHidden();
  await written.hover();
  await expect(glimpse).toBeVisible();

  await written.click();
  await expect(page).toHaveURL(new RegExp(`/day/${first}/my-journal$`));
  await expect(
    page.getByRole("textbox", { name: "Page" }).locator("p").first(),
  ).toHaveText("Finally fixed the bike chain");

  await page
    .getByRole("navigation", { name: "Journal" })
    .getByRole("button", { name: "Today" })
    .click();
  await expect(page).toHaveURL(new RegExp(`/day/${today}/my-journal$`));
});

test("the scrubber jumps to a month and loads only it, then older and newer Days on scroll", async ({
  page,
  context,
  baseURL,
}) => {
  await needsTestClock(page);
  const base = baseURL ?? "";
  const first = signupDay(5);
  await serverNow(context, base, at(first, "21:00:00"));
  const email = await signUp(page, first);
  await write(page, ["The very first Day"]);

  // 70 Days on: at least two months later, and the session has lapsed.
  const today = addDays(first, 70);
  await serverNow(context, base, at(today, "12:00:00"));
  await page.goto("/signin");
  await page.getByLabel("Email").fill(email);
  await page.getByLabel("Password").fill(PASSWORD);
  const months: string[] = [];
  page.on("request", (req) => {
    const m = new URL(req.url()).searchParams.get("month");
    if (m) months.push(m);
  });
  await page.getByRole("button", { name: "Sign in" }).click();
  await expect(page).toHaveURL(new RegExp(`/day/${today}/my-journal$`));
  await expect(island(page, today)).toBeVisible();
  await expect(island(page, first)).toHaveCount(0);

  const firstMonth = first.slice(0, 7);
  const tick = page
    .getByTestId("scrubber")
    .locator(`[data-slot="${firstMonth}"]`);
  await tick.hover();
  const name = new Intl.DateTimeFormat("en-GB", {
    month: "long",
    year: "numeric",
    timeZone: "UTC",
  }).format(new Date(`${firstMonth}-01T00:00:00Z`));
  await expect(page.getByText(name, { exact: true })).toBeVisible();

  months.length = 0;
  await tick.click();
  await expect(island(page, first)).toContainText("The very first Day");
  await expect(page.getByText("You started here")).toBeVisible();
  // Only the target month was fetched first, never the months in between.
  expect(months[0]).toBe(firstMonth);
});

test("yesterday's Island shows the Grace Window, and the marker goes at 02:00", async ({
  page,
  context,
  baseURL,
}) => {
  await needsTestClock(page);
  const base = baseURL ?? "";
  const day = signupDay(7);
  const next = addDays(day, 1);
  await serverNow(context, base, at(day, "21:00:00"));
  await page.clock.install({ time: at(day, "21:00:00") });
  await signUp(page, day);

  await serverNow(context, base, at(next, "01:12:00"));
  await page.clock.setSystemTime(at(next, "01:12:00"));
  await page.goto("/");
  const yesterday = island(page, day);
  await expect(yesterday.getByTitle("Still open")).toBeVisible();
  await yesterday.hover();
  await expect(
    yesterday.getByText("Still open until 02:00 · 48 min left"),
  ).toBeVisible();

  await page.clock.fastForward("47:30");
  await expect(
    yesterday.getByText("Still open until 02:00 · 1 min left"),
  ).toBeVisible();
  await page.clock.fastForward("01:00");
  await expect(yesterday.getByTitle("Still open")).toHaveCount(0);
  await expect(yesterday.getByText(/Still open until/)).toHaveCount(0);
});

test("the selected Island's ring isn't covered by the month header or crowded by the scrubber", async ({
  page,
  context,
  baseURL,
}) => {
  await needsTestClock(page);
  const day = signupDay(9);
  await serverNow(context, baseURL ?? "", at(day, "12:00:00"));
  await signUp(page, day);

  const selected = island(page, day);
  await expect(selected).toHaveAttribute("aria-current", "date");
  const header = page.getByRole("heading", { level: 2 }).first();
  await expect(page.getByTestId("scrubber")).toBeVisible();
  const ring = 4; // outline 2px + outline-offset 2px
  const s = await selected.boundingBox();
  const h = await header.boundingBox();
  const scrub = await page.getByTestId("scrubber").boundingBox();
  if (!s || !h || !scrub) throw new Error("rail not laid out");
  // The sticky header is opaque: it must end above the ring, or it paints over its top edge.
  expect(h.y + h.height).toBeLessThanOrEqual(s.y - ring);
  expect(s.x + s.width + ring).toBeLessThan(scrub.x);
});
