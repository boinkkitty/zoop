// Formatting by keyboard shortcut: the editor behaviour on top of the document schema.
// Marks, bullet and numbered lists keep their TipTap shortcuts (⌘B ⌘I ⌘U ⌘⇧H ⌘⇧8 ⌘⇧7);
// this adds ⌘⇧L, ⌘⇧9, Tab nesting capped at MAX_LIST_DEPTH, and ⌘Enter for checkboxes.
// Mod is ⌘ on a Mac and Ctrl elsewhere. Markdown input and paste rules are off, and
// pasting goes through the schema, so only the v1 set survives (link text without the link).
import { type AnyExtension, type Editor, Extension } from "@tiptap/core";
import { ListKeymap } from "@tiptap/extension-list";
import { Fragment, type Node as PMNode } from "@tiptap/pm/model";
import { liftListItem, sinkListItem } from "@tiptap/pm/schema-list";
import { Plugin, PluginKey, type Transaction } from "@tiptap/pm/state";
import { isList, listDepth, MAX_LIST_DEPTH } from "./schema";

const listItemTypes = ["listItem", "taskItem"];

// The innermost list or checklist item around the cursor, if any.
function currentItem(editor: Editor) {
  const { $from } = editor.state.selection;
  for (let d = $from.depth; d > 0; d--) {
    const node = $from.node(d);
    if (listItemTypes.includes(node.type.name))
      return { node, pos: $from.before(d) };
  }
  return null;
}

// Tab nests the item under the one above, unless that would take lists past MAX_LIST_DEPTH.
// Inside a list Tab is always taken, so it never moves focus out of the Page.
function nest(editor: Editor) {
  const item = currentItem(editor);
  if (!item) return false;
  const { state, view } = editor;
  let tr: Transaction | undefined;
  sinkListItem(item.node.type)(state, (t) => {
    tr = t;
  });
  if (tr && listDepth(tr.doc) <= MAX_LIST_DEPTH)
    view.dispatch(tr.scrollIntoView());
  return true;
}

function unnest(editor: Editor) {
  const item = currentItem(editor);
  if (!item) return false;
  liftListItem(item.node.type)(editor.state, editor.view.dispatch);
  return true;
}

// ⌘Enter ticks or unticks the checklist item with the cursor.
function toggleCheckbox(editor: Editor) {
  const { $from } = editor.state.selection;
  for (let d = $from.depth; d > 0; d--) {
    const node = $from.node(d);
    if (node.type.name === "taskItem") {
      editor.view.dispatch(
        editor.state.tr.setNodeMarkup($from.before(d), undefined, {
          ...node.attrs,
          checked: !node.attrs.checked,
        }),
      );
      return true;
    }
  }
  return false;
}

// flattenDeepLists replaces any list nested past MAX_LIST_DEPTH with its items' contents,
// so a pasted (or joined) list can never break the nesting rule.
export function flattenDeepLists(tr: Transaction): boolean {
  let changed = false;
  for (;;) {
    const deep = findTooDeepList(tr.doc);
    if (!deep) return changed;
    const blocks: PMNode[] = [];
    deep.node.forEach((item) => {
      item.forEach((block) => {
        blocks.push(block);
      });
    });
    tr.replaceWith(
      deep.pos,
      deep.pos + deep.node.nodeSize,
      Fragment.from(blocks),
    );
    changed = true;
  }
}

function findTooDeepList(doc: PMNode) {
  let found: { node: PMNode; pos: number } | null = null;
  doc.descendants((node, pos) => {
    if (found) return false;
    if (isList(node) && listsAround(doc, pos) + 1 > MAX_LIST_DEPTH) {
      found = { node, pos };
      return false;
    }
    return true;
  });
  return found as { node: PMNode; pos: number } | null;
}

function listsAround(doc: PMNode, pos: number) {
  const $pos = doc.resolve(pos);
  let n = 0;
  for (let d = $pos.depth; d > 0; d--) if (isList($pos.node(d))) n++;
  return n;
}

const Formatting = Extension.create({
  name: "zoopFormatting",
  priority: 1000, // ahead of the list items' own Tab and Shift-Tab

  addKeyboardShortcuts() {
    return {
      "Mod-Shift-l": () => this.editor.commands.toggleTaskList(),
      "Mod-Shift-9": () => this.editor.commands.toggleBlockquote(),
      Tab: () => nest(this.editor),
      "Shift-Tab": () => unnest(this.editor),
      "Mod-Enter": () => toggleCheckbox(this.editor),
    };
  },

  addProseMirrorPlugins() {
    return [
      new Plugin({
        key: new PluginKey("zoopListDepth"),
        appendTransaction: (trs, _old, state) => {
          if (!trs.some((tr) => tr.docChanged)) return null;
          if (listDepth(state.doc) <= MAX_LIST_DEPTH) return null;
          const tr = state.tr;
          return flattenDeepLists(tr) ? tr : null;
        },
      }),
    ];
  },
});

// The editor options every Page editor shares (PageEditor adds placeholders and history).
export const editorOptions = {
  extensions: [Formatting, ListKeymap] as AnyExtension[],
  // Typed Markdown stays as literal characters, typed or pasted.
  enableInputRules: false,
  enablePasteRules: false,
};
