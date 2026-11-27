import { expect, type Page, test } from "./fixtures";

// Day 1: three one-time coach marks, and the rail with a single Day.

const PASSWORD = "correct horse battery staple";

async function signUp(page: Page) {
  const email = `e2e+${Date.now()}${Math.floor(Math.random() * 1e6)}@example.com`;
  await page.goto("/signup");
  await page.getByLabel("Email").fill(email);
  await page.getByLabel("Password").fill(PASSWORD);
  await page.getByRole("button", { name: "Create account" }).click();
  await expect(page).toHaveURL(/\/day\/[\d-]+\/my-journal$/);
  return email;
}

const coachMark = (page: Page) => page.getByRole("dialog", { name: /.+/ });

test("a new signup is walked through three coach marks, one at a time", async ({
  page,
}) => {
  await signUp(page);

  const rail = page.getByRole("navigation", { name: "Days" });
  await expect(rail).toContainText("You started here · today");
  await expect(rail).toContainText(
    "Every Day gets an Island here. Its Title is the first line you write.",
  );

  await expect(coachMark(page)).toHaveCount(1);
  await expect(coachMark(page)).toHaveAccessibleName("Today’s Page");
  await expect(coachMark(page)).toContainText("1 of 3");
  await coachMark(page).getByRole("button", { name: "Next" }).click();

  await expect(coachMark(page)).toHaveAccessibleName("Two Notebooks");
  await expect(coachMark(page)).toContainText("2 of 3");
  await coachMark(page).getByRole("button", { name: "Next" }).click();

  await expect(coachMark(page)).toHaveAccessibleName("Your Days");
  await expect(coachMark(page)).toContainText("3 of 3");
  await expect(coachMark(page)).toContainText("02:00");
  await coachMark(page).getByRole("button", { name: "Got it" }).click();

  await expect(coachMark(page)).toHaveCount(0);
  await page.reload();
  await expect(page.getByRole("textbox", { name: "Page" })).toBeVisible();
  await expect(coachMark(page)).toHaveCount(0);
});

test("typing ends the coach marks for good", async ({ page }) => {
  await signUp(page);
  await expect(coachMark(page)).toContainText("1 of 3");

  const saved = page.waitForResponse(
    (res) => res.request().method() === "PUT" && res.ok(),
  );
  await page.getByRole("textbox", { name: "Page" }).click();
  await page.keyboard.type("First day");
  await expect(coachMark(page)).toHaveCount(0);

  await saved;
  await page.reload();
  await expect(page.getByRole("textbox", { name: "Page" })).toBeVisible();
  await expect(coachMark(page)).toHaveCount(0);
});

test("an existing user on a new browser sees no coach marks", async ({
  page,
  browser,
  baseURL,
  extraHTTPHeaders,
}) => {
  const email = await signUp(page);
  const saved = page.waitForResponse(
    (res) => res.request().method() === "PUT" && res.ok(),
  );
  await page.getByRole("textbox", { name: "Page" }).click();
  await page.keyboard.type("Already writing");
  await saved;

  const fresh = await browser.newContext({ baseURL, extraHTTPHeaders });
  const other = await fresh.newPage();
  await other.goto("/signin");
  await other.getByLabel("Email").fill(email);
  await other.getByLabel("Password").fill(PASSWORD);
  await other.getByRole("button", { name: "Sign in" }).click();
  await expect(other.getByRole("textbox", { name: "Page" })).toBeVisible();
  await expect(
    other.getByRole("navigation", { name: "Days" }),
  ).not.toContainText("Every Day gets an Island here.");
  await expect(coachMark(other)).toHaveCount(0);
  await fresh.close();
});
