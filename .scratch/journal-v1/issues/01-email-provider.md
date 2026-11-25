# Email provider for verification and password reset

Type: research
Status: resolved
Blocked by:

## Question

Which transactional email provider should Zoop use to send email-verification and password-reset emails from a Go backend: Resend, Postmark, Amazon SES, or another? Compare the free tier and price at small scale (under 10k emails/month), deliverability reputation, the Go SDK or a plain HTTP API, the domain setup required (SPF/DKIM/DMARC), and how much setup is needed for a solo developer. Recommend one.

Research: ../research/email-provider.md

## Answer

**Resend.** It has an official Go SDK (`resend-go/v4`), needs only 3 DNS records, and has no account review. Its free tier is 3,000 emails/month but capped at **100/day**; Pro is $20/month and removes the cap. Also publish DMARC, send from a dedicated subdomain, and wrap sending in a small Go `Mailer` interface so the provider can be swapped. Fallbacks: Postmark if deliverability matters more than cost, SES if cost matters most. Full findings: [research/email-provider.md](../research/email-provider.md).

## Answer

**Use Resend** via the official Go SDK (`github.com/resend/resend-go/v4`), behind a small Go `Mailer` interface so Postmark or SES can be swapped in later.

- The free tier is 3,000 emails/month with a **100/day cap**. Move to Pro ($20/mo, 50k) before daily volume gets near 100, or password resets will fail.
- Domain setup: DKIM, SPF (`TXT` + `MX` on a `send.` subdomain) and our own DMARC record. No account approval and no sandbox.
- Runners-up: Postmark (best published deliverability, $15/mo at 10k, no official Go SDK) and SES (about $1 at 10k, but the most setup: IAM, leaving the sandbox, handling bounces).

Full comparison and sources: [research/email-provider.md](../research/email-provider.md)
