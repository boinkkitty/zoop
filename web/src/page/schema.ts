// The Page document schema: TipTap v3 with an explicit extension list, never StarterKit.
// internal/pages/doc.go validates the same grammar on the server; the shared
// fixtures in testdata/page-docs keep the two in step.
import { type AnyExtension, getSchema } from "@tiptap/core";
import Document from "@tiptap/extension-document";
import Paragraph from "@tiptap/extension-paragraph";
import Text from "@tiptap/extension-text";
import type { Node as PMNode, Schema } from "@tiptap/pm/model";
import type { Doc } from "../api";
import { isNotebook, NOTEBOOKS, type Notebook } from "./notebooks";

// Bump together with pages.SchemaVersion and testdata/page-docs/schema-version.json.
export const SCHEMA_VERSION = 1;

export type { Doc } from "../api";

export const EMPTY_DOC: Doc = { type: "doc", content: [{ type: "paragraph" }] };

// Each Notebook's grammar, as on the server (internal/pages/doc.go). My Journal starts with
// the Title, a plain paragraph whose bold is CSS. I Learned has no Title.
const grammars: Record<Notebook, string> = {
  "my-journal": "paragraph block*",
  "i-learned": "block+",
};

// The nodes and marks both Notebooks share.
const contentExtensions: AnyExtension[] = [Paragraph, Text];

// documentExtensions are one Notebook's document extensions only: editor behaviour
// (placeholders, history) is added by the editor.
export const documentExtensions = (notebook: Notebook): AnyExtension[] => [
  Document.extend({ content: grammars[notebook] }),
  ...contentExtensions,
];

const schemas = Object.fromEntries(
  NOTEBOOKS.map(({ slug }) => [slug, getSchema(documentExtensions(slug))]),
) as Record<Notebook, Schema>;

// parseDoc returns the ProseMirror node for json if it's a valid Page in notebook, or null.
// It also refuses anything ProseMirror would silently drop, so what's sent is what's stored.
export function parseDoc(notebook: string, json: unknown): PMNode | null {
  if (!isNotebook(notebook)) return null;
  const pageSchema = schemas[notebook];
  try {
    const node = pageSchema.nodeFromJSON(json);
    if (node.type !== pageSchema.topNodeType) return null;
    node.check();
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
