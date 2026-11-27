import { expect, test } from "@playwright/test";

test.use({ timezoneId: "Asia/Tokyo" });

test("what's written on today's Page autosaves and survives a reload", async ({
  page,
}) => {
  const email = `e2e+${Date.now()}${Math.floor(Math.random() * 1e6)}@example.com`;
  await page.goto("/signup");
  await page.getByLabel("Email").fill(email);
  await page.getByLabel("Password").fill("correct horse battery staple");
  await page.getByRole("button", { name: "Create account" }).click();
  await expect(page).toHaveURL(/\/day\/\d{4}-\d{2}-\d{2}\/my-journal$/);

  const editor = page.getByRole("textbox", { name: "Page" });
  await expect(editor).toBeVisible();
  await editor.click();
  await page.keyboard.type("A quiet Thursday");
  await page.keyboard.press("Enter");
  await page.keyboard.type("Rain all morning, then **sun**.");

  // Autosave is silent: wait for the request rather than anything on screen.
  await page.waitForResponse(
    (res) =>
      res.request().method() === "PUT" &&
      res.url().includes("/api/v1/pages/") &&
      res.ok(),
  );
  await page.reload();

  await expect(editor.locator("p").first()).toHaveText("A quiet Thursday");
  // Typed Markdown stays literal.
  await expect(editor.locator("p").nth(1)).toHaveText(
    "Rain all morning, then **sun**.",
  );
});
