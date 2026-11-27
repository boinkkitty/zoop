import { useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import type { Editor } from "@tiptap/core";
import { Placeholder, UndoRedo } from "@tiptap/extensions";
import { EditorContent, useEditor } from "@tiptap/react";
import { useEffect, useRef, useState, useSyncExternalStore } from "react";
import type { Page } from "../api";
import { type Notice, NoticePill } from "../notices/NoticePill";
import { DayHead } from "./DayHead";
import styles from "./PageEditor.module.css";
import { pageQuery } from "./queries";
import { createPageSaver, type PageSaver, type SaverStatus } from "./saver";
import { type Doc, documentExtensions } from "./schema";

type Words = { title: string; body: string };
const myJournal: Words = { title: "Title", body: "Write about your day…" };
const placeholders: Record<string, Words> = {
  "my-journal": myJournal,
  "i-learned": { title: "What did you learn today?", body: "" },
};

// The editor for one Page, with its date and Notices. Mount it with a key per Page so each gets its own saver.
export function PageEditor({
  page,
  owner,
  today,
  timezone,
  accountNotices = [],
}: {
  page: Page;
  owner: string;
  today: string;
  timezone: string;
  accountNotices?: Notice[]; // about the account rather than this Page
}) {
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
    content: saver.restored ?? page.doc,
    editable: !status.locked,
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
  editorRef.current = editor;
  // At locksAt the saver has sent the last words; now the editor stops taking more.
  useEffect(() => {
    if (status.locked && editor.isEditable) editor.setEditable(false, false);
  }, [editor, status.locked]);

  const notices = useSaverNotices(saver, status);
  const dayClosed = useDayClosedNotice(page, today, status.locked);
  if (dayClosed) notices.push(dayClosed);
  notices.push(...accountNotices);

  return (
    <>
      <NoticePill notices={notices} />
      <DayHead
        page={page}
        today={today}
        timezone={timezone}
        readOnly={status.locked}
      />
      <EditorContent editor={editor} className={styles.editor} />
    </>
  );
}

// The Day closed Notice: on yesterday's Page once it's read only, until dismissed for that Day.
function useDayClosedNotice(
  page: Page,
  today: string,
  locked: boolean,
): Notice | null {
  const navigate = useNavigate();
  const key = `zoop:dismissed:dayClosed:${page.day}`;
  const [dismissed, setDismissed] = useState(
    () => localStorage.getItem(key) !== null,
  );
  if (!locked || dismissed || page.day !== dayBefore(today)) return null;
  return {
    id: "dayClosed",
    text: "Yesterday closed at 02:00. Everything you wrote is saved.",
    actions: [
      {
        label: "Go to today",
        onClick: () =>
          navigate({
            to: "/day/$day/$notebook",
            params: { day: today, notebook: page.notebook },
          }),
      },
    ],
    onDismiss: () => {
      localStorage.setItem(key, "1");
      setDismissed(true);
    },
  };
}

const dayBefore = (day: string) =>
  new Date(Date.parse(`${day}T00:00:00Z`) - 86_400_000)
    .toISOString()
    .slice(0, 10);

// Saves what's pending when the page is hidden, unloaded, or left for another route.
function useFlushWhenLeaving(saver: PageSaver) {
  useEffect(() => {
    const onVisibility = () => {
      if (document.visibilityState === "hidden") saver.flush();
    };
    const onPageHide = () => saver.flushOnUnload();
    document.addEventListener("visibilitychange", onVisibility);
    window.addEventListener("pagehide", onPageHide);
    return () => {
      document.removeEventListener("visibilitychange", onVisibility);
      window.removeEventListener("pagehide", onPageHide);
      saver.dispose(); // route change or another Page
    };
  }, [saver]);
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
