import { useQueryClient } from "@tanstack/react-query";
import { Placeholder, UndoRedo } from "@tiptap/extensions";
import { EditorContent, useEditor } from "@tiptap/react";
import { useEffect, useState } from "react";
import type { Page } from "../api";
import styles from "./PageEditor.module.css";
import { pageQuery } from "./queries";
import { createPageSaver } from "./saver";
import { type Doc, documentExtensions } from "./schema";

type Words = { title: string; body: string };
const myJournal: Words = { title: "Title", body: "Write about your day…" };
const placeholders: Record<string, Words> = {
  "my-journal": myJournal,
  "i-learned": { title: "What did you learn today?", body: "" },
};

// The editor for one Page. Mount it with a key per Page so each gets its own saver.
export function PageEditor({ page }: { page: Page }) {
  const queryClient = useQueryClient();
  const [saver] = useState(() =>
    createPageSaver({
      day: page.day,
      notebook: page.notebook,
      page,
      fetch: (...args) => fetch(...args),
      storage: localStorage,
      onSaved: (saved) =>
        queryClient.setQueryData(
          pageQuery(saved.day, saved.notebook).queryKey,
          saved,
        ),
    }),
  );
  useEffect(() => () => saver.dispose(), [saver]);

  const words = placeholders[page.notebook] ?? myJournal;
  const editor = useEditor({
    extensions: [
      ...documentExtensions,
      UndoRedo,
      Placeholder.configure({
        placeholder: ({ pos }) => (pos === 0 ? words.title : ""),
        showOnlyCurrent: false,
      }),
    ],
    content: page.doc,
    editable: page.editable,
    // Typed Markdown stays as literal characters.
    enableInputRules: false,
    enablePasteRules: false,
    immediatelyRender: true,
    editorProps: {
      attributes: {
        class: styles.prose ?? "",
        "aria-label": "Page",
        "data-body-placeholder": words.body,
      },
    },
    onUpdate: ({ editor }) => saver.edit(editor.getJSON() as Doc),
    onBlur: () => saver.flush(),
  });

  return <EditorContent editor={editor} className={styles.editor} />;
}
