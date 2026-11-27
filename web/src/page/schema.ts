// The Page document schema: TipTap v3 with an explicit extension list, never StarterKit.
// internal/pages/doc.go validates the same grammar on the server; the shared
// fixtures in testdata/page-docs keep the two in step.
import { type AnyExtension, getSchema } from "@tiptap/core";
import Document from "@tiptap/extension-document";
import Paragraph from "@tiptap/extension-paragraph";
import Text from "@tiptap/extension-text";
import type { Node as PMNode } from "@tiptap/pm/model";
import type { Doc } from "../api";

// Bump together with pages.SchemaVersion and testdata/page-docs/schema-version.json.
export const SCHEMA_VERSION = 1;

export type { Doc } from "../api";

export const EMPTY_DOC: Doc = { type: "doc", content: [{ type: "paragraph" }] };

// My Journal's grammar: the Title paragraph, then any blocks. The Title's bold is CSS.
const PageDocument = Document.extend({ content: "paragraph block*" });

// The document extensions only: editor behaviour (placeholders, history) is added by the editor.
export const documentExtensions: AnyExtension[] = [
  PageDocument,
  Paragraph,
  Text,
];

export const pageSchema = getSchema(documentExtensions);

// parseDoc returns the ProseMirror node for json if it's a valid Page, or null.
// It also refuses anything ProseMirror would silently drop, so what's sent is what's stored.
export function parseDoc(json: unknown): PMNode | null {
  try {
    const node = pageSchema.nodeFromJSON(json);
    if (node.type !== pageSchema.topNodeType) return null;
    node.check();
    return sameJSON(node.toJSON(), json) ? node : null;
  } catch {
    return null;
  }
}

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
