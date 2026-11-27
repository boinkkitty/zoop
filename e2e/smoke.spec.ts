import { expect, test } from "./fixtures";

test("the app shell loads and healthz is green", async ({ page, request }) => {
  await page.goto("/");
  await expect(page).toHaveTitle("Zoop");
  await expect(page.getByRole("main")).toBeVisible();

  await page.goto("/some/client/route");
  await expect(page).toHaveTitle("Zoop");
  await expect(page.getByRole("main")).toBeVisible();

  const health = await request.get("/api/healthz");
  expect(health.status()).toBe(200);
});
