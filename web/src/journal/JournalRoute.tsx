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
import styles from "./JournalRoute.module.css";

export const Route = createLazyRoute("/day/$day/$notebook")({
  component: JournalRoute,
});

// A Day is a calendar date, so it's formatted in UTC: never shifted by the browser's zone.
const weekday = new Intl.DateTimeFormat(undefined, {
  weekday: "long",
  timeZone: "UTC",
});
const longDate = new Intl.DateTimeFormat(undefined, {
  day: "numeric",
  month: "long",
  year: "numeric",
  timeZone: "UTC",
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

  const date = new Date(`${page.day}T00:00:00Z`);
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
        <header className={styles.head}>
          <p className={styles.weekday}>
            {weekday.format(date)}
            {page.day === me.today && " · Today"}
          </p>
          <h1 className={styles.date}>{longDate.format(date)}</h1>
        </header>
        <PageEditor
          key={`${page.day}/${page.notebook}`}
          page={page}
          owner={me.email}
        />
      </article>
    </main>
  );
}
