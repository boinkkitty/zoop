import { type BrowserContext, expect, test } from "@playwright/test";

// The 02:00 lock, with time moved on both sides: Playwright's clock in the browser,
// and the test-only server clock (built with -tags testclock), shifted per browser by a cookie.
test.use({ timezoneId: "Asia/Tokyo" });

const DAY_MS = 24 * 3600 * 1000;
const tokyoDate = (ms: number) =>
  new Date(ms + 9 * 3600 * 1000).toISOString().slice(0, 10);
const at = (day: string, time: string) => Date.parse(`${day}T${time}+09:00`);

// serverNow makes this browser's requests see the server's clock at ms, flowing from there.
async function serverNow(context: BrowserContext, baseURL: string, ms: number) {
  await context.addCookies([
    {
      name: "zoop_test_now_offset",
      value: String(Math.round(ms - Date.now())),
      url: baseURL,
    },
  ]);
}

test("yesterday's Page turns read only at 02:00 without losing the last words", async ({
  page,
  context,
  baseURL,
}) => {
  const probe = await page.request.get("/api/healthz");
  test.skip(
    !probe.headers()["zoop-test-clock"],
    "the server wasn't built with -tags testclock",
  );
  const base = baseURL ?? "";

  // Sign up in the evening of a Day ahead of the real one, so cookies stay fresh.
  const day = tokyoDate(Date.now() + DAY_MS);
  const next = tokyoDate(at(day, "12:00:00") + DAY_MS);
  await serverNow(context, base, at(day, "21:00:00"));
  await page.clock.install({ time: at(day, "21:00:00") });
  await page.goto("/signup");
  await page
    .getByLabel("Email")
    .fill(`e2e+${Date.now()}${Math.floor(Math.random() * 1e6)}@example.com`);
  await page.getByLabel("Password").fill("correct horse battery staple");
  await page.getByRole("button", { name: "Create account" }).click();
  await expect(page).toHaveURL(new RegExp(`/day/${day}/my-journal$`));

  // Just before 02:00 the next night, yesterday is still open.
  await serverNow(context, base, at(next, "01:59:50"));
  await page.clock.setSystemTime(at(next, "01:59:50"));
  await page.goto(`/day/${day}/my-journal`);
  await expect(page.getByText("Still open until 02:00")).toBeVisible();
  const editor = page.getByRole("textbox", { name: "Page" });

  // Typed half a second before 02:00, with no pause for the 1s autosave:
  // the lock itself has to send the words.
  await page.clock.pauseAt(at(next, "01:59:59.500"));
  await editor.click();
  await page.keyboard.type("Last words before two");
  const saved = page.waitForResponse(
    (res) => res.request().method() === "PUT" && res.ok(),
  );
  await page.clock.runFor(1000);
  await saved;
  await page.clock.resume();
  await expect(editor).toHaveAttribute("contenteditable", "false");
  await expect(page.getByText("Read only")).toBeVisible();
  const closed = page.getByText(
    "Yesterday closed at 02:00. Everything you wrote is saved.",
  );
  await expect(closed).toBeVisible();

  // Reopened after 02:00, the words are there and the Page stays shut.
  await serverNow(context, base, at(next, "02:00:30"));
  await page.clock.setSystemTime(at(next, "02:00:30"));
  await page.reload();
  await expect(editor.locator("p").first()).toHaveText("Last words before two");
  await expect(editor).toHaveAttribute("contenteditable", "false");
  await expect(closed).toBeVisible();

  await page.getByRole("button", { name: "Go to today" }).click();
  await expect(page).toHaveURL(new RegExp(`/day/${next}/my-journal$`));
  await expect(closed).toBeHidden();

  // Dismissed, it stays dismissed for that Day.
  await page.goBack();
  await closed.locator("..").getByRole("button", { name: "Dismiss" }).click();
  await expect(closed).toBeHidden();
  await page.reload();
  await expect(editor.locator("p").first()).toHaveText("Last words before two");
  await expect(closed).toBeHidden();
});
