# Sidebar nav and Settings

Type: grilling
Status: resolved
Assignee: boinkkitty
Blocked by: 12

## Question

What are the 2–3 buttons above search in the left sidebar (placeholders Home / Filler 1 / Filler 2 in [Journal page look and feel](05-journal-page-prototype.md)), and what does Settings contain?
- Which buttons exist (for example Today, Settings/Account, a shortcut-sheet button), and their icons or labels.
- Where the ⌘/ keyboard-shortcut sheet opens from, besides the shortcut itself.
- Settings contents: timezone (with the pending switch from [API contract and autosave](04-api-contract-and-autosave.md)), email and verification state, password set/change via the forgot-password flow, "Deactivate account" (dialog and type-your-email confirmation from [Privacy and data ownership](12-privacy-and-data-ownership.md); there is no export in v1), a link to the privacy page, sign out.
- Settings as a route, a dialog, or a panel over the sheet, and how it behaves in the narrow-window drawer.

## Answer

Grilled 2026-11-25.

- **Sidebar nav: two rows above search**, Today and Settings, with 16px line icons (a sun for Today, a gear for Settings) in the prototype's row style. These replace the Home / Filler 1 / Filler 2 placeholders from [Journal page look and feel](05-journal-page-prototype.md). There is no Shortcuts row.
- **Today** opens today's Page (the server's today), jumps the rail to the top, and closes the drawer in a narrow window. The row is active while today's Page is open. During the Grace Window it still means today, not yesterday. It has no keyboard shortcut (⌘T belongs to the browser).
- **Settings is a centred Radix dialog** over a dimmed sheet, with no URL of its own. Esc or clicking outside closes it. It opens from the nav row or **⌘,**. In a narrow window the drawer closes first, and the dialog goes almost full-screen. It works the same for every signed-in user, unverified or past the Grace Window.
- **The ⌘/ shortcut sheet** opens from ⌘/ or from the first row in Settings. Clicking that row closes Settings and opens the sheet, so the two never stack. This updates [Formatting on Pages](09-page-formatting.md), which assumed a sidebar item.
- **Settings contents**, in one scrolling column, in this order:
  1. **Keyboard shortcuts · ⌘/**: opens the shortcut sheet, as above.
  2. **Email**: the address, plus "Verified ✓" or "Not verified · Resend email" (`POST /auth/verify/resend`).
  3. **Sign-in**: "Password · Change" or "Password · Set a password". Both send the forgot-password email (`POST /auth/password/forgot`) and show "Check your email". There is no in-app current/new password form. The reset signs out every device, including this one. "Google · Connected" or "Not connected" is display only, with the hint *"Sign in with Google once to connect it."* There are no connect or unlink buttons in v1.
  4. **Timezone**: the current zone plus a searchable picker of IANA zones. Choosing a zone records a pending switch for **tomorrow**, the same rule as the banner, so a Day is never skipped or repeated. A pending switch shows as *"Switching to Tokyo on Thu 26 Nov · Cancel"*. Cancel, or choosing the current zone, sends `PATCH /me {timezone: <current>}`, which clears the pending pair.
  5. **Privacy**: a link to the static privacy page, opening in a new tab.
  6. **Sign out**: this device only, with no confirmation, then the login screen. There is no "sign out everywhere" (a password reset does that).
  7. **Deactivate account**: last, in red, opening the type-your-email dialog from [Privacy and data ownership](12-privacy-and-data-ownership.md).

  No plan or "Free" label is shown, since v1 has no billing.

**What this asks of other tickets:**
- [API contract and autosave](04-api-contract-and-autosave.md): `GET /me` gains `signInMethods` (a subset of `["password","google"]`). `PATCH /me` with the current effective zone clears a pending switch.

No ADR: easy to reverse. No new glossary terms.
