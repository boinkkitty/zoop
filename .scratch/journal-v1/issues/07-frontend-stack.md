# Frontend stack and editor

Type: grilling
Status: resolved
Assignee: boinkkitty
Blocked by: 05

## Question

What is the React setup for v1?
- Build tool / framework: Vite SPA vs Next.js, given a separate Go API
- Routing
- Server-state / data-fetching library
- Styling approach
- The editor component: a plain textarea can't show a bold first line, so the choice is between contenteditable, a textarea with an overlay, or a minimal editor library such as Lexical or TipTap, while keeping the content plain text

Note (2026-11-25): Pages are no longer plain text. The editor choice must support whatever Formatting on Pages decides, which makes TipTap or Lexical much more likely than a textarea.

Update (2026-11-25): **The editor is TipTap.** [Formatting on Pages](09-page-formatting.md) decided this, along with the formatting set, the keyboard shortcuts, the rule that the first block of My Journal is always a plain paragraph, and the paste behaviour. This ticket only needs to decide how TipTap is set up, not which editor to use.

## Answer

Settled with the user on 2026-11-25 over three grilling rounds.

- **Build**: Vite + React + TypeScript (strict). The built `dist/` is embedded in the Go binary with `go:embed`. In dev, Vite's proxy forwards `/api` to the Go server so cookies stay same-origin. Go serves `index.html` for every path that isn't `/api`. Next.js was ruled out: there's no Node server in production, so its server-first model buys nothing.
- **Routing**: TanStack Router, with routes defined in code (no file-based code generation). Each Day+Notebook has its own URL, for example `/day/2026-11-25/my-journal`, so it can be linked and works with back/forward. The Page route is **lazy-loaded** so the auth screens don't load TipTap.
- **Server state**: TanStack Query. The rail uses `useInfiniteQuery` with `getNextPageParam` / `getPreviousPageParam` to page in both directions from any anchor date. One small hand-typed `api.ts` fetch wrapper mirrors [API contract and autosave](04-api-contract-and-autosave.md), with no generated client.
- **Client state**: React state + the URL only. No store library (Zustand, Jotai) in v1.
- **Styling**: **CSS Modules** (one `.module.css` per component) + one `tokens.css` of CSS custom properties (colours, radii, `--ease-spring: cubic-bezier(.32,1.25,.5,1)`, `--hover-delay: 150ms`) + one plain `page.css` for the editor content, which TipTap renders and which can't carry component classes. **No component library**, because it would drag the look toward generic SaaS. **Radix** primitives (unstyled) are used only for the ⌘/ shortcut-sheet dialog and the narrow-window slide-over drawer. Tailwind was considered and rejected: the Page has to be plain CSS anyway, and the hand-tuned values would mostly be `[arbitrary]` classes.
- **Fonts**: the system font stack (`-apple-system, BlinkMacSystemFont, "Segoe UI", …`). SF Pro's licence doesn't allow self-hosting it on the web.
- **Animation**: plain CSS transitions, no animation library. The Island push-down animates `grid-template-rows: auto 0fr → auto 1fr` with `var(--ease-spring)`, and collapsing is instant.
- **Editor (TipTap v3)**:
  - An **explicit extension list** in one `pageSchema.ts` (Document, Paragraph, Text, Bold, Italic, Underline, Highlight, BulletList, OrderedList, ListItem, TaskList, TaskItem, Blockquote, undo history, …), never StarterKit, so library updates can't quietly add nodes the Go validator rejects.
  - Keyboard shortcuts are rebound to match the table in [Formatting on Pages](09-page-formatting.md) exactly. TipTap's Markdown input rules are **off**.
  - **Two document grammars**: *My Journal* is `content: 'paragraph block*'`, so ProseMirror itself refuses a list or quote as the first block, and a block shortcut on line 1 does nothing. *I Learned* is `block+`.
  - The Title's bold is **CSS only** (`.my-journal p:first-child`), not a stored mark. The JSON stays a plain paragraph, and deleting line 1 makes the next line the Title. ⌘B inside the Title is allowed and visually redundant, with no special handling.
- **Keeping client and server schemas in step**: a shared `schema-fixtures/valid/*.json` and `schema-fixtures/invalid/*.json` folder, loaded by both Vitest and `go test`. `schemaVersion` is a constant in both codebases, and a fixture check asserts they match.
- **Tooling**: pnpm, Biome (lint + format), Vitest + Testing Library, and a few Playwright end-to-end smoke tests (login → type → autosave → reload). The frontend lives in `web/` in the same repo as the Go module, and GitHub Actions builds it before `go build`.

No ADR: each choice is easy to reverse or unsurprising. No new glossary terms.
