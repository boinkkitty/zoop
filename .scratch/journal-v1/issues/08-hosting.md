# Hosting and deployment

Type: grilling
Status: open
Blocked by:

## Question

Where do the Go API, PostgreSQL and the React frontend run for v1, and how are they deployed? Options include Fly.io, Railway, Render, or a VPS, with managed Postgres such as Neon or Supabase-Postgres. Constraints:
- the frontend and API must share a site, so the HttpOnly session cookie works
- low cost at launch
- backups
