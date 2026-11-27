import { Link } from "@tanstack/react-router";
import styles from "./PrivacyPage.module.css";

// The public privacy page (no sign-in). Google's OAuth consent screen links here.
// Keep its claims plain: hosting-provider encryption only, never end-to-end, and no export.
export function PrivacyPage() {
  return (
    <main className={styles.screen}>
      <article className={styles.sheet}>
        <h1>Privacy</h1>
        <p className={styles.lede}>
          Zoop is a journal, so here is plainly how yours is kept.
        </p>

        <h2>Where your journal lives</h2>
        <p>
          Your account and Pages are stored in a database run by Railway, in
          Singapore. Nightly backups are encrypted and kept on Cloudflare R2.
          Everything is encrypted in transit and at rest by our hosting
          providers.
        </p>

        <h2>Who handles it</h2>
        <p>These services process data for Zoop:</p>
        <ul>
          <li>
            <strong>Railway</strong>: runs the app and its database.
          </li>
          <li>
            <strong>Cloudflare R2</strong>: stores the encrypted backups.
          </li>
          <li>
            <strong>Resend</strong>: sends account emails, such as verifying
            your address or resetting your password.
          </li>
          <li>
            <strong>Sentry</strong>: receives error reports, so we can fix what
            breaks.
          </li>
          <li>
            <strong>Google</strong>: only if you choose Continue with Google, to
            sign you in.
          </li>
        </ul>

        <h2>What is never logged</h2>
        <p>
          What you write on your Pages, your email address, and sign-in or email
          tokens are never written to our logs or sent to Sentry.
        </p>

        <h2>Deactivating your account</h2>
        <p>
          You can deactivate your account from Settings. Your journal is hidden
          at once and you're signed out everywhere. It's erased within a year,
          and gone from backups 90 days after that. There's no way back, and the
          email can't be used for a new account.
        </p>

        <p className={styles.back}>
          <Link to="/">Back to Zoop</Link>
        </p>
      </article>
    </main>
  );
}
