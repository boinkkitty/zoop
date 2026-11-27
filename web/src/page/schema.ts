// The Page document schema: TipTap v3 with an explicit extension list, never StarterKit.
// internal/pages/schema.go validates the same grammar on the server; the shared
// fixtures in testdata/page-docs keep the two in step.
import {
  type AnyExtension,
  getSchema,
  type KeyboardShortcutCommand,
  mergeAttributes,
} from "@tiptap/core";
import Blockquote from "@tiptap/extension-blockquote";
import Bold from "@tiptap/extension-bold";
import Document from "@tiptap/extension-document";
import Highlight from "@tiptap/extension-highlight";
import Italic from "@tiptap/extension-italic";
import {
  BulletList,
  ListItem,
  OrderedList,
  TaskItem,
  TaskList,
} from "@tiptap/extension-list";
import Paragraph from "@tiptap/extension-paragraph";
import Text from "@tiptap/extension-text";
import Underline from "@tiptap/extension-underline";
import type { Node as PMNode, Schema } from "@tiptap/pm/model";
import type { Doc } from "../api";
import { isNotebook, NOTEBOOKS, type Notebook } from "./notebooks";

// Bump together with pages.SchemaVersion and testdata/page-docs/schema-version.json.
// 2: formatting (marks, lists, checklists, quotes).
export const SCHEMA_VERSION = 2;

// Lists nest at most this deep: a list inside at most two others (pages.MaxListDepth).
export const MAX_LIST_DEPTH = 3;

export type { Doc } from "../api";

export const EMPTY_DOC: Doc = { type: "doc", content: [{ type: "paragraph" }] };

// Each Notebook's grammar, as on the server (docRules in internal/pages/schema.go).
// My Journal starts with the Title, a plain paragraph whose bold is CSS, so it can
// never become a list or a quote. I Learned has no Title.
const grammars: Record<Notebook, string> = {
  "my-journal": "paragraph block*",
  "i-learned": "block+",
};

// Numbered lists always count from 1: no start or type attrs, and no plugin that
// turns pasted "1. …" text into a list (pasted text stays literal).
const PageOrderedList = OrderedList.extend({
  addAttributes: () => ({}),
  renderHTML({ HTMLAttributes }) {
    return [
      "ol",
      mergeAttributes(this.options.HTMLAttributes, HTMLAttributes),
      0,
    ];
  },
  addProseMirrorPlugins: () => [],
});

// Upstream binds the checklist to ⌘⇧9 and the quote to ⌘⇧B; Zoop binds them in
// formatting.ts (⌘⇧L and ⌘⇧9), so those defaults are dropped here.
const PageTaskList = TaskList.extend({ addKeyboardShortcuts: () => ({}) });
const PageBlockquote = Blockquote.extend({
  addKeyboardShortcuts() {
    const keys: Record<string, KeyboardShortcutCommand> = {};
    const Backspace = this.parent?.().Backspace;
    if (Backspace) keys.Backspace = Backspace;
    return keys;
  },
});

// Checklist items nest like list items, and checked is always true or false.
const PageTaskItem = TaskItem.extend({
  addAttributes: () => ({
    checked: {
      default: false,
      validate: "boolean",
      keepOnSplit: false,
      parseHTML: (element) => {
        const checked = element.getAttribute("data-checked");
        return checked === "" || checked === "true";
      },
      renderHTML: (attributes) => ({ "data-checked": attributes.checked }),
    },
  }),
}).configure({ nested: true });

// The nodes and marks both Notebooks share: the v1 set (with their own Enter, Backspace
// and mark shortcuts). Zoop's other shortcuts and paste rules are added by formatting.ts.
const contentExtensions: AnyExtension[] = [
  Paragraph,
  Text,
  Bold,
  Italic,
  Underline,
  Highlight,
  BulletList,
  PageOrderedList,
  ListItem,
  PageTaskList,
  PageTaskItem,
  PageBlockquote,
];

// documentExtensions are one Notebook's document extensions only: editor behaviour
// (placeholders, history, formatting.ts) is added by the editor.
export const documentExtensions = (notebook: Notebook): AnyExtension[] => [
  Document.extend({ content: grammars[notebook] }),
  ...contentExtensions,
];

const schemas = Object.fromEntries(
  NOTEBOOKS.map(({ slug }) => [slug, getSchema(documentExtensions(slug))]),
) as Record<Notebook, Schema>;

// listDepth is how deeply lists nest inside node: 0 for none, 1 for a list with no list inside.
export function listDepth(node: PMNode): number {
  let deepest = 0;
  node.forEach((child) => {
    deepest = Math.max(deepest, listDepth(child));
  });
  return isList(node) ? deepest + 1 : deepest;
}

export const isList = (node: PMNode) => node.type.isInGroup("list");

// parseDoc returns the ProseMirror node for json if it's a valid Page in notebook, or null.
// It also refuses anything ProseMirror would silently drop, so what's sent is what's stored.
export function parseDoc(notebook: string, json: unknown): PMNode | null {
  if (!isNotebook(notebook)) return null;
  const pageSchema = schemas[notebook];
  try {
    const node = pageSchema.nodeFromJSON(json);
    if (node.type !== pageSchema.topNodeType) return null;
    node.check();
    if (listDepth(node) > MAX_LIST_DEPTH) return null;
    return sameJSON(node.toJSON(), json) ? node : null;
  } catch {
    return null;
  }
}

// sameDoc compares two docs by content, ignoring key order (Postgres's JSONB reorders keys).
export const sameDoc = (a: Doc, b: Doc) => sameJSON(a, b);

function sameJSON(a: unknown, b: unknown): boolean {
  return JSON.stringify(sorted(a)) === JSON.stringify(sorted(b));
}

function sorted(v: unknown): unknown {
  if (Array.isArray(v)) return v.map(sorted);
  if (v && typeof v === "object") {
    return Object.fromEntries(
      Object.entries(v)
        .sort(([a], [b]) => a.localeCompare(b))
        .map(([k, x]) => [k, sorted(x)]),
    );
  }
  return v;
}
