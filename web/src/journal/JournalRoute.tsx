import {
  useMutation,
  useQueryClient,
  useSuspenseQuery,
} from "@tanstack/react-query";
import { createLazyRoute, useNavigate } from "@tanstack/react-router";
import { useState } from "react";
import { logout, type Me, setTimezone } from "../api";
import type { Notice } from "../notices/NoticePill";
import { keepTimezone, timezoneNotice } from "../notices/timezone";
import { PageEditor } from "../page/PageEditor";
import { pageQuery } from "../page/queries";
import { Sidebar } from "../rail/Sidebar";
import { meQuery } from "../router";
import { ShortcutSheet } from "../shortcuts/ShortcutSheet";
import styles from "./JournalRoute.module.css";
import { NotebookIndex } from "./NotebookIndex";

export const Route = createLazyRoute("/day/$day/$notebook")({
  component: JournalRoute,
});

function JournalRoute() {
  const { day, notebook } = Route.useParams();
  const { data: me } = useSuspenseQuery(meQuery);
  const { data: page } = useSuspenseQuery(pageQuery(day, notebook));
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const timezone = useTimezoneNotice(me);
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

  const signOut = useMutation({
    mutationFn: logout,
    onSuccess: async () => {
      queryClient.clear();
      await navigate({ to: "/signin" });
    },
  });

  return (
    <div className={styles.app}>
      <Sidebar
        me={me}
        day={day}
        notebook={notebook}
        onOpenResult={(r, q) => {
          setOpened({ day: r.day, notebook: r.notebook, q });
          void navigate({
            to: "/day/$day/$notebook",
            params: { day: r.day, notebook: r.notebook },
          });
        }}
      />
      <main className={styles.desk}>
        <header className={styles.bar}>
          <span>{me.email}</span>
          <button
            type="button"
            onClick={() => signOut.mutate()}
            disabled={signOut.isPending}
          >
            Sign out
          </button>
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
          />
          <NotebookIndex day={page.day} active={page.notebook} />
        </article>
        <ShortcutSheet />
      </main>
    </div>
  );
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
