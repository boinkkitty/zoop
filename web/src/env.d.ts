interface ImportMetaEnv {
  /** Sentry DSN for the browser; unset turns error reporting off. */
  readonly VITE_SENTRY_DSN?: string;
  readonly VITE_SENTRY_RELEASE?: string;
}
