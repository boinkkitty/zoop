import {
  type APIRequestContext,
  expect,
  type Page,
  test,
} from "@playwright/test";

// Its own client address (the app trusts one proxy hop), so these signups don't use up
// the other specs' signups-per-IP allowance.
const octet = () => Math.floor(Math.random() * 256);
test.use({
  timezoneId: "Asia/Tokyo",
  extraHTTPHeaders: {
    "X-Forwarded-For": `10.${octet()}.${octet()}.${octet()}`,
  },
});

const password = "correct horse battery staple";
const journal = /\/day\/\d{4}-\d{2}-\d{2}\/my-journal$/;
const newEmail = () =>
  `e2e+${Date.now()}${Math.floor(Math.random() * 1e6)}@example.com`;

async function signUp(page: Page, email: string) {
  await page.goto("/signup");
  await page.getByLabel("Email").fill(email);
  await page.getByLabel("Password").fill(password);
  await page.getByRole("button", { name: "Create account" }).click();
  await expect(page).toHaveURL(journal);
}

// The newest emailed link to `to`, read from the test-only outbox (-tags testclock builds).
async function emailedLink(request: APIRequestContext, to: string) {
  const res = await request.get(
    `/api/test/outbox?to=${encodeURIComponent(to)}`,
  );
  expect(res.ok()).toBe(true);
  const [newest] = (await res.json()) as { Text: string }[];
  expect(newest, `an email to ${to}`).toBeDefined();
  const link = new URL(/https?:\/\/\S+/.exec(newest?.Text ?? "")?.[0] ?? "");
  return `${link.pathname}${link.search}`; // on whatever host the test reaches the app at
}

test("the verify email Notice offers Resend and stays dismissed for the session", async ({
  page,
}) => {
  const email = newEmail();
  await signUp(page, email);
  const notices = page.getByRole("region", { name: "Notices" });
  await expect(notices).toContainText(`Check your inbox to verify ${email}.`);

  await notices.getByRole("button", { name: "Resend" }).click();
  await expect(notices).toContainText("Sent again");
  await expect(notices.getByRole("button", { name: "Resend" })).toHaveCount(0);

  await notices.getByRole("button", { name: "Dismiss" }).click();
  await expect(notices).toHaveCount(0);
  await page.reload();
  await expect(page.getByRole("textbox", { name: "Page" })).toBeVisible();
  await expect(
    page.getByText(`Check your inbox to verify ${email}.`),
  ).toHaveCount(0);
});

test("the emailed link verifies the email, even after a link-scanner fetched it", async ({
  page,
}) => {
  const email = newEmail();
  await signUp(page, email);
  const link = await emailedLink(page.request, email);

  expect(link).toMatch(/^\/verify\?token=/);
  expect((await page.request.get(link)).ok()).toBe(true); // a scanner's GET
  await page.goto(link);
  await expect(
    page.getByRole("heading", { name: "Email verified" }),
  ).toBeVisible();

  await page.getByRole("link", { name: "Go to your journal" }).click();
  await expect(page).toHaveURL(journal);
  await expect(page.getByRole("textbox", { name: "Page" })).toBeVisible();
  await expect(
    page.getByText(`Check your inbox to verify ${email}.`),
  ).toHaveCount(0);
});

test("a forgotten password is reset from the sign-in screen", async ({
  page,
}) => {
  const email = newEmail();
  await signUp(page, email);
  await page.getByRole("button", { name: "Sign out" }).click();
  await expect(page).toHaveURL(/\/signin$/);

  await page.getByRole("link", { name: "Forgot password?" }).click();
  await expect(page).toHaveURL(/\/forgot$/);
  await page.getByLabel("Email").fill(email);
  await page.getByRole("button", { name: "Send reset link" }).click();
  await expect(page.getByRole("status")).toContainText(
    `If ${email} has an account`,
  );

  const link = await emailedLink(page.request, email);
  expect(link).toMatch(/^\/reset\?token=/);
  await page.goto(link);
  await page.getByLabel("New password").fill("a whole new passphrase");
  await page.getByRole("button", { name: "Set password" }).click();
  await expect(page.getByRole("status")).toContainText("Your password is set");

  await page.getByRole("link", { name: "Sign in" }).click();
  await page.getByLabel("Email").fill(email);
  await page.getByLabel("Password").fill("a whole new passphrase");
  await page.getByRole("button", { name: "Sign in" }).click();
  await expect(page).toHaveURL(journal);
});
