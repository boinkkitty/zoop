// Search matches on the opened Page: decorations, not marks, so they're never saved and
// never look like the user's own highlight. The first match is the focused one.
import { Extension } from "@tiptap/core";
import type { Node as PMNode } from "@tiptap/pm/model";
import { Plugin, PluginKey } from "@tiptap/pm/state";
import { Decoration, DecorationSet } from "@tiptap/pm/view";
import { matches } from "./match";

export const searchHighlightKey = new PluginKey<DecorationSet>(
  "searchHighlight",
);

// Each textblock holds only text, so an offset into its textContent is pos + 1 + offset.
function decorate(
  doc: PMNode,
  words: readonly string[],
  cls: string,
  current: string,
) {
  const decos: Decoration[] = [];
  doc.descendants((node, pos) => {
    if (!node.isTextblock) return true;
    for (const { from, to } of matches(node.textContent, words)) {
      decos.push(
        Decoration.inline(pos + 1 + from, pos + 1 + to, {
          class: decos.length === 0 ? `${cls} ${current}` : cls,
          "data-search-match": "",
        }),
      );
    }
    return false;
  });
  return DecorationSet.create(doc, decos);
}

// setSearchWords marks words on the Page; an empty list clears the marks.
export const SearchHighlight = Extension.create<{
  className: string;
  currentClassName: string;
}>({
  name: "searchHighlight",
  addOptions: () => ({ className: "", currentClassName: "" }),
  addProseMirrorPlugins() {
    const { className, currentClassName } = this.options;
    return [
      new Plugin<DecorationSet>({
        key: searchHighlightKey,
        state: {
          init: () => DecorationSet.empty,
          apply(tr, set) {
            const words = tr.getMeta(searchHighlightKey) as
              | string[]
              | undefined;
            if (words)
              return decorate(tr.doc, words, className, currentClassName);
            // Typing clears them: the user has moved on from why the Page matched.
            if (tr.docChanged) return DecorationSet.empty;
            return set;
          },
        },
        props: {
          decorations: (state) => searchHighlightKey.getState(state),
        },
      }),
    ];
  },
});
