import {
  useMutation,
  useQueryClient,
  useSuspenseQuery,
} from "@tanstack/react-query";
import { createLazyRoute } from "@tanstack/react-router";
import { useEffect, useState } from "react";
import { type Me, setTimezone } from "../api";
import type { Notice } from "../notices/NoticePill";
import { keepTimezone, timezoneNotice } from "../notices/timezone";
import { PageEditor } from "../page/PageEditor";
import { pageQuery } from "../page/queries";
import { Sidebar } from "../rail/Sidebar";
import { meQuery } from "../router";
import { DeactivateDialog, SettingsDialog } from "../settings/SettingsDialog";
import { ShortcutSheet } from "../shortcuts/ShortcutSheet";
import { isSettingsShortcut, isSheetShortcut } from "../shortcuts/shortcuts";
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
  const [dialog, setDialog] = useDialog();
  const openIf = (name: Dialog) => (open: boolean) =>
    setDialog(open ? name : null);

  return (
    <div className={styles.app}>
      <Sidebar
        me={me}
        day={day}
        notebook={notebook}
        onOpenSettings={() => setDialog("settings")}
      />
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
          />
          <NotebookIndex day={page.day} active={page.notebook} />
        </article>
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
