import { expect, test } from "./fixtures";

// The public privacy page: readable without signing in, and reachable from both auth screens.

for (const screen of ["/signin", "/signup"]) {
  test(`a visitor on ${screen} can open the privacy page without signing in`, async ({
    page,
  }) => {
    await page.goto(screen);
    await page.getByRole("link", { name: "Privacy" }).click();
    await expect(page).toHaveURL(/\/privacy$/);
    await expect(page.getByRole("heading", { level: 1 })).toHaveText("Privacy");
  });
}

test("the privacy page names every subprocessor and never claims end-to-end encryption", async ({
  page,
}) => {
  await page.goto("/privacy");
  await expect(page).toHaveURL(/\/privacy$/);
  const body = page.locator("main");
  for (const name of [
    "Railway",
    "Cloudflare R2",
    "Resend",
    "Sentry",
    "Google",
  ]) {
    await expect(body).toContainText(name);
  }
  await expect(body).toContainText(
    "encrypted in transit and at rest by our hosting providers",
  );
  await expect(body).toContainText(
    "erased within a year, and gone from backups 90 days after that",
  );
  await expect(body).not.toContainText(/end-to-end/i);
});
