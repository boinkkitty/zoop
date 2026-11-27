import { useQueryClient } from "@tanstack/react-query";
import { useRouter } from "@tanstack/react-router";
import type { Editor } from "@tiptap/core";
import { Placeholder, UndoRedo } from "@tiptap/extensions";
import { EditorContent, useEditor } from "@tiptap/react";
import { useEffect, useRef, useState, useSyncExternalStore } from "react";
import type { Page } from "../api";
import { type Notice, NoticePill } from "../notices/NoticePill";
import type { Notebook } from "./notebooks";
import styles from "./PageEditor.module.css";
import { pageQuery } from "./queries";
import { createPageSaver, type PageSaver, type SaverStatus } from "./saver";
import { type Doc, documentExtensions } from "./schema";

// Only My Journal has a Title: its first line, bold by style. I Learned is plain lines.
type Words = { first: string; body: string; titled: boolean };
const notebookWords: Record<Notebook, Words> = {
  "my-journal": { first: "Title", body: "Write about your day…", titled: true },
  "i-learned": { first: "What did you learn today?", body: "", titled: false },
};

// The editor for one Page, with its Notices. Mount it with a key per Page so each gets its own saver.
export function PageEditor({ page, owner }: { page: Page; owner: string }) {
  const queryClient = useQueryClient();
  const editorRef = useRef<Editor | null>(null);
  const [saver] = useState(() =>
    createPageSaver({
      day: page.day,
      notebook: page.notebook,
      page,
      owner,
      fetch: (...args) => fetch(...args),
      storage: localStorage,
      onSaved: (saved) =>
        queryClient.setQueryData(
          pageQuery(saved.day, saved.notebook).queryKey,
          saved,
        ),
      onReplace: (doc) =>
        editorRef.current?.commands.setContent(doc, { emitUpdate: false }),
    }),
  );
  // The Page is refetched when the window regains focus: a newer version from another device.
  useEffect(() => saver.remoteChanged(page), [saver, page]);
  useFlushWhenLeaving(saver);
  const status = useSyncExternalStore(saver.subscribe, saver.status);

  const words = notebookWords[page.notebook];
  const editor = useEditor({
    extensions: [
      ...documentExtensions(page.notebook),
      UndoRedo,
      Placeholder.configure({
        placeholder: ({ pos }) => (pos === 0 ? words.first : ""),
        showOnlyCurrent: false,
      }),
    ],
    content: saver.restored ?? page.doc,
    editable: page.editable,
    // Typed Markdown stays as literal characters.
    enableInputRules: false,
    enablePasteRules: false,
    immediatelyRender: true,
    editorProps: {
      attributes: {
        class: [styles.prose, words.titled && styles.titled]
          .filter(Boolean)
          .join(" "),
        "aria-label": "Page",
        "data-body-placeholder": words.body,
      },
    },
    onUpdate: ({ editor }) => saver.edit(editor.getJSON() as Doc),
    onBlur: () => saver.flush(),
  });
  editorRef.current = editor;

  return (
    <>
      <NoticePill notices={useSaverNotices(saver, status)} />
      <EditorContent editor={editor} className={styles.editor} />
    </>
  );
}

// Saves what's pending when the page is hidden, unloaded, or left for another route.
// A route change (another Notebook, back/forward) flushes before the next route loads.
function useFlushWhenLeaving(saver: PageSaver) {
  const router = useRouter();
  useEffect(() => {
    const onVisibility = () => {
      if (document.visibilityState === "hidden") saver.flush();
    };
    const onPageHide = () => saver.flushOnUnload();
    document.addEventListener("visibilitychange", onVisibility);
    window.addEventListener("pagehide", onPageHide);
    const unsubscribe = router.subscribe("onBeforeNavigate", () =>
      saver.flush(),
    );
    return () => {
      document.removeEventListener("visibilitychange", onVisibility);
      window.removeEventListener("pagehide", onPageHide);
      unsubscribe();
      saver.dispose(); // another Page
    };
  }, [saver, router]);
}

function useSaverNotices(saver: PageSaver, status: SaverStatus): Notice[] {
  const now = useNow(status.notSaved !== null);
  const notices: Notice[] = [];
  if (status.outdated) {
    notices.push({
      id: "update",
      text: "Zoop was updated. Reload to keep saving. What you wrote is kept on this device.",
      actions: [{ label: "Reload", onClick: () => window.location.reload() }],
    });
  }
  if (status.conflict) {
    notices.push({
      id: "conflict",
      text: "This Page changed on another device.",
      actions: [
        { label: "Keep this version", onClick: () => saver.keepMine() },
        { label: "Use the other", onClick: () => saver.takeTheirs() },
      ],
    });
  }
  if (status.notSaved) {
    const secs = Math.max(0, Math.ceil((status.notSaved.retryAt - now) / 1000));
    notices.push({
      id: "notSaved",
      text: `Not saved, retrying in ${secs}s. Your words are kept on this device.`,
      actions: [{ label: "Try now", onClick: () => saver.retryNow() }],
    });
  }
  return notices;
}

// The current time, ticking each second while on.
function useNow(on: boolean) {
  const [now, setNow] = useState(Date.now);
  useEffect(() => {
    if (!on) return;
    setNow(Date.now());
    const t = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(t);
  }, [on]);
  return now;
}
