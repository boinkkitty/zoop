import { useQueryClient } from "@tanstack/react-query";
import { useNavigate, useRouter } from "@tanstack/react-router";
import type { Editor } from "@tiptap/core";
import { Placeholder, UndoRedo } from "@tiptap/extensions";
import { EditorContent, useEditor } from "@tiptap/react";
import { useEffect, useRef, useState, useSyncExternalStore } from "react";
import { type Page, resendVerify } from "../api";
import { type Notice, NoticePill } from "../notices/NoticePill";
import { type ResendState, verifyEmailNotice } from "../notices/verifyEmail";
import { railKey } from "../rail/queries";
import { queryWords } from "../search/match";
import search from "../search/Search.module.css";
import { SearchHighlight, searchHighlightKey } from "../search/SearchHighlight";
import { DayHead } from "./DayHead";
import { editorOptions } from "./formatting";
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

// The editor for one Page, with its date and Notices. Mount it with a key per Page so each gets its own saver.
export function PageEditor({
  page,
  owner,
  emailVerified,
  today,
  timezone,
  accountNotices = [],
  searchQuery = null,
  onClearSearch = () => {},
  onEdit = () => {},
}: {
  page: Page;
  owner: string;
  emailVerified: boolean;
  today: string;
  timezone: string;
  accountNotices?: Notice[]; // about the account rather than this Page
  searchQuery?: string | null; // the search this Page was opened from: its matches are marked
  onClearSearch?: () => void;
  onEdit?: () => void; // the user typed (first run ends its coach marks)
}) {
  const queryClient = useQueryClient();
  const editorRef = useRef<Editor | null>(null);
  const onClearSearchRef = useRef(onClearSearch);
  onClearSearchRef.current = onClearSearch;
  const onEditRef = useRef(onEdit);
  onEditRef.current = onEdit;
  const [saver] = useState(() =>
    createPageSaver({
      day: page.day,
      notebook: page.notebook,
      page,
      owner,
      fetch: (...args) => fetch(...args),
      storage: localStorage,
      onSaved: (saved) => {
        queryClient.setQueryData(
          pageQuery(saved.day, saved.notebook).queryKey,
          saved,
        );
        // The Day's Island shows its Title and glimpses, and the scrubber its Written Days.
        void queryClient.invalidateQueries({ queryKey: railKey });
      },
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
    ...editorOptions,
    extensions: [
      ...documentExtensions(page.notebook),
      ...editorOptions.extensions,
      UndoRedo,
      SearchHighlight.configure({
        className: search.match,
        currentClassName: search.current,
      }),
      Placeholder.configure({
        placeholder: ({ pos }) => (pos === 0 ? words.first : ""),
        showOnlyCurrent: false,
      }),
    ],
    content: saver.restored ?? page.doc,
    editable: !status.locked,
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
    onUpdate: ({ editor }) => {
      saver.edit(editor.getJSON() as Doc);
      onClearSearchRef.current(); // typing clears the matches
      onEditRef.current();
    },
    onBlur: () => saver.flush(),
  });
  editorRef.current = editor;
  // At locksAt the saver has sent the last words; now the editor stops taking more.
  useEffect(() => {
    if (status.locked && editor.isEditable) editor.setEditable(false, false);
  }, [editor, status.locked]);

  const matchWords = useSearchMatches(editor, searchQuery, onClearSearch);

  const notices = useSaverNotices(saver, status);
  const dayClosed = useDayClosedNotice(page, today, status.locked);
  if (dayClosed) notices.push(dayClosed);
  // Account Notices come last, in the spec's order: timezone, then verify email.
  notices.push(...accountNotices);
  const verify = useVerifyEmailNotice(owner, emailVerified);
  if (verify) notices.push(verify);

  return (
    <>
      <NoticePill notices={notices} />
      {searchQuery !== null && (
        <div className={search.matchPill}>
          <span>Matches for “{searchQuery.trim()}”</span>
          <button type="button" onClick={onClearSearch}>
            Clear · Esc
          </button>
        </div>
      )}
      <DayHead
        page={page}
        today={today}
        timezone={timezone}
        readOnly={status.locked}
        searchWords={matchWords}
      />
      <EditorContent
        editor={editor}
        className={styles.editor}
        data-coach="page"
      />
    </>
  );
}

// useSearchMatches marks q's matches on the Page, scrolls the first to the middle of the
// window, and clears them on Esc. It returns the words being marked.
function useSearchMatches(
  editor: Editor,
  q: string | null,
  onClear: () => void,
): string[] {
  const words = q === null ? [] : queryWords(q);
  const key = words.join(" ");
  useEffect(() => {
    const ws = key ? key.split(" ") : [];
    editor.view.dispatch(editor.state.tr.setMeta(searchHighlightKey, ws));
    if (ws.length > 0)
      editor.view.dom
        .querySelector("[data-search-match]")
        ?.scrollIntoView({ block: "center" });
  }, [editor, key]);
  useEffect(() => {
    if (q === null) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClear();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [q, onClear]);
  return words;
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

// Dismissing it lasts for this browser session: it comes back in a new one until the email is verified.
const verifyDismissedKey = "zoop:dismissed:verifyEmail";

// The verify email Notice, with Resend, until the email is verified or the Notice dismissed.
function useVerifyEmailNotice(
  email: string,
  emailVerified: boolean,
): Notice | null {
  const [dismissed, setDismissed] = useState(
    () => sessionStorage.getItem(verifyDismissedKey) !== null,
  );
  const [resend, setResend] = useState<ResendState>("idle");
  const n = verifyEmailNotice({ email, emailVerified, dismissed, resend });
  if (!n) return null;
  return {
    id: "verifyEmail",
    text: n.text,
    status: n.status,
    actions: n.action
      ? [
          {
            label: n.action,
            onClick: () => {
              setResend("sending");
              resendVerify(email).then(
                () => setResend("sent"),
                () => setResend("failed"),
              );
            },
          },
        ]
      : [],
    onDismiss: () => {
      sessionStorage.setItem(verifyDismissedKey, "1");
      setDismissed(true);
    },
  };
}

const dayBefore = (day: string) =>
  new Date(Date.parse(`${day}T00:00:00Z`) - 86_400_000)
    .toISOString()
    .slice(0, 10);

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
