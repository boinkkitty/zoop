// @vitest-environment jsdom
import { Editor, type JSONContent } from "@tiptap/core";
import { afterEach, describe, expect, it } from "vitest";
import { editorOptions } from "./formatting";
import { documentExtensions, parseDoc } from "./schema";

// The editor exactly as PageEditor configures it, minus placeholders and history.
let editor: Editor;
function open(content: JSONContent) {
  editor = new Editor({
    ...editorOptions,
    element: document.body.appendChild(document.createElement("div")),
    extensions: [...documentExtensions, ...editorOptions.extensions],
    content,
  });
  return editor;
}
afterEach(() => {
  editor?.destroy();
  document.body.replaceChildren();
});

const t = (text: string, ...marks: string[]): JSONContent =>
  marks.length
    ? { type: "text", text, marks: marks.map((type) => ({ type })) }
    : { type: "text", text };
const p = (...content: JSONContent[]): JSONContent =>
  content.length ? { type: "paragraph", content } : { type: "paragraph" };
const doc = (...content: JSONContent[]): JSONContent => ({
  type: "doc",
  content,
});
const li = (...content: JSONContent[]) => ({ type: "listItem", content });
const ti = (checked: boolean, ...content: JSONContent[]) => ({
  type: "taskItem",
  attrs: { checked },
  content,
});
const ul = (...content: JSONContent[]) => ({ type: "bulletList", content });
const ol = (...content: JSONContent[]) => ({ type: "orderedList", content });
const tl = (...content: JSONContent[]) => ({ type: "taskList", content });
const bq = (...content: JSONContent[]) => ({ type: "blockquote", content });

// press sends a keydown the way a browser does. jsdom isn't a Mac, so Mod is Ctrl.
function press(combo: string) {
  const parts = combo.split("-");
  const key = parts.pop() ?? "";
  const event = new KeyboardEvent("keydown", {
    key,
    ctrlKey: parts.includes("Mod"),
    shiftKey: parts.includes("Shift"),
    bubbles: true,
    cancelable: true,
  });
  editor.view.dom.dispatchEvent(event);
}

// at puts the cursor inside the first text that contains needle (or selects it all with select).
function at(needle: string, select = false) {
  let found: number | null = null;
  editor.state.doc.descendants((node, pos) => {
    if (found === null && node.isText && node.text?.includes(needle)) {
      found = pos + (node.text?.indexOf(needle) ?? 0);
    }
  });
  if (found === null) throw new Error(`no text ${needle}`);
  const from: number = found;
  editor.commands.setTextSelection(
    select ? { from, to: from + needle.length } : from + 1,
  );
}

// The empty paragraph's position (cursor inside the nth empty textblock).
function atEmpty(n = 0) {
  let seen = 0;
  let pos: number | null = null;
  editor.state.doc.descendants((node, p) => {
    if (pos === null && node.isTextblock && node.childCount === 0) {
      if (seen++ === n) pos = p + 1;
    }
  });
  if (pos === null) throw new Error("no empty textblock");
  editor.commands.setTextSelection(pos);
}

const json = () => editor.getJSON();

describe("mark shortcuts", () => {
  it.each([
    ["Mod-b", "bold"],
    ["Mod-i", "italic"],
    ["Mod-u", "underline"],
    ["Mod-Shift-h", "highlight"],
  ])("%s toggles %s", (combo, mark) => {
    open(doc(p(t("Title")), p(t("some words"))));
    at("words", true);
    press(combo);
    expect(json()).toEqual(doc(p(t("Title")), p(t("some "), t("words", mark))));
    press(combo);
    expect(json()).toEqual(doc(p(t("Title")), p(t("some words"))));
  });
});

describe("block shortcuts", () => {
  it.each([
    ["Mod-Shift-8", ul(li(p(t("milk"))))],
    ["Mod-Shift-7", ol(li(p(t("milk"))))],
    ["Mod-Shift-l", tl(ti(false, p(t("milk"))))],
    ["Mod-Shift-9", bq(p(t("milk")))],
  ])(
    "%s toggles the block, and again turns it back into a paragraph",
    (combo, block) => {
      open(doc(p(t("Title")), p(t("milk"))));
      at("milk");
      press(combo);
      expect(json()).toEqual(doc(p(t("Title")), block));
      press(combo);
      expect(json()).toEqual(doc(p(t("Title")), p(t("milk"))));
    },
  );

  it.each(["Mod-Shift-8", "Mod-Shift-7", "Mod-Shift-l", "Mod-Shift-9"])(
    "%s leaves the Title a plain line",
    (combo) => {
      const before = doc(p(t("Title")), p(t("milk")));
      open(before);
      at("Title");
      press(combo);
      expect(json()).toEqual(before);
      // Nor when the selection spans the Title and the body.
      editor.commands.setTextSelection({ from: 2, to: 10 });
      press(combo);
      expect(json()).toEqual(before);
    },
  );

  it("switches one kind of list to another", () => {
    open(doc(p(t("Title")), ul(li(p(t("milk"))))));
    at("milk");
    press("Mod-Shift-7");
    expect(json()).toEqual(doc(p(t("Title")), ol(li(p(t("milk"))))));
  });

  it("Enter on an empty list item ends the list", () => {
    open(doc(p(t("Title")), ul(li(p(t("milk"))), li(p()))));
    atEmpty();
    press("Enter");
    expect(json()).toEqual(doc(p(t("Title")), ul(li(p(t("milk")))), p()));
  });

  it("Enter on an empty checklist item ends the checklist", () => {
    open(doc(p(t("Title")), tl(ti(true, p(t("milk"))), ti(false, p()))));
    atEmpty();
    press("Enter");
    expect(json()).toEqual(doc(p(t("Title")), tl(ti(true, p(t("milk")))), p()));
  });
});

describe("nesting", () => {
  const flat = doc(
    p(t("Title")),
    ul(li(p(t("one"))), li(p(t("two"))), li(p(t("three"))), li(p(t("four")))),
  );

  it("Tab nests a list item and Shift-Tab brings it back", () => {
    open(flat);
    at("two");
    press("Tab");
    expect(json()).toEqual(
      doc(
        p(t("Title")),
        ul(
          li(p(t("one")), ul(li(p(t("two"))))),
          li(p(t("three"))),
          li(p(t("four"))),
        ),
      ),
    );
    press("Shift-Tab");
    expect(json()).toEqual(flat);
  });

  it("Tab nests at most 3 levels", () => {
    open(flat);
    at("two");
    press("Tab");
    at("three");
    press("Tab");
    press("Tab");
    at("four");
    press("Tab");
    press("Tab");
    press("Tab");
    press("Tab");
    const three = doc(
      p(t("Title")),
      ul(
        li(
          p(t("one")),
          ul(li(p(t("two")), ul(li(p(t("three"))), li(p(t("four")))))),
        ),
      ),
    );
    expect(json()).toEqual(three);
  });

  it("Tab nests checklist items too", () => {
    open(doc(p(t("Title")), tl(ti(false, p(t("one"))), ti(true, p(t("two"))))));
    at("two");
    press("Tab");
    expect(json()).toEqual(
      doc(p(t("Title")), tl(ti(false, p(t("one")), tl(ti(true, p(t("two"))))))),
    );
  });
});

describe("checkboxes", () => {
  it("Mod-Enter toggles the checkbox of the item with the cursor", () => {
    open(
      doc(p(t("Title")), tl(ti(false, p(t("ferns"))), ti(false, p(t("mum"))))),
    );
    at("mum");
    press("Mod-Enter");
    expect(json()).toEqual(
      doc(p(t("Title")), tl(ti(false, p(t("ferns"))), ti(true, p(t("mum"))))),
    );
    press("Mod-Enter");
    expect(json()).toEqual(
      doc(p(t("Title")), tl(ti(false, p(t("ferns"))), ti(false, p(t("mum"))))),
    );
  });

  it("a click on the checkbox toggles it", () => {
    open(doc(p(t("Title")), tl(ti(false, p(t("ferns"))))));
    const box = editor.view.dom.querySelector<HTMLInputElement>(
      'input[type="checkbox"]',
    );
    box?.click();
    expect(json()).toEqual(doc(p(t("Title")), tl(ti(true, p(t("ferns"))))));
  });
});

describe("paste", () => {
  const paste = (html: string) =>
    editor.view.pasteHTML(html, new Event("paste") as ClipboardEvent);
  const pasteText = (text: string) =>
    editor.view.pasteText(text, new Event("paste") as ClipboardEvent);

  it("keeps only the v1 formatting: link text stays, links and foreign styles go", () => {
    open(doc(p(t("Title")), p()));
    atEmpty();
    paste(
      '<p>See <a href="https://example.com">the site</a>, <span style="color:red;font-family:Comic Sans">red</span>, ' +
        "<strong>bold</strong>, <em>it</em>, <u>under</u>, <mark>marked</mark>, <s>struck</s>, <code>code</code>.</p>",
    );
    expect(json()).toEqual(
      doc(
        p(t("Title")),
        p(
          t("See the site, red, "),
          t("bold", "bold"),
          t(", "),
          t("it", "italic"),
          t(", "),
          t("under", "underline"),
          t(", "),
          t("marked", "highlight"),
          t(", struck, code."),
        ),
      ),
    );
    expect(parseDoc(json())).not.toBeNull();
  });

  it("turns headings and other blocks into paragraphs", () => {
    open(doc(p(t("Title")), p()));
    atEmpty();
    paste("<h1>Big</h1><pre>code</pre><p>after</p>");
    const got = json();
    expect(parseDoc(got)).not.toBeNull();
    expect(
      editor.state.doc.textBetween(0, editor.state.doc.content.size, "\n"),
    ).toBe("Title\nBig\ncode\nafter");
  });

  it("flattens pasted lists deeper than 3 levels", () => {
    open(doc(p(t("Title")), p()));
    atEmpty();
    paste(
      "<ul><li>1<ul><li>2<ul><li>3<ul><li>4<ul><li>5</li></ul></li></ul></li></ul></li></ul></li></ul>",
    );
    expect(json()).toEqual(
      doc(
        p(t("Title")),
        ul(
          li(
            p(t("1")),
            ul(li(p(t("2")), ul(li(p(t("3")), p(t("4")), p(t("5")))))),
          ),
        ),
      ),
    );
    expect(parseDoc(json())).not.toBeNull();
  });

  it("drops an ordered list's start number", () => {
    open(doc(p(t("Title")), p()));
    atEmpty();
    paste('<ol start="4" type="a"><li>d</li></ol>');
    expect(json()).toEqual(doc(p(t("Title")), ol(li(p(t("d"))))));
  });

  it("keeps pasted Markdown and numbered text literal", () => {
    open(doc(p(t("Title")), p()));
    atEmpty();
    pasteText("- milk\n1. eggs\n2. **flour**");
    expect(json()).toEqual(
      doc(p(t("Title")), p(t("- milk")), p(t("1. eggs")), p(t("2. **flour**"))),
    );
  });
});
