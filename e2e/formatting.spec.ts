import { expect, type Locator, type Page, test } from "@playwright/test";

test.use({ timezoneId: "Asia/Tokyo" });

async function signUpToToday(page: Page) {
  const email = `e2e+fmt${Date.now()}${Math.floor(Math.random() * 1e6)}@example.com`;
  await page.goto("/signup");
  await page.getByLabel("Email").fill(email);
  await page.getByLabel("Password").fill("correct horse battery staple");
  await page.getByRole("button", { name: "Create account" }).click();
  await expect(page).toHaveURL(/\/day\/\d{4}-\d{2}-\d{2}\/my-journal$/);
  const editor = page.getByRole("textbox", { name: "Page" });
  await editor.click();
  return editor;
}

// paste fires a paste event the way the browser does, with the clipboard's html and text.
async function paste(editor: Locator, html: string, text: string) {
  await editor.evaluate(
    (el, { html, text }) => {
      const data = new DataTransfer();
      data.setData("text/html", html);
      data.setData("text/plain", text);
      el.dispatchEvent(
        new ClipboardEvent("paste", {
          clipboardData: data,
          bubbles: true,
          cancelable: true,
        }),
      );
    },
    { html, text },
  );
}

test("formatting by keyboard shortcut is saved and survives a reload", async ({
  page,
}) => {
  const editor = await signUpToToday(page);
  await page.keyboard.type("Errands");

  // The Title stays a plain line.
  await page.keyboard.press("ControlOrMeta+Shift+8");
  await page.keyboard.press("ControlOrMeta+Shift+9");
  await expect(editor.locator("> p").first()).toHaveText("Errands");
  await expect(editor.locator("ul, blockquote")).toHaveCount(0);

  await page.keyboard.press("Enter");
  await page.keyboard.type("Buy figs");
  for (let i = 0; i < 4; i++) await page.keyboard.press("Shift+ArrowLeft");
  await page.keyboard.press("ControlOrMeta+b");
  await page.keyboard.press("ControlOrMeta+Shift+h");
  await expect(editor.locator("strong mark, mark strong")).toHaveText("figs");

  // A bulleted list, nested, then back to a paragraph with the same shortcut.
  await page.keyboard.press("ArrowRight"); // to the end of the line
  await page.keyboard.press("Enter");
  await page.keyboard.press("ControlOrMeta+Shift+8");
  await page.keyboard.type("milk");
  await page.keyboard.press("Enter");
  await page.keyboard.type("oat");
  await page.keyboard.press("Tab");
  await expect(editor.locator("ul ul li")).toHaveText("oat");
  await page.keyboard.press("Shift+Tab");
  await page.keyboard.press("ControlOrMeta+Shift+8");
  await expect(editor.locator("> p").last()).toHaveText("oat");

  // Enter on an empty list item ends the list.
  await page.keyboard.press("Enter");
  await page.keyboard.press("ControlOrMeta+Shift+7");
  await page.keyboard.type("one");
  await page.keyboard.press("Enter");
  await page.keyboard.press("Enter");
  await page.keyboard.type("after the list");
  await expect(editor.locator("ol li")).toHaveText(["one"]);
  await expect(editor.locator("> p").last()).toHaveText("after the list");

  // A checklist: ticked by a click and by ⌘Enter.
  await page.keyboard.press("Enter");
  await page.keyboard.press("ControlOrMeta+Shift+l");
  await page.keyboard.type("water the ferns");
  const box = editor.getByRole("checkbox");
  await box.click();
  await expect(box).toBeChecked();
  // The click leaves the cursor where it was, at the end of the item.
  await page.keyboard.press("ControlOrMeta+Enter");
  await expect(box).not.toBeChecked();
  await page.keyboard.press("ControlOrMeta+Enter");
  await expect(box).toBeChecked();

  // A quote, and typed Markdown stays literal.
  await page.keyboard.press("Enter");
  await page.keyboard.press("Enter");
  await page.keyboard.press("ControlOrMeta+Shift+9");
  await page.keyboard.type("- not a list, **not bold**");
  await expect(editor.locator("blockquote")).toHaveText(
    "- not a list, **not bold**",
  );

  // Wait for the save that carries the last words typed.
  await page.waitForResponse(
    (res) =>
      res.request().method() === "PUT" &&
      res.url().includes("/api/v1/pages/") &&
      (res.request().postData() ?? "").includes("**not bold**") &&
      res.ok(),
  );
  await page.reload();

  await expect(editor.locator("> p").first()).toHaveText("Errands");
  await expect(editor.locator("strong mark, mark strong")).toHaveText("figs");
  await expect(editor.getByRole("checkbox")).toBeChecked();
  await expect(editor.locator("blockquote")).toHaveText(
    "- not a list, **not bold**",
  );
});

test("paste keeps only Zoop's formatting, and ⌘⇧V pastes plain text", async ({
  page,
}) => {
  const editor = await signUpToToday(page);
  await page.keyboard.type("Pasted");
  await page.keyboard.press("Enter");

  const html =
    '<p style="font-family:Comic Sans;color:red"><strong>Bold</strong> and <a href="https://example.com">a link</a></p>';
  await paste(editor, html, "Bold and a link");
  await expect(editor.locator("strong")).toHaveText("Bold");
  await expect(editor.locator("a")).toHaveCount(0);
  await expect(editor.locator("[style]")).toHaveCount(0);
  await expect(editor).toContainText("Bold and a link");

  await page.keyboard.press("Enter");
  await page.keyboard.down("Shift");
  await paste(editor, "<p><em>Plain</em> please</p>", "Plain please");
  await page.keyboard.up("Shift");
  await expect(editor.locator("> p").last()).toHaveText("Plain please");
  await expect(editor.locator("em")).toHaveCount(0);
});

test("⌘/ opens the shortcut sheet", async ({ page }) => {
  await signUpToToday(page);
  await page.keyboard.press("ControlOrMeta+/");
  const sheet = page.getByRole("dialog", { name: "Keyboard shortcuts" });
  await expect(sheet).toBeVisible();
  await expect(sheet.getByText("Checklist")).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(sheet).toBeHidden();
});
