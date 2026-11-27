import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Link, useNavigate } from "@tanstack/react-router";
import { type FormEvent, useState } from "react";
import { googleErrors, googleStartURL, login, signup } from "../api";
import styles from "./AuthScreen.module.css";

type Mode = "signin" | "signup";

const copy = {
  signin: {
    title: "Sign in to Zoop",
    submit: "Sign in",
    switchPrompt: "New to Zoop?",
    switchLink: "Create an account",
    switchTo: "/signup",
    autoComplete: "current-password",
  },
  signup: {
    title: "Start your journal",
    submit: "Create account",
    switchPrompt: "Already have an account?",
    switchLink: "Sign in",
    switchTo: "/signin",
    autoComplete: "new-password",
  },
} as const;

export function AuthScreen({
  mode,
  googleError,
}: {
  mode: Mode;
  googleError?: string; // the code the Google callback sent back, if it refused
}) {
  const text = copy[mode];
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const queryClient = useQueryClient();
  const navigate = useNavigate();

  const submit = useMutation({
    mutationFn: () =>
      mode === "signup" ? signup(email, password) : login(email, password),
    onSuccess: async () => {
      queryClient.removeQueries({ queryKey: ["me"] });
      await navigate({ to: "/" });
    },
  });

  function onSubmit(e: FormEvent) {
    e.preventDefault();
    submit.mutate();
  }

  return (
    <main className={styles.screen}>
      <form className={styles.card} onSubmit={onSubmit}>
        <h1 className={styles.title}>{text.title}</h1>
        <a className={styles.google} href={googleStartURL()}>
          Continue with Google
        </a>
        <p className={styles.or}>or</p>
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
        <label className={styles.field}>
          <span>Password</span>
          <input
            type="password"
            autoComplete={text.autoComplete}
            required
            minLength={mode === "signup" ? 8 : undefined}
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
        </label>
        {(submit.error || googleError) && (
          <p role="alert" className={styles.error}>
            {submit.error
              ? submit.error.message
              : (googleErrors[googleError ?? ""] ?? googleErrors.google_failed)}
          </p>
        )}
        <button
          className={styles.submit}
          type="submit"
          disabled={submit.isPending}
        >
          {text.submit}
        </button>
        <p className={styles.switch}>
          {text.switchPrompt} <Link to={text.switchTo}>{text.switchLink}</Link>
        </p>
      </form>
    </main>
  );
}
