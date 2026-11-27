import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { type FormEvent, useEffect, useRef, useState } from "react";
import { forgotPassword, resetPassword, verifyEmail } from "../api";
import styles from "./AuthScreen.module.css";

// /verify?token=: POSTs the token once on arrival, so a scanner's GET of the link can't use it up.
export function VerifyScreen({ token }: { token: string }) {
  const queryClient = useQueryClient();
  const verify = useMutation({
    mutationFn: () => verifyEmail(token),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["me"] }),
  });
  const sent = useRef(false); // StrictMode runs effects twice; the token works once
  useEffect(() => {
    if (sent.current) return;
    sent.current = true;
    verify.mutate();
  }, [verify]);

  return (
    <main className={styles.screen}>
      <div className={styles.card}>
        <h1 className={styles.title}>
          {verify.isSuccess ? "Email verified" : "Verifying your email…"}
        </h1>
        {verify.error && (
          <p role="alert" className={styles.error}>
            {verify.error.message}
          </p>
        )}
        {!verify.isPending && (
          <p className={styles.switch}>
            <Link to="/">Go to your journal</Link>
          </p>
        )}
      </div>
    </main>
  );
}

// /forgot: asks for a reset link. It says the same whether or not the email has an account.
export function ForgotScreen() {
  const [email, setEmail] = useState("");
  const send = useMutation({ mutationFn: () => forgotPassword(email) });

  function onSubmit(e: FormEvent) {
    e.preventDefault();
    send.mutate();
  }

  return (
    <main className={styles.screen}>
      <form className={styles.card} onSubmit={onSubmit}>
        <h1 className={styles.title}>Reset your password</h1>
        {send.isSuccess ? (
          <p role="status">
            If {email} has an account, a link to reset its password is on its
            way. It works for an hour.
          </p>
        ) : (
          <>
            <label className={styles.field}>
              <span>Email</span>
              <input
                type="email"
                autoComplete="email"
                required
                value={email}
                onChange={(e) => setEmail(e.target.value)}
              />
            </label>
            {send.error && (
              <p role="alert" className={styles.error}>
                {send.error.message}
              </p>
            )}
            <button
              className={styles.submit}
              type="submit"
              disabled={send.isPending}
            >
              Send reset link
            </button>
          </>
        )}
        <p className={styles.switch}>
          <Link to="/signin">Back to sign in</Link>
        </p>
      </form>
    </main>
  );
}

// /reset?token=: sets a new password. Every device is signed out, so they sign in again after.
export function ResetScreen({ token }: { token: string }) {
  const [password, setPassword] = useState("");
  const reset = useMutation({
    mutationFn: () => resetPassword(token, password),
  });

  function onSubmit(e: FormEvent) {
    e.preventDefault();
    reset.mutate();
  }

  return (
    <main className={styles.screen}>
      <form className={styles.card} onSubmit={onSubmit}>
        <h1 className={styles.title}>Choose a new password</h1>
        {reset.isSuccess ? (
          <p role="status">
            Your password is set. You've been signed out everywhere.
          </p>
        ) : (
          <>
            <label className={styles.field}>
              <span>New password</span>
              <input
                type="password"
                autoComplete="new-password"
                required
                minLength={8}
                value={password}
                onChange={(e) => setPassword(e.target.value)}
              />
            </label>
            {reset.error && (
              <p role="alert" className={styles.error}>
                {reset.error.message}
              </p>
            )}
            <button
              className={styles.submit}
              type="submit"
              disabled={reset.isPending}
            >
              Set password
            </button>
          </>
        )}
        <p className={styles.switch}>
          <Link to="/signin">Sign in</Link>
        </p>
      </form>
    </main>
  );
}
