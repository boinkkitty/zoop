# Frontend stack and editor

Type: grilling
Status: open
Blocked by: 05, 09

## Question

What is the React setup for v1?
- Build tool / framework: Vite SPA vs Next.js, given a separate Go API
- Routing
- Server-state / data-fetching library
- Styling approach
- The editor component: a plain textarea can't show a bold first line, so the choice is between contenteditable, a textarea with an overlay, or a minimal editor library such as Lexical or TipTap, while keeping the content plain text

Note (2026-11-25): Pages are no longer plain text. The editor choice must support whatever Formatting on Pages decides, which makes TipTap or Lexical much more likely than a textarea.
