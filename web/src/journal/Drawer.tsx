import * as Dialog from "@radix-ui/react-dialog";
import { type ReactNode, useSyncExternalStore } from "react";
import styles from "./Drawer.module.css";

// A window too narrow for the sidebar beside the sheet (the 760px window of the rail prototype).
const narrowQuery = "(max-width: 900px)";

export function useNarrow(): boolean {
  return useSyncExternalStore(
    (onChange) => {
      const mq = window.matchMedia(narrowQuery);
      mq.addEventListener("change", onChange);
      return () => mq.removeEventListener("change", onChange);
    },
    () => window.matchMedia(narrowQuery).matches,
  );
}

// The narrow-window sidebar: a menu button at the top-left slides it in over the sheet,
// with a light scrim. Choosing a Day, clicking outside or Esc closes it.
export function Drawer({
  open,
  onOpenChange,
  focusSearch,
  children,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  focusSearch: boolean; // opened by ⌘K: search gets the focus, not Today
  children: ReactNode;
}) {
  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <Dialog.Trigger
        className={styles.menu}
        aria-label="Open sidebar"
        data-coach="menu"
      >
        <MenuIcon />
      </Dialog.Trigger>
      <Dialog.Portal>
        <Dialog.Overlay className={styles.scrim} />
        <Dialog.Content
          className={styles.drawer}
          aria-describedby={undefined}
          onOpenAutoFocus={(e) => {
            if (!focusSearch) return;
            e.preventDefault();
            const box = (
              e.currentTarget as HTMLElement
            ).querySelector<HTMLInputElement>('input[type="search"]');
            box?.focus();
          }}
          onEscapeKeyDown={(e) => {
            // Esc in the search box clears it first, then leaves the box; the next Esc closes.
            if ((e.target as Element | null)?.matches?.('input[type="search"]'))
              e.preventDefault();
          }}
        >
          <Dialog.Title className={styles.srOnly}>Sidebar</Dialog.Title>
          {children}
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}

function MenuIcon() {
  return (
    <svg
      width="16"
      height="16"
      viewBox="0 0 16 16"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.5"
      strokeLinecap="round"
      aria-hidden="true"
    >
      <path d="M2.5 4h11M2.5 8h11M2.5 12h11" />
    </svg>
  );
}
