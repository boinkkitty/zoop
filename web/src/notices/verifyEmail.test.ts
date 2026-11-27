import { describe, expect, it } from "vitest";
import { verifyEmailNotice } from "./verifyEmail";

const unverified = {
  email: "ada@example.com",
  emailVerified: false,
  dismissed: false,
  resend: "idle" as const,
};

describe("the verify email Notice", () => {
  it("asks an unverified user to check their inbox, offering Resend", () => {
    expect(verifyEmailNotice(unverified)).toEqual({
      text: "Check your inbox to verify ada@example.com.",
      action: "Resend",
    });
  });

  it("says Sent again once the email was resent, with nothing more to press", () => {
    expect(verifyEmailNotice({ ...unverified, resend: "sent" })).toEqual({
      text: "Check your inbox to verify ada@example.com.",
      action: null,
      status: "Sent again",
    });
  });

  it("offers Resend again if resending failed", () => {
    expect(verifyEmailNotice({ ...unverified, resend: "failed" })).toEqual({
      text: "Couldn't send it. Try again in a little while.",
      action: "Resend",
    });
  });

  it("is gone once dismissed", () => {
    expect(verifyEmailNotice({ ...unverified, dismissed: true })).toBeNull();
  });

  it("clears once the email is verified", () => {
    expect(
      verifyEmailNotice({ ...unverified, emailVerified: true }),
    ).toBeNull();
  });
});
