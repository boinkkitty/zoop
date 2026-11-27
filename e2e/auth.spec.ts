import { expect, type Page, test } from "@playwright/test";

const timezone = "Asia/Tokyo";
test.use({ timezoneId: timezone });

// The lazy chunk holding the signed-in app (and, later, the editor).
const appChunk = /\/assets\/JournalRoute-[^/]*\.js$/;

function todayIn(zone: string): string {
  // en-CA formats as YYYY-MM-DD.
  return new Intl.DateTimeFormat("en-CA", { timeZone: zone }).format(
    new Date(),
  );
}

function recordScripts(page: Page): string[] {
  const urls: string[] = [];
  page.on("request", (req) => {
    if (req.resourceType() === "script") urls.push(req.url());
  });
  return urls;
}

test("sign up, land in the app, sign out, sign in again", async ({ page }) => {
  const scripts = recordScripts(page);
  const email = `e2e+${Date.now()}${Math.floor(Math.random() * 1e6)}@example.com`;
  const password = "correct horse battery staple";
  const journal = new RegExp(`/day/${todayIn(timezone)}/my-journal$`);

  await page.goto("/");
  await expect(page).toHaveURL(/\/signin$/);
  await page.getByRole("link", { name: "Create an account" }).click();
  await expect(page).toHaveURL(/\/signup$/);

  await page.getByLabel("Email").fill(email);
  await page.getByLabel("Password").fill(password);
  expect(scripts.filter((u) => appChunk.test(u))).toEqual([]);
  await page.getByRole("button", { name: "Create account" }).click();

  await expect(page).toHaveURL(journal);
  await expect(page.getByText(email)).toBeVisible();
  expect(scripts.some((u) => appChunk.test(u))).toBe(true);

  await page.getByRole("button", { name: "Sign out" }).click();
  await expect(page).toHaveURL(/\/signin$/);
  await page.goto("/");
  await expect(page).toHaveURL(/\/signin$/);

  await page.getByLabel("Email").fill(email.toUpperCase());
  await page.getByLabel("Password").fill("wrong password!");
  await page.getByRole("button", { name: "Sign in" }).click();
  await expect(page.getByRole("alert")).toHaveText("Wrong email or password.");

  await page.getByLabel("Password").fill(password);
  await page.getByRole("button", { name: "Sign in" }).click();
  await expect(page).toHaveURL(journal);
  await expect(page.getByText(email)).toBeVisible();
});

test("sign-in and signup offer Google, carrying the browser's timezone", async ({
  page,
}) => {
  const start = `/api/v1/auth/google/start?timezone=${encodeURIComponent(timezone)}`;
  for (const screen of ["/signin", "/signup"]) {
    await page.goto(screen);
    await expect(
      page.getByRole("link", { name: "Continue with Google" }),
    ).toHaveAttribute("href", start);
  }
});

test("a refused Google sign-in is explained on the sign-in screen", async ({
  page,
}) => {
  await page.goto("/signin?error=google_unverified");
  await expect(page.getByRole("alert")).toHaveText(
    "Google hasn't verified the email on that account, so it can't be used to sign in.",
  );
});

test("the auth screens never load the app chunk", async ({ page }) => {
  const scripts = recordScripts(page);
  await page.goto("/signin");
  await expect(page.getByRole("button", { name: "Sign in" })).toBeVisible();
  await page.goto("/signup");
  await expect(
    page.getByRole("button", { name: "Create account" }),
  ).toBeVisible();
  expect(scripts.length).toBeGreaterThan(0);
  expect(scripts.filter((u) => appChunk.test(u))).toEqual([]);
});
