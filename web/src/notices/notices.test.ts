import { describe, expect, it } from "vitest";
import { canDismiss, type NoticeId, noticeStack } from "./notices";

const n = (id: NoticeId) => ({ id });

describe("the Notice stack", () => {
  it("puts the most urgent Notice first", () => {
    const shuffled = [
      n("verifyEmail"),
      n("dayClosed"),
      n("update"),
      n("timezone"),
      n("notSaved"),
      n("conflict"),
    ];
    expect(noticeStack(shuffled, new Set()).map((x) => x.id)).toEqual([
      "update",
      "conflict",
      "notSaved",
      "dayClosed",
      "timezone",
      "verifyEmail",
    ]);
  });

  it("lets only Day closed and verify email be dismissed", () => {
    expect(
      (
        [
          "update",
          "conflict",
          "notSaved",
          "dayClosed",
          "timezone",
          "verifyEmail",
        ] as const
      ).filter(canDismiss),
    ).toEqual(["dayClosed", "verifyEmail"]);
  });

  it("hides dismissed Notices, but never one that can't be dismissed", () => {
    const active = [n("notSaved"), n("dayClosed"), n("verifyEmail")];
    const dismissed = new Set<NoticeId>(["notSaved", "dayClosed"]);
    expect(noticeStack(active, dismissed).map((x) => x.id)).toEqual([
      "notSaved",
      "verifyEmail",
    ]);
  });

  it("is empty when nothing needs saying", () => {
    expect(noticeStack([], new Set())).toEqual([]);
  });
});
