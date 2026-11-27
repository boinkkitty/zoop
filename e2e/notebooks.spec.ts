import { expect, type Page, test } from "@playwright/test";

test.use({ timezoneId: "Asia/Tokyo" });

async function signUp(page: Page) {
  const email = `e2e+${Date.now()}${Math.floor(Math.random() * 1e6)}@example.com`;
  await page.goto("/signup");
  await page.getByLabel("Email").fill(email);
  await page.getByLabel("Password").fill("correct horse battery staple");
  await page.getByRole("button", { name: "Create account" }).click();
  await expect(page).toHaveURL(/\/day\/\d{4}-\d{2}-\d{2}\/my-journal$/);
  return new URL(page.url()).pathname.split("/")[2];
}

// "METHOD notebook" for each Page request, in the order the browser sent them.
function recordPageRequests(page: Page): string[] {
  const seen: string[] = [];
  page.on("request", (req) => {
    const m = req.url().match(/\/api\/v1\/pages\/[^/]+\/([^/?]+)/);
    if (m) seen.push(`${req.method()} ${m[1]}`);
  });
  return seen;
}

test("the Notebook index switches Notebooks, each at its own URL", async ({
  page,
}) => {
  const day = await signUp(page);
  const requests = recordPageRequests(page);
  const index = page.getByRole("navigation", { name: "Notebooks" });
  const journalTab = index.getByRole("link", { name: "My Journal" });
  const learnedTab = index.getByRole("link", { name: "I Learned" });
  const editor = page.getByRole("textbox", { name: "Page" });

  // An inset column on the sheet's right edge, the open Notebook marked.
  const box = await index.boundingBox();
  expect(box?.width).toBeCloseTo(52, 0);
  await expect(journalTab).toHaveAttribute("aria-current", "page");
  await expect(learnedTab).not.toHaveAttribute("aria-current");
  await expect(journalTab).toHaveCSS("writing-mode", "vertical-rl");
  await expect(journalTab).toHaveCSS("text-transform", "uppercase");

  // Typing then switching straight away: the edit is sent before I Learned loads.
  // A dispatched click keeps focus in the editor, so it's the route change that flushes, not blur.
  await editor.click();
  await page.keyboard.type("A quiet Thursday");
  await learnedTab.dispatchEvent("click");
  await expect(page).toHaveURL(new RegExp(`/day/${day}/i-learned$`));
  expect(requests.indexOf("PUT my-journal")).toBeGreaterThanOrEqual(0);
  expect(requests.indexOf("PUT my-journal")).toBeLessThan(
    requests.indexOf("GET i-learned"),
  );
  await expect(learnedTab).toHaveAttribute("aria-current", "page");

  // I Learned has no Title: its first line isn't bold.
  await expect(editor.locator("p").first()).toHaveText("");
  await editor.click();
  await page.keyboard.type("Octopuses have three hearts.");
  await expect(editor.locator("p").first()).toHaveCSS("font-weight", "400");

  // Back restores My Journal with what was written; forward, I Learned.
  await page.goBack();
  await expect(page).toHaveURL(new RegExp(`/day/${day}/my-journal$`));
  await expect(editor.locator("p").first()).toHaveText("A quiet Thursday");
  await expect(editor.locator("p").first()).toHaveCSS("font-weight", "650");
  await page.goForward();
  await expect(page).toHaveURL(new RegExp(`/day/${day}/i-learned$`));
  await expect(editor.locator("p").first()).toHaveText(
    "Octopuses have three hearts.",
  );

  // Both Notebooks were saved on their own.
  await page.reload();
  await expect(editor.locator("p").first()).toHaveText(
    "Octopuses have three hearts.",
  );
  await journalTab.click();
  await expect(editor.locator("p").first()).toHaveText("A quiet Thursday");

  // An unknown Notebook goes home. (Same account: signups are rate limited per IP.)
  await page.goto(`/day/${day}/diary`);
  await expect(page).toHaveURL(new RegExp(`/day/${day}/my-journal$`));
});
