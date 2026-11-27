import * as Dialog from "@radix-ui/react-dialog";
import styles from "./ShortcutSheet.module.css";
import { keyLabels, shortcutGroups } from "./shortcuts";

// The shortcut sheet, opened by ⌘/ or from Settings. The journal owns whether it's open,
// so it and Settings never stack.
export function ShortcutSheet({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
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
