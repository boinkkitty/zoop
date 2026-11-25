import { expect, test } from "@playwright/test";

test("the app shell loads and healthz is green", async ({ page, request }) => {
  await page.goto("/");
  await expect(page.getByRole("main")).toHaveText("Zoop");

  await page.goto("/some/client/route");
  await expect(page.getByRole("main")).toHaveText("Zoop");

  const health = await request.get("/api/healthz");
  expect(health.status()).toBe(200);
});
