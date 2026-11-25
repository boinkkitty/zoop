# Sheet banners and first run

Type: prototype
Status: resolved
Assignee: boinkkitty
Blocked by:

## Question

How do the quiet notices on the sheet look and stack, and what does a brand-new user see on Day 1?
- Banners from [API contract and autosave](04-api-contract-and-autosave.md): verify your email; timezone "Switch / Keep"; "This Page changed on another device — Keep this version / Use the other"; "Not saved — retrying"; "Zoop was updated — reload"; and the read-only state after the Grace Window locks at 02:00.
- Placement and style that fit the Apple-minimal sheet ([Journal page look and feel](05-journal-page-prototype.md)), how several stack, and which can be dismissed.
- First run: the empty Day 1 sheet, any hint about the two Notebooks (*My Journal*, *I Learned*) and the bold Title, a pointer to ⌘/, and how the rail looks with a single Day.

## Answer

Settled with the user on 2026-11-25 in [sheet-banners-PROTOTYPE.html](../prototypes/sheet-banners-PROTOTYPE.html). The winners are **B (Island) for Notices** and **F3 (coach marks) for Day 1**. A (strip on the sheet), C (header line), D (split by scope), F1 (placeholders only) and F2 (welcome note) lost and stay in the file for reference.

**Notices: one notice pill at the top of the sheet area**
- A black pill, centred over the sheet's top edge in the Island style (Spring motion, drops in when the first Notice appears). It shows the **most urgent Notice** with its actions, plus a "+N" count when there are more. Hovering (after the rail's 150ms delay) or focusing it expands the pill downward into the whole stack, one row per Notice with its own actions. Nothing shows when there are no Notices.
- In a narrow window the pill centres over the sheet the same way. It does not move into the drawer.
- It is called the **notice pill**, never an Island: in the glossary, an Island is a Day in the rail.

**The six Notices, most urgent first**

| Notice | Text | Actions | Clears when |
|---|---|---|---|
| Update | "Zoop was updated. Reload to keep saving. What you wrote is kept on this device." | Reload | Reloading. Saving is blocked until then (`client_outdated`). |
| Changed elsewhere | "This Page changed on another device." | Keep this version · Use the other | You pick one. Shown only when both sides have edits. |
| Not saved | "Not saved, retrying in Ns. Your words are kept on this device." | Try now | A save succeeds. The countdown follows the backoff. |
| Day closed | "Yesterday closed at 02:00. Everything you wrote is saved." | Go to today · × | Dismissed, per Day. Only on yesterday's Page, which turns read-only with the "Read only" pill. |
| Timezone | "Looks like you're on Tokyo time. Start your Days on Tokyo time from tomorrow?" | Switch · Keep London | Answered (Keep is remembered per detected zone, as before). |
| Verify email | "Check your inbox to verify you@…" → "Sent again" after Resend | Resend · × | Verified, or × for this browser session. It returns on the next load until verified. Settings always shows the state, per [Sidebar nav and Settings](13-sidebar-nav-and-settings.md). |

Only Day closed and Verify email can be dismissed. The other four clear only when their cause is gone or answered.

**First run (Day 1): coach marks**
- The sheet shows grey placeholders: "Title" and "Write about your day…" on My Journal, "What did you learn today?" on I Learned.
- **Three coach marks**, one at a time, as dark callouts in the notice pill's style with "N of 3 · Skip · Next / Got it":
  1. Today's Page: "Just start writing. It saves as you type. The first line becomes the Day's Title and turns bold."
  2. The Notebooks index: My Journal for your day, I Learned for one thing you found out, each with its own Page every Day.
  3. The rail: each Day gets an Island with its Title. You can write until 02:00, then the Day closes. ⌘/ shows formatting shortcuts.
- Typing, Skip or Got it ends them for good. Shown only while the user has **no Written Day** (from `GET /days/summary`) and they haven't been ended in this browser (`localStorage`). So an existing user on a new browser never sees them. In a narrow window, step 3 points at the drawer button instead of the rail.
- **Rail with one Day**: the "November 2026" header, one grey "Today" Island, then "You started here · today", with a faint line under it until the first Written Day: "Every Day gets an Island here. Its Title is the first line you write." The Island turns black and shows the Title as soon as the user types.
- On Day 1 the verify-email Notice usually shows too (for email signups). The pill and the coach marks don't overlap: the first coach mark points at the Page below the date.

**What this asks of other tickets:** nothing new. Everything uses existing fields (`GET /me` `emailVerified`, `GET /days/summary`, `locksAt`, and the 409 codes from [API contract and autosave](04-api-contract-and-autosave.md)).

No ADR: easy to reverse. New glossary term: **Notice**.
