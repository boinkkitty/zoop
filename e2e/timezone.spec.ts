import { type Browser, expect, type Page, test } from "@playwright/test";

// The timezone Notice: an account made in Tokyo, opened by a browser in London.
test.use({ timezoneId: "Asia/Tokyo" });

const notice = "Looks like you're on London time.";

// signUpThenFlyToLondon signs up here in Tokyo, then opens the same session in a London browser.
async function signUpThenFlyToLondon(page: Page, browser: Browser) {
  await page.goto("/signup");
  await page
    .getByLabel("Email")
    .fill(`e2e+${Date.now()}${Math.floor(Math.random() * 1e6)}@example.com`);
  await page.getByLabel("Password").fill("correct horse battery staple");
  await page.getByRole("button", { name: "Create account" }).click();
  await expect(page).toHaveURL(/\/day\/\d{4}-\d{2}-\d{2}\/my-journal$/);
  await expect(page.getByText(notice)).toHaveCount(0);

  const london = await browser.newContext({
    timezoneId: "Europe/London",
    storageState: await page.context().storageState(),
  });
  const there = await london.newPage();
  await there.goto("/");
  return there;
}

test("Switch starts the Days on the browser's time from tomorrow", async ({
  page,
  browser,
}) => {
  const london = await signUpThenFlyToLondon(page, browser);
  await expect(london.getByText(notice)).toBeVisible();

  const patched = london.waitForResponse(
    (res) => res.request().method() === "PATCH" && res.url().endsWith("/me"),
  );
  await london.getByRole("button", { name: "Switch" }).click();
  const me = await (await patched).json();
  expect(me.pendingTimezone).toBe("Europe/London");
  await expect(london.getByText(notice)).toHaveCount(0);

  // Answered on the server: no other London browser asks again.
  await london.evaluate(() => localStorage.clear());
  await london.reload();
  await expect(london.getByRole("textbox", { name: "Page" })).toBeVisible();
  await expect(london.getByText(notice)).toHaveCount(0);
});

test("Keep is remembered for that zone in this browser", async ({
  page,
  browser,
}) => {
  const london = await signUpThenFlyToLondon(page, browser);
  await london.getByRole("button", { name: "Keep Tokyo" }).click();
  await expect(london.getByText(notice)).toHaveCount(0);

  await london.reload();
  await expect(london.getByRole("textbox", { name: "Page" })).toBeVisible();
  await expect(london.getByText(notice)).toHaveCount(0);
  const me = await (await london.request.get("/api/v1/me")).json();
  expect(me.timezone).toBe("Asia/Tokyo");
  expect(me.pendingTimezone).toBeNull();
});
