// The verify email Notice: what it says for an account whose email isn't verified yet.

export type ResendState = "idle" | "sending" | "sent" | "failed";

export type VerifyEmailNotice = {
  text: string;
  action: "Resend" | null;
  status?: string;
};

export function verifyEmailNotice(s: {
  email: string;
  emailVerified: boolean;
  dismissed: boolean;
  resend: ResendState;
}): VerifyEmailNotice | null {
  if (s.emailVerified || s.dismissed) return null;
  const text = `Check your inbox to verify ${s.email}.`;
  switch (s.resend) {
    case "sending":
      return { text, action: null, status: "Sending…" };
    case "sent":
      return { text, action: null, status: "Sent again" };
    case "failed":
      return {
        text: "Couldn't send it. Try again in a little while.",
        action: "Resend",
      };
    default:
      return { text, action: "Resend" };
  }
}
