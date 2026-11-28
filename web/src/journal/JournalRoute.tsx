import {
  useMutation,
  useQueryClient,
  useSuspenseQuery,
} from "@tanstack/react-query";
import { createLazyRoute, useNavigate } from "@tanstack/react-router";
import { useEffect, useState } from "react";
import { type Me, setTimezone } from "../api";
import { CoachMarks, useCoachMarks } from "../firstrun/CoachMarks";
import type { Notice } from "../notices/NoticePill";
import { keepTimezone, timezoneNotice } from "../notices/timezone";
import { PageEditor } from "../page/PageEditor";
import { pageQuery } from "../page/queries";
import { Sidebar } from "../rail/Sidebar";
import { meQuery } from "../router";
import { DeactivateDialog, SettingsDialog } from "../settings/SettingsDialog";
import { ShortcutSheet } from "../shortcuts/ShortcutSheet";
import {
  isSearchShortcut,
  isSettingsShortcut,
  isSheetShortcut,
} from "../shortcuts/shortcuts";
import { Drawer, useNarrow } from "./Drawer";
import styles from "./JournalRoute.module.css";
import { NotebookIndex } from "./NotebookIndex";

export const Route = createLazyRoute("/day/$day/$notebook")({
  component: JournalRoute,
});

function JournalRoute() {
  const { day, notebook } = Route.useParams();
  const { data: me } = useSuspenseQuery(meQuery);
  const { data: page } = useSuspenseQuery(pageQuery(day, notebook));
  const timezone = useTimezoneNotice(me);
  const navigate = useNavigate();
  // The query a search result was opened with, kept until the user moves on from its matches.
  const [opened, setOpened] = useState<{
    day: string;
    notebook: string;
    q: string;
  } | null>(null);
  const searchQuery =
    opened && opened.day === day && opened.notebook === notebook
      ? opened.q
      : null;
  // Kept here so the query (and its results) are still there when the drawer reopens.
  const [query, setQuery] = useState("");
  const coach = useCoachMarks();
  const [dialog, setDialog] = useDialog();
  const openIf = (name: Dialog) => (open: boolean) =>
    setDialog(open ? name : null);
  const narrow = useNarrow();
  const drawer = useDrawer(narrow, dialog !== null);

  const sidebar = (
    <Sidebar
      me={me}
      day={day}
      notebook={notebook}
      query={query}
      onQueryChange={setQuery}
      onOpenResult={(r, q) => {
        setOpened({ day: r.day, notebook: r.notebook, q });
        void navigate({
          to: "/day/$day/$notebook",
          params: { day: r.day, notebook: r.notebook },
        });
      }}
      onOpenSettings={() => setDialog("settings")}
      onChoose={drawer.close}
    />
  );

  return (
    <div className={`${styles.app} ${narrow ? styles.narrow : ""}`}>
      {narrow ? (
        <Drawer
          open={drawer.open}
          onOpenChange={drawer.setOpen}
          focusSearch={drawer.focusSearch}
        >
          {sidebar}
        </Drawer>
      ) : (
        sidebar
      )}
      <main className={styles.desk}>
        <header className={styles.bar}>
          <span>{me.email}</span>
        </header>
        <article className={styles.sheet}>
          <PageEditor
            key={`${page.day}/${page.notebook}`}
            page={page}
            owner={me.email}
            emailVerified={me.emailVerified}
            today={me.today}
            timezone={me.timezone}
            accountNotices={timezone ? [timezone] : []}
            searchQuery={searchQuery}
            onClearSearch={() => setOpened(null)}
            onEdit={coach.step === null ? undefined : coach.end}
          />
          <NotebookIndex day={page.day} active={page.notebook} />
        </article>
        <CoachMarks
          step={coach.step}
          target={narrow ? (m) => (m === "rail" ? "menu" : m) : undefined}
          onNext={coach.next}
          onSkip={coach.end}
        />
        <SettingsDialog
          me={me}
          open={dialog === "settings"}
          onOpenChange={openIf("settings")}
          onShowShortcuts={() => setDialog("shortcuts")}
          onDeactivate={() => setDialog("deactivate")}
        />
        <ShortcutSheet
          open={dialog === "shortcuts"}
          onOpenChange={openIf("shortcuts")}
        />
        <DeactivateDialog
          me={me}
          open={dialog === "deactivate"}
          onOpenChange={openIf("deactivate")}
        />
      </main>
    </div>
  );
}

type Dialog = "settings" | "shortcuts" | "deactivate";

// useDrawer is the narrow-window drawer's state. ⌘K opens it with search focused (once
// open, the Sidebar's own ⌘K handles it), and it closes before any dialog opens.
function useDrawer(narrow: boolean, dialogOpen: boolean) {
  const [state, setState] = useState({ open: false, focusSearch: false });
  useEffect(() => {
    if (!narrow || state.open) return;
    const onKey = (e: KeyboardEvent) => {
      if (!isSearchShortcut(e)) return;
      e.preventDefault();
      setState({ open: true, focusSearch: true });
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [narrow, state.open]);
  useEffect(() => {
    if (dialogOpen) setState({ open: false, focusSearch: false });
  }, [dialogOpen]);

  return {
    open: narrow && state.open,
    focusSearch: state.focusSearch,
    setOpen: (open: boolean) => setState({ open, focusSearch: false }),
    close: () => setState({ open: false, focusSearch: false }),
  };
}

// useDialog is the one dialog open, if any, so they never stack. ⌘, toggles Settings and ⌘/ the
// shortcut sheet, from anywhere in the signed-in app; either replaces whatever else is open.
function useDialog() {
  const [dialog, setDialog] = useState<Dialog | null>(null);
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      const want = isSettingsShortcut(e)
        ? "settings"
        : isSheetShortcut(e)
          ? "shortcuts"
          : null;
      if (!want) return;
      e.preventDefault();
      setDialog((open) => (open === want ? null : want));
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);
  return [dialog, setDialog] as const;
}

// The timezone Notice, from comparing the browser's zone with the profile on load.
function useTimezoneNotice(me: Me): Notice | null {
  const queryClient = useQueryClient();
  const [browserZone] = useState(
    () => Intl.DateTimeFormat().resolvedOptions().timeZone,
  );
  const [answered, setAnswered] = useState(false);
  const switchZone = useMutation({
    mutationFn: setTimezone,
    onSuccess: (updated) => {
      queryClient.setQueryData(meQuery.queryKey, updated);
      setAnswered(true);
    },
  });
  const ask = timezoneNotice({ browserZone, me, storage: localStorage });
  if (!ask || answered) return null;
  return {
    id: "timezone",
    text: `Looks like you're on ${ask.place} time. Start your Days on ${ask.place} time from tomorrow?`,
    actions: [
      {
        label: "Switch",
        onClick: () => {
          if (!switchZone.isPending) switchZone.mutate(ask.zone);
        },
      },
      {
        label: `Keep ${ask.currentPlace}`,
        onClick: () => {
          keepTimezone(localStorage, ask.zone);
          setAnswered(true);
        },
      },
    ],
  };
}
