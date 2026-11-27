import { expect, type Page, test } from "./fixtures";

// In a narrow window the sidebar slides over the sheet from a menu button.
test.use({ viewport: { width: 700, height: 820 } });

const PASSWORD = "correct horse battery staple";

async function signUp(page: Page) {
  const email = `e2e+${Date.now()}${Math.floor(Math.random() * 1e6)}@example.com`;
  await page.goto("/signup");
  await page.getByLabel("Email").fill(email);
  await page.getByLabel("Password").fill(PASSWORD);
  await page.getByRole("button", { name: "Create account" }).click();
  await expect(page).toHaveURL(/\/day\/[\d-]+\/my-journal$/);
}

const drawer = (page: Page) => page.getByRole("dialog", { name: "Sidebar" });
const menu = (page: Page) => page.getByRole("button", { name: "Open sidebar" });

test("the drawer opens from the menu button and closes on choosing a Day", async ({
  page,
}) => {
  await signUp(page);
  const days = page.getByRole("navigation", { name: "Days" });
  await expect(days).toBeHidden();

  await menu(page).click();
  await expect(drawer(page)).toBeVisible();
  // Today's Island, the only Day a new signup has.
  await drawer(page)
    .getByRole("navigation", { name: "Days" })
    .getByRole("button")
    .first()
    .click();
  await expect(drawer(page)).toBeHidden();
  await expect(days).toBeHidden();
});

test("Esc closes the drawer", async ({ page }) => {
  await signUp(page);
  await menu(page).click();
  await expect(drawer(page)).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(drawer(page)).toBeHidden();
});

test("⌘K opens the drawer with search focused, and opening a result closes it", async ({
  page,
}) => {
  await signUp(page);
  const saved = page.waitForResponse(
    (res) => res.request().method() === "PUT" && res.ok(),
  );
  await page.getByRole("textbox", { name: "Page" }).click();
  await page.keyboard.type("Narrow window notes");
  await saved;

  await page.keyboard.press("ControlOrMeta+k");
  await expect(drawer(page)).toBeVisible();
  const box = drawer(page).getByRole("searchbox", { name: "Search" });
  await expect(box).toBeFocused();
  await page.keyboard.type("narrow");
  await expect(
    drawer(page)
      .getByRole("list", { name: "Search results" })
      .getByRole("button")
      .first(),
  ).toContainText("Narrow window notes");
  await page.keyboard.press("Enter");
  await expect(drawer(page)).toBeHidden();
});
