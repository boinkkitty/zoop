// The keyboard shortcuts as the shortcut sheet lists them. The editor binds them in
// web/src/page (schema.ts and formatting.ts); Mod is ⌘ on a Mac and Ctrl elsewhere.

export type Shortcut = { label: string; keys: string[] };
export type ShortcutGroup = { title: string; shortcuts: Shortcut[] };

export const shortcutGroups: ShortcutGroup[] = [
  {
    title: "Text",
    shortcuts: [
      { label: "Bold", keys: ["Mod", "B"] },
      { label: "Italic", keys: ["Mod", "I"] },
      { label: "Underline", keys: ["Mod", "U"] },
      { label: "Highlight", keys: ["Mod", "Shift", "H"] },
    ],
  },
  {
    title: "Blocks",
    shortcuts: [
      { label: "Bulleted list", keys: ["Mod", "Shift", "8"] },
      { label: "Numbered list", keys: ["Mod", "Shift", "7"] },
      { label: "Checklist", keys: ["Mod", "Shift", "L"] },
      { label: "Quote", keys: ["Mod", "Shift", "9"] },
    ],
  },
  {
    title: "Lists",
    shortcuts: [
      { label: "Nest (up to 3 levels)", keys: ["Tab"] },
      { label: "Un-nest", keys: ["Shift", "Tab"] },
      { label: "Tick or untick", keys: ["Mod", "Enter"] },
      { label: "End the list (on an empty item)", keys: ["Enter"] },
    ],
  },
  {
    title: "Other",
    shortcuts: [
      { label: "Paste as plain text", keys: ["Mod", "Shift", "V"] },
      { label: "Show these shortcuts", keys: ["Mod", "/"] },
      { label: "Open Settings", keys: ["Mod", ","] },
    ],
  },
];

// isMac matches ProseMirror's own test, so the sheet shows the keys the editor listens for.
export const isMac = (platform = globalThis.navigator?.platform ?? "") =>
  /Mac|iP(hone|[oa]d)/.test(platform);

const macKeys: Record<string, string> = {
  Mod: "⌘",
  Shift: "⇧",
  Enter: "↩",
  Tab: "⇥",
};
const otherKeys: Record<string, string> = { Mod: "Ctrl" };

// keyLabels names each key as this platform's keyboard shows it.
export function keyLabels(keys: string[], mac = isMac()): string[] {
  const names = mac ? macKeys : otherKeys;
  return keys.map((k) => names[k] ?? k);
}

// isSheetShortcut is ⌘/ (Ctrl+/ off a Mac). Shift is allowed: some layouts need it for "/".
export function isSheetShortcut(e: KeyboardEvent, mac = isMac()): boolean {
  const mod = mac ? e.metaKey && !e.ctrlKey : e.ctrlKey && !e.metaKey;
  return mod && !e.altKey && e.key === "/";
}

// isSettingsShortcut is ⌘, (Ctrl+, off a Mac).
export function isSettingsShortcut(e: KeyboardEvent, mac = isMac()): boolean {
  const mod = mac ? e.metaKey && !e.ctrlKey : e.ctrlKey && !e.metaKey;
  return mod && !e.altKey && !e.shiftKey && e.key === ",";
}
