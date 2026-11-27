import * as Dialog from "@radix-ui/react-dialog";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { useId, useMemo, useState } from "react";
import {
  deactivate,
  forgotPassword,
  logout,
  type Me,
  resendVerify,
  setTimezone,
} from "../api";
import { placeName } from "../notices/timezone";
import { meQuery } from "../router";
import { keyLabels } from "../shortcuts/shortcuts";
import styles from "./SettingsDialog.module.css";
import { pendingSwitch, searchZones } from "./timezones";

// Settings: everything about the account, in one dialog opened from the sidebar or ⌘,.
export function SettingsDialog({
  me,
  open,
  onOpenChange,
  onShowShortcuts,
  onDeactivate,
}: {
  me: Me;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onShowShortcuts: () => void; // closes Settings and opens the shortcut sheet
  onDeactivate: () => void; // closes Settings and opens the confirmation
}) {
  const signOut = useSignOut(logout);

  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <Dialog.Portal>
        <Dialog.Overlay className={styles.overlay} />
        <Dialog.Content className={styles.dialog} aria-describedby={undefined}>
          <Dialog.Title className={styles.title}>Settings</Dialog.Title>

          <Row title="Keyboard shortcuts">
            <button
              type="button"
              className={styles.link}
              aria-label="Show keyboard shortcuts"
              onClick={onShowShortcuts}
            >
              {keyLabels(["Mod", "/"]).map((k) => (
                <kbd key={k}>{k}</kbd>
              ))}
            </button>
          </Row>

          <Row title="Email">
            <span className={styles.value}>{me.email}</span>
            {me.emailVerified ? (
              <span className={styles.muted}>Verified</span>
            ) : (
              <ResendEmail email={me.email} />
            )}
          </Row>

          <Row title="Sign-in">
            <PasswordByEmail me={me} />
            <span className={styles.muted}>
              Google{" "}
              {me.signInMethods.includes("google")
                ? "connected"
                : "not connected"}
            </span>
          </Row>

          <Row title="Timezone">
            <TimezonePicker me={me} />
          </Row>

          <Row title="Privacy">
            <a
              className={styles.link}
              href="/privacy"
              target="_blank"
              rel="noopener"
            >
              How your journal is kept
            </a>
          </Row>

          <Row title="Sign out">
            <button
              type="button"
              className={styles.button}
              disabled={signOut.isPending}
              onClick={() => signOut.mutate()}
            >
              Sign out
            </button>
          </Row>

          <Row title="Deactivate account">
            <button
              type="button"
              className={`${styles.button} ${styles.danger}`}
              onClick={onDeactivate}
            >
              Deactivate account
            </button>
          </Row>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}

function Row({
  title,
  children,
}: {
  title: string;
  children: React.ReactNode;
}) {
  return (
    <section className={styles.row}>
      <h3 className={styles.rowTitle}>{title}</h3>
      <div className={styles.rowBody}>{children}</div>
    </section>
  );
}

// useSignOut runs end (sign out, or deactivate) and then leaves for the sign-in screen.
export function useSignOut<T>(end: (arg: T) => Promise<void>) {
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  return useMutation({
    mutationFn: end,
    onSuccess: async () => {
      queryClient.clear();
      await navigate({ to: "/signin" });
    },
  });
}

function ResendEmail({ email }: { email: string }) {
  const resend = useMutation({ mutationFn: () => resendVerify(email) });
  if (resend.isSuccess) return <span className={styles.muted}>Sent</span>;
  return (
    <button
      type="button"
      className={styles.button}
      disabled={resend.isPending}
      onClick={() => resend.mutate()}
    >
      {resend.isError ? "Try again" : "Resend email"}
    </button>
  );
}

// The password is changed or set through the reset email, so whoever does it holds the inbox.
function PasswordByEmail({ me }: { me: Me }) {
  const send = useMutation({ mutationFn: () => forgotPassword(me.email) });
  const label = me.signInMethods.includes("password")
    ? "Change password"
    : "Set a password";
  if (send.isSuccess)
    return (
      <span className={styles.muted} role="status">
        Check your inbox for a link
      </span>
    );
  return (
    <button
      type="button"
      className={styles.button}
      disabled={send.isPending}
      onClick={() => send.mutate()}
    >
      {label}
    </button>
  );
}

const allZones: readonly string[] = Intl.supportedValuesOf("timeZone");

function TimezonePicker({ me }: { me: Me }) {
  const queryClient = useQueryClient();
  const listId = useId();
  const [query, setQuery] = useState("");
  const matches = useMemo(
    () => (query.trim() ? searchZones(query, allZones).slice(0, 50) : []),
    [query],
  );
  const choose = useMutation({
    mutationFn: setTimezone,
    onSuccess: (updated) => {
      queryClient.setQueryData(meQuery.queryKey, updated);
      setQuery("");
    },
  });
  const pending = pendingSwitch(me);

  return (
    <div className={styles.timezone}>
      <span className={styles.value}>{placeName(me.timezone)}</span>
      <input
        className={styles.search}
        role="combobox"
        aria-label="Timezone"
        aria-expanded={matches.length > 0}
        aria-controls={listId}
        aria-autocomplete="list"
        placeholder="Search for a city"
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        onKeyDown={(e) => {
          const first = matches[0];
          if (e.key === "Enter" && first) choose.mutate(first);
        }}
      />
      {matches.length > 0 && (
        <div id={listId} role="listbox" className={styles.options}>
          {matches.map((zone) => (
            <button
              key={zone}
              type="button"
              role="option"
              aria-selected={false}
              className={styles.option}
              onClick={() => choose.mutate(zone)}
            >
              {placeName(zone)}
              <span className={styles.muted}>{zone}</span>
            </button>
          ))}
        </div>
      )}
      {pending && (
        <p className={styles.muted}>
          {pending} ·{" "}
          <button
            type="button"
            className={styles.link}
            disabled={choose.isPending}
            onClick={() => choose.mutate(me.timezone)}
          >
            Cancel
          </button>
        </p>
      )}
      {choose.isError && (
        <p role="alert" className={styles.error}>
          {choose.error.message}
        </p>
      )}
    </div>
  );
}

// The confirmation, shown in place of Settings: typing the account's email is the only way through.
export function DeactivateDialog({
  me,
  open,
  onOpenChange,
}: {
  me: Me;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const [typed, setTyped] = useState("");
  const leave = useSignOut(deactivate);
  const matches = typed.trim().toLowerCase() === me.email;

  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <Dialog.Portal>
        <Dialog.Overlay className={styles.overlay} />
        <Dialog.Content className={`${styles.dialog} ${styles.confirm}`}>
          <Dialog.Title className={styles.title}>
            Deactivate account
          </Dialog.Title>
          <Dialog.Description className={styles.description}>
            Your journal is hidden at once and you're signed out everywhere.
            It's erased within a year, and gone from backups 90 days after that.
            There's no way back, and {me.email} can't be used for a new account.
          </Dialog.Description>
          <form
            onSubmit={(e) => {
              e.preventDefault();
              if (matches) leave.mutate(typed);
            }}
          >
            <label className={styles.field}>
              <span>Type your email to confirm</span>
              <input
                type="email"
                autoComplete="off"
                value={typed}
                onChange={(e) => setTyped(e.target.value)}
              />
            </label>
            {leave.isError && (
              <p role="alert" className={styles.error}>
                {leave.error.message}
              </p>
            )}
            <div className={styles.actions}>
              <Dialog.Close className={styles.button}>
                Keep my account
              </Dialog.Close>
              <button
                type="submit"
                className={`${styles.button} ${styles.dangerSolid}`}
                disabled={!matches || leave.isPending}
              >
                Deactivate
              </button>
            </div>
          </form>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
