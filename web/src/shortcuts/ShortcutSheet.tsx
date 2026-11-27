import * as Dialog from "@radix-ui/react-dialog";
import { useEffect, useState } from "react";
import styles from "./ShortcutSheet.module.css";
import { isSheetShortcut, keyLabels, shortcutGroups } from "./shortcuts";

// The shortcut sheet, opened (and closed again) by ⌘/ from anywhere on the signed-in app.
export function ShortcutSheet() {
  const [open, setOpen] = useState(false);
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (!isSheetShortcut(e)) return;
      e.preventDefault();
      setOpen((o) => !o);
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);

  return (
    <Dialog.Root open={open} onOpenChange={setOpen}>
      <Dialog.Portal>
        <Dialog.Overlay className={styles.overlay} />
        <Dialog.Content className={styles.sheet}>
          <Dialog.Title className={styles.title}>
            Keyboard shortcuts
          </Dialog.Title>
          <Dialog.Description className={styles.description}>
            Formatting is by keyboard only. Typed Markdown stays as you typed
            it.
          </Dialog.Description>
          <div className={styles.groups}>
            {shortcutGroups.map((group) => (
              <section key={group.title}>
                <h3 className={styles.group}>{group.title}</h3>
                <dl className={styles.list}>
                  {group.shortcuts.map((s) => (
                    <div key={s.label} className={styles.row}>
                      <dt>{s.label}</dt>
                      <dd>
                        {keyLabels(s.keys).map((k) => (
                          <kbd key={k}>{k}</kbd>
                        ))}
                      </dd>
                    </div>
                  ))}
                </dl>
              </section>
            ))}
          </div>
          <Dialog.Close className={styles.done}>Done</Dialog.Close>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
