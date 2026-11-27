import {
  useMutation,
  useQueryClient,
  useSuspenseQuery,
} from "@tanstack/react-query";
import { createLazyRoute, useNavigate } from "@tanstack/react-router";
import { logout } from "../api";
import { PageEditor } from "../page/PageEditor";
import { pageQuery } from "../page/queries";
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

  const signOut = useMutation({
    mutationFn: logout,
    onSuccess: async () => {
      queryClient.clear();
      await navigate({ to: "/signin" });
    },
  });

  return (
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
          today={me.today}
          timezone={me.timezone}
        />
        <NotebookIndex day={page.day} active={page.notebook} />
      </article>
      <ShortcutSheet />
    </main>
  );
}
