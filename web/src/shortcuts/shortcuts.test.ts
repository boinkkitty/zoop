import { describe, expect, it } from "vitest";
import { isMac, isSheetShortcut, keyLabels } from "./shortcuts";

const key = (init: Partial<KeyboardEvent>) =>
  ({
    key: "/",
    metaKey: false,
    ctrlKey: false,
    altKey: false,
    shiftKey: false,
    ...init,
  }) as KeyboardEvent;

describe("shortcut keys", () => {
  it("recognises Macs the way ProseMirror does", () => {
    expect(isMac("MacIntel")).toBe(true);
    expect(isMac("iPad")).toBe(true);
    expect(isMac("Win32")).toBe(false);
    expect(isMac("Linux x86_64")).toBe(false);
  });

  it("shows ⌘ symbols on a Mac and Ctrl elsewhere", () => {
    expect(keyLabels(["Mod", "Shift", "H"], true)).toEqual(["⌘", "⇧", "H"]);
    expect(keyLabels(["Mod", "Enter"], true)).toEqual(["⌘", "↩"]);
    expect(keyLabels(["Mod", "Shift", "H"], false)).toEqual([
      "Ctrl",
      "Shift",
      "H",
    ]);
  });

  it("opens the sheet with ⌘/ on a Mac and Ctrl+/ elsewhere", () => {
    expect(isSheetShortcut(key({ metaKey: true }), true)).toBe(true);
    expect(isSheetShortcut(key({ ctrlKey: true }), true)).toBe(false);
    expect(isSheetShortcut(key({ ctrlKey: true }), false)).toBe(true);
    expect(isSheetShortcut(key({ metaKey: true }), false)).toBe(false);
    expect(isSheetShortcut(key({ ctrlKey: true, shiftKey: true }), false)).toBe(
      true,
    );
    expect(isSheetShortcut(key({ ctrlKey: true, altKey: true }), false)).toBe(
      false,
    );
    expect(isSheetShortcut(key({ key: "?", ctrlKey: true }), false)).toBe(
      false,
    );
    expect(isSheetShortcut(key({}), false)).toBe(false);
  });
});
