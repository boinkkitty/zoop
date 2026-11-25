# Research: transactional email provider for Zoop

Ticket: `issues/01-email-provider.md`
Researched: 2026-11-25, from official pricing and docs pages (read via browser that day).

## Question

Which provider should Zoop's Go backend use to send email-verification and password-reset emails: Resend, Postmark, Amazon SES, or another? Compare free tier and small-scale price (under 10k emails/month), deliverability, Go SDK or HTTP API, DNS setup (SPF/DKIM/DMARC), and setup effort for a solo developer.

## Recommendation

**Use Resend.** Its free tier (3,000 emails/month, 100/day) covers verification and reset traffic for an early web-only v1 at $0. It has an official Go SDK, it needs no account approval or sandbox exit, and DNS setup is one domain's worth of copy-paste records. Upgrade to Pro ($20/mo, 50k emails) when daily volume gets near 100. **Postmark** is the strong runner-up: it has the best-documented deliverability focus, but it costs $15/mo once you pass 100 emails/month, it has no official Go SDK, and a person must approve your account first. **Amazon SES** is by far the cheapest per email (about $1-$1.60 for 10k), but the AWS account, IAM, sandbox exit request and bounce handling cost a solo developer more time than they save in money at this volume.

Put the provider behind a small Go interface (for example `Mailer.Send(ctx, to, subject, html, text)`). Switching to Postmark or SES later then only means writing a new adapter and changing DNS.

## Comparison

| | Resend | Postmark | Amazon SES |
|---|---|---|---|
| Free tier | 3,000/mo, **100/day cap**, 3 domains [1] | 100/mo, never expires, no overages [2] | No permanent free tier. New AWS accounts get up to $200 in Free Tier credits, usable within 12 months [3] |
| Cost at 10k/mo | $20/mo (Pro, includes 50k; extra emails $0.90/1k) [1] | $15/mo (Basic, includes 10k; extra emails $1.80/1k) [2] | ~$1.00 à la carte ($0.10/1k) or ~$1.60 on the Essentials plan ($0.16/1k), which new accounts default to from 2026-07-21 [3] |
| Go integration | Official SDK `github.com/resend/resend-go/v4` [4][5]; REST API | No official Go SDK (official libraries list has none) [6]; three community Go libs [7]; plain REST `POST https://api.postmarkapp.com/email` with `X-Postmark-Server-Token` header [8] | Official AWS SDK `github.com/aws/aws-sdk-go-v2/service/sesv2` (v1.80.0, published 2026-11-18) [9] |
| DNS setup | DKIM TXT plus SPF (`TXT` and `MX` on Return-Path subdomain, default `send.<domain>`); subdomain recommended [10][11] | DKIM TXT plus optional custom Return-Path CNAME; no SPF include needed [12][13] | Easy DKIM (2048-bit, CNAMEs; auto-created if you use Route 53) [14]; optional custom MAIL FROM needs MX + SPF TXT [15] |
| Gatekeeping | None documented beyond verifying the domain [10] | A person reviews each new account, in under 24h on weekdays; before approval you can only send to your own verified domains [16] | New accounts start in a sandbox (verified recipients only, 200 msgs/24h, per region) until a production-access request is approved [17] |
| Deliverability signals | "Pristine shared IPs", automatic suppression list, DKIM/SPF/DMARC on all plans [1] | Publishes median Time-to-Inbox (Gmail ~7.5s, Yahoo ~11s, Hotmail ~14s; internal tests) [18]; vetted shared IPs on all plans [2] | Virtual Deliverability Manager included in plans [3]; reputation is largely yours to manage (bounces/complaints) |
| Solo-dev setup effort | Low | Low to medium (approval wait, hand-rolled HTTP client) | High (AWS account, IAM keys, sandbox exit, bounce/complaint handling via SNS/MX) |

## Findings

### Pricing at small scale

- **Resend** [1]: Free is $0/mo for 3,000 emails/mo, but limited to **100 emails a day**, with 3 domains and 30-day retention. Pro is $20/mo for 50,000 emails with no daily limit and $0.90 per extra 1,000. Pay-as-you-go overage is only available on paid plans. DKIM/SPF/DMARC authentication and automatic suppression are on every tier.
- **Postmark** [2]: Free is $0 for "100 emails/month", "No overages allowed", aimed at "Testing your integration". Basic costs $15/mo at the 10k slider position ("For solo developers and smaller teams", extra $1.80/1k, 5 domains, unlimited emails/day, 45-day retention). Pro costs $16.50/mo and Platform $18/mo at 10k.
- **Amazon SES** [3]: Since 2026, SES has three plans: Essentials ($0.16/1k for 0-10M), Pro ($0.22/1k + $105/mo) and Enterprise ($0.23/1k + $500/mo). "New SES accounts ... will start on the Essentials plan beginning July 21, 2026. You can upgrade or switch to à-la-carte pricing at any time." À la carte outbound is $0.10 per 1,000 plus $0.12/GB of attachments. The old permanent free tier is gone; the current offer is "up to $200 in AWS Free Tier credits", to be used within 12 months of account creation.

At 10k/month: SES ~$1-1.60, Postmark $15, Resend $20. At Zoop's likely early volume (well under 3,000/month and under 100/day), Resend costs $0, while Postmark costs $15 as soon as you send more than 100 a month.

### Deliverability

No independent, primary-source benchmark compares these three. Every claim below comes from a vendor:
- Postmark is the only one that publishes a Time-to-Inbox page. Its "internal performance testing data" shows medians of about 7.5s for Gmail through about 22s for Apple, and it says the same delivery quality applies on free plans [18]. Its manual account approval exists to protect shared-IP reputation [16].
- Resend advertises "pristine shared IPs", automatic suppression, and deliverability insights on all plans [1]. It recommends sending from a subdomain to isolate reputation [10].
- On SES, much more of the reputation work falls on you. The sandbox and production-access review exist for that reason [17].

For low-volume auth email, any of the three will land in the inbox **if DKIM and DMARC are set up correctly**. Gmail requires SPF or DKIM from all senders, and SPF, DKIM and DMARC from senders of 5,000+ messages/day [19].

### Go integration

- **Resend**: official SDK, `go get github.com/resend/resend-go/v4` [4]. Sending is `resend.NewClient(key)` then `client.Emails.Send(&resend.SendEmailRequest{...})` [4]. It returns typed errors: `*resend.RateLimitError` for HTTP 429 and `*resend.ResendError` otherwise [5]. The docs include Chi and Gin example apps [4].
- **Postmark**: official libraries cover TypeScript, Python, Ruby/Rails, .NET, Java, PHP, Node.js and others, but not Go [6]. The community page lists three Go libraries [7]. The REST API is simple enough to call with `net/http` directly: a JSON POST to `https://api.postmarkapp.com/email` with an `X-Postmark-Server-Token` header [8].
- **SES**: AWS SDK for Go v2 `sesv2` package, actively maintained (v1.80.0 on 2026-11-18) [9]. It brings the AWS credential chain and IAM setup with it.

### Domain setup (SPF / DKIM / DMARC)

- **Resend**: add a domain, ideally a subdomain such as `mail.zoop.app`, then paste the DKIM and SPF records (`TXT` and `MX`, or `CNAME`) it generates. Return-Path defaults to `send.<domain>` [11]. "If you have a verified domain with Resend, it means you are already passing SPF and DKIM"; you then add a `_dmarc` TXT record yourself [20].
- **Postmark**: add a DKIM TXT record, which verifies within 48h or immediately with the Verify button, plus an optional custom Return-Path CNAME [12]. "It is no longer required to include Postmark in your own custom SPF record." Use `aspf=r` in DMARC if you want SPF alignment [13].
- **SES**: Easy DKIM adds 2048-bit signing through DNS records, which SES creates for you on Route 53 [14]. By default, MAIL FROM is an `amazonses.com` subdomain, so SPF passes but does not align with your domain. For SPF-based DMARC alignment, set up a custom MAIL FROM, which needs an MX record and an SPF TXT record [15]. DKIM alone is enough for DMARC.

For any of them, also publish a DMARC record (start at `v=DMARC1; p=none; rua=...`, then tighten).

### Setup effort for a solo developer

- **Resend**: sign up, verify the domain, create an API key, `go get`. Nothing documented has to wait on a person.
- **Postmark**: sign up, verify the domain, then **request approval**, which takes under 24h on weekdays. Until then you can send only to your own verified domains or to `test@blackhole.postmarkapp.com` [16]. You also write a roughly 40-line HTTP client or pick a community library.
- **SES**: create an AWS account, an IAM user or role with SES permissions, verify the domain identity, and **request production access** (sandbox limits: verified recipients only, 200 messages/24h, per region) [17]. You choose a pricing plan, set up the AWS SDK credential chain, and ideally wire bounce and complaint notifications.

## Risks and open points

- **Resend's 100/day free cap is a hard ceiling for password resets.** If a launch spike exceeds it, resets fail. Watch the volume and move to Pro ($20) before launch if sign-ups could pass about 100/day.
- Resend is younger than Postmark and SES. A thin `Mailer` interface keeps switching cheap.
- Deliverability claims come from the vendors themselves. Before launch, verify with real sends to Gmail, Outlook and iCloud test accounts (check headers for `dkim=pass` and `dmarc=pass`).
- Postmark is now part of ActiveCampaign (site footer), so its pricing and policies may change.

## Sources

1. Resend pricing: https://resend.com/pricing
2. Postmark pricing: https://postmarkapp.com/pricing
3. Amazon SES pricing: https://aws.amazon.com/ses/pricing/
4. Resend, Send emails with Go: https://resend.com/docs/send-with-go
5. resend-go repository (README): https://github.com/resend/resend-go
6. Postmark official libraries: https://postmarkapp.com/developer/integration/official-libraries
7. Postmark community libraries: https://postmarkapp.com/developer/integration/community-libraries
8. Postmark Email API: https://postmarkapp.com/developer/api/email-api
9. AWS SDK for Go v2, sesv2: https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/sesv2
10. Resend, Verified Domains: https://resend.com/docs/dashboard/domains/introduction
11. Resend, Add and verify a domain: https://resend.com/docs/add-a-domain
12. Postmark, How do I verify a domain: https://postmarkapp.com/support/article/how-do-i-verify-a-domain
13. Postmark, How do I set up SPF for Postmark: https://postmarkapp.com/support/article/how-do-i-set-up-spf-for-postmark
14. Amazon SES, Easy DKIM: https://docs.aws.amazon.com/ses/latest/dg/send-email-authentication-dkim-easy.html
15. Amazon SES, Custom MAIL FROM domain: https://docs.aws.amazon.com/ses/latest/dg/mail-from.html
16. Postmark, Account approval process: https://postmarkapp.com/support/article/1084-how-does-the-account-approval-process-work
17. Amazon SES, Request production access: https://docs.aws.amazon.com/ses/latest/dg/request-production-access.html
18. Postmark Time to Inbox: https://tti.postmarkapp.com/
19. Google, Email sender guidelines: https://support.google.com/mail/answer/81126
20. Resend, Implementing DMARC: https://resend.com/docs/dashboard/domains/dmarc
