import {
  useMutation,
  useQueryClient,
  useSuspenseQuery,
} from "@tanstack/react-query";
import { createLazyRoute, useNavigate } from "@tanstack/react-router";
import { logout } from "../api";
import { meQuery } from "../router";
import styles from "./JournalRoute.module.css";

export const Route = createLazyRoute("/day/$day/$notebook")({
  component: JournalRoute,
});

// A placeholder for today's Page until the sheet and editor arrive (#21).
function JournalRoute() {
  const { day } = Route.useParams();
  const { data: me } = useSuspenseQuery(meQuery);
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
        <h1 className={styles.date}>{day}</h1>
      </article>
    </main>
  );
}
