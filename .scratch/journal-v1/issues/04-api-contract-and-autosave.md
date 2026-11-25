# API contract and autosave

Type: grilling
Status: open
Blocked by: 03

## Question

What are the v1 HTTP endpoints, their request/response shapes, and how does autosave behave? Cover:
- auth flows
- fetching a range of Days for the rail, including Titles and hover glimpses. Per [Island rail and hover expansion](06-island-rail-prototype.md), a page must be fetchable from **any anchor date in either direction** (a jump loads only the target month, then newer Days above and older Days below), and the rail's scrubber needs a **per-month written-Day summary across the whole history**
- fetching and saving one Page
- search
- autosave debounce
- conflicts when the same Page is open in two browser tabs or on two devices
- how the server decides "today" and enforces the Grace Window lock for Free Users at 02:00
