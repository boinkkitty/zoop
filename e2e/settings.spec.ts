import { expect, type Page, test } from "./fixtures";

test.use({ timezoneId: "Europe/London" });

const password = "correct horse battery staple";
const journal = /\/day\/\d{4}-\d{2}-\d{2}\/my-journal$/;
const newEmail = () =>
  `e2e+${Date.now()}${Math.floor(Math.random() * 1e6)}@example.com`;

async function signUp(page: Page) {
  const email = newEmail();
  await page.goto("/signup");
  await page.getByLabel("Email").fill(email);
  await page.getByLabel("Password").fill(password);
  await page.getByRole("button", { name: "Create account" }).click();
  await expect(page).toHaveURL(journal);
  await expect(page.getByRole("textbox", { name: "Page" })).toBeVisible();
  return email;
}

const settings = (page: Page) => page.getByRole("dialog", { name: "Settings" });

test("Settings opens from the sidebar or ⌘, and lists the account's rows in order", async ({
  page,
}) => {
  const email = await signUp(page);

  await page.getByRole("button", { name: "Settings" }).click();
  await expect(settings(page)).toBeVisible();
  await expect(settings(page).getByRole("heading", { level: 3 })).toHaveText([
    "Keyboard shortcuts",
    "Email",
    "Sign-in",
    "Timezone",
    "Privacy",
    "Sign out",
    "Deactivate account",
  ]);
  await expect(settings(page)).toContainText(email);
  await expect(settings(page)).not.toContainText(/free|premium/i);
  await page.keyboard.press("Escape");
  await expect(settings(page)).toHaveCount(0);

  await page.keyboard.press("ControlOrMeta+,");
  await expect(settings(page)).toBeVisible();
});

test("the shortcut row swaps Settings for the shortcut sheet", async ({
  page,
}) => {
  await signUp(page);
  await page.keyboard.press("ControlOrMeta+,");
  await settings(page)
    .getByRole("button", { name: "Show keyboard shortcuts" })
    .click();

  await expect(
    page.getByRole("dialog", { name: "Keyboard shortcuts" }),
  ).toBeVisible();
  await expect(settings(page)).toHaveCount(0);
});

test("the timezone switches from tomorrow and the switch can be cancelled", async ({
  page,
}) => {
  await signUp(page);
  await page.keyboard.press("ControlOrMeta+,");

  // New York is behind London, so its switch is always still pending tomorrow.
  await settings(page)
    .getByRole("combobox", { name: "Timezone" })
    .fill("new york");
  await settings(page)
    .getByRole("option", { name: /New York/ })
    .click();
  await expect(settings(page)).toContainText(
    /Switching to New York on \w{3} \d{1,2} \w{3}/,
  );

  await settings(page).getByRole("button", { name: "Cancel" }).click();
  await expect(settings(page)).not.toContainText("Switching to New York");
  const me = await (await page.request.get("/api/v1/me")).json();
  expect(me.pendingTimezone).toBeNull();
});

test("Sign out signs this device out without asking", async ({ page }) => {
  await signUp(page);
  await page.keyboard.press("ControlOrMeta+,");
  await settings(page).getByRole("button", { name: "Sign out" }).click();

  await expect(page).toHaveURL(/\/signin$/);
  await page.goto("/");
  await expect(page).toHaveURL(/\/signin$/);
});

test("deactivating needs the email typed, then the account can't be signed into", async ({
  page,
}) => {
  const email = await signUp(page);
  await page.keyboard.press("ControlOrMeta+,");
  await settings(page)
    .getByRole("button", { name: "Deactivate account" })
    .click();

  const confirm = page.getByRole("dialog", { name: "Deactivate account" });
  await expect(confirm).toContainText("erased within a year");
  await expect(confirm).toContainText("90 days");
  const button = confirm.getByRole("button", { name: "Deactivate" });
  await expect(button).toBeDisabled();
  await confirm.getByLabel("Type your email to confirm").fill(email);
  await button.click();

  await expect(page).toHaveURL(/\/signin$/);
  await page.getByLabel("Email").fill(email);
  await page.getByLabel("Password").fill(password);
  await page.getByRole("button", { name: "Sign in" }).click();
  await expect(page.getByRole("alert")).toHaveText(
    "This account was deactivated.",
  );
});
