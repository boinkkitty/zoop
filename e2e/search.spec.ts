import { type BrowserContext, expect, type Page, test } from "./fixtures";

// Search, with time moved on the test-only server clock (built with -tags testclock)
// so there's more than one Day to find.
test.use({ timezoneId: "Asia/Tokyo" });

const DAY_MS = 24 * 3600 * 1000;
const PASSWORD = "correct horse battery staple";
const tokyoDate = (ms: number) =>
  new Date(ms + 9 * 3600 * 1000).toISOString().slice(0, 10);
const at = (day: string, time: string) => Date.parse(`${day}T${time}+09:00`);
// Ahead of the real Day so cookies stay fresh, and apart from rail.spec.ts's Days.
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

test("search finds a Page, opens it with the matches marked, and Esc goes back to the rail", async ({
  page,
  context,
  baseURL,
}) => {
  await needsTestClock(page);
  const base = baseURL ?? "";
  const first = signupDay(20);
  const today = addDays(first, 2);
  await serverNow(context, base, at(first, "21:00:00"));
  await page.goto("/signup");
  await page
    .getByLabel("Email")
    .fill(`e2e+${Date.now()}${Math.floor(Math.random() * 1e6)}@example.com`);
  await page.getByLabel("Password").fill(PASSWORD);
  await page.getByRole("button", { name: "Create account" }).click();
  await expect(page).toHaveURL(new RegExp(`/day/${first}/my-journal$`));
  await write(page, ["Sunday baking", "Walked to the bakery with Mum."]);

  await serverNow(context, base, at(today, "12:00:00"));
  await page.goto("/");
  await expect(page).toHaveURL(new RegExp(`/day/${today}/my-journal$`));
  const rail = page.getByRole("navigation", { name: "Days" });
  await expect(rail).toBeVisible();

  // ⌘K focuses search; results replace the rail and hide the scrubber.
  await page.keyboard.press("ControlOrMeta+k");
  const box = page.getByRole("searchbox", { name: "Search" });
  await expect(box).toBeFocused();
  await page.keyboard.type("bak mum");
  const results = page.getByRole("list", { name: "Search results" });
  const hit = results.getByRole("button").first();
  await expect(hit).toContainText("Sunday baking");
  await expect(hit.locator("mark")).toHaveText(["baking", "bakery", "Mum"]);
  await expect(rail).toBeHidden();
  await expect(page.getByTestId("scrubber")).toBeHidden();

  // Enter opens it: the result keeps a bar, the Page shows every match in blue.
  await page.keyboard.press("Enter");
  await expect(page).toHaveURL(new RegExp(`/day/${first}/my-journal$`));
  await expect(hit).toHaveAttribute("aria-current", "page");
  const editor = page.getByRole("textbox", { name: "Page" });
  await expect(editor.locator("[data-search-match]")).toHaveText([
    "baking",
    "bakery",
    "Mum",
  ]);
  await expect(page.getByText("Matches for “bak mum”")).toBeVisible();

  // Esc clears the highlights; the results stay while reading.
  await page.keyboard.press("Escape");
  await expect(editor.locator("[data-search-match]")).toHaveCount(0);
  await expect(page.getByText("Matches for “bak mum”")).toBeHidden();
  await expect(results).toBeVisible();

  // Esc in the box clears the query and brings the rail back at the opened Day.
  await box.click();
  await page.keyboard.press("Escape");
  await expect(box).toHaveValue("");
  await expect(rail).toBeVisible();
  await expect(rail.locator(`[data-day="${first}"]`)).toHaveAttribute(
    "aria-current",
    "date",
  );
});

test("a query that matches nothing says so", async ({ page }) => {
  await page.goto("/signup");
  await page
    .getByLabel("Email")
    .fill(`e2e+${Date.now()}${Math.floor(Math.random() * 1e6)}@example.com`);
  await page.getByLabel("Password").fill(PASSWORD);
  await page.getByRole("button", { name: "Create account" }).click();
  await expect(page).toHaveURL(/\/day\/.*\/my-journal$/);
  await page.getByRole("searchbox", { name: "Search" }).fill("zebra");
  await expect(page.getByText("No Pages match “zebra”")).toBeVisible();
});
