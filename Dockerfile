# syntax=docker/dockerfile:1
# One image: the Vite app built and embedded into the Go binary, nothing else.

FROM node:24-slim AS web
WORKDIR /src
RUN corepack enable
COPY package.json pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY tsconfig.json vite.config.ts ./
COPY web ./web
# Sentry for the browser (all optional). The DSN is public; the auth token is a
# build secret, so it never lands in a layer, and only CI on main passes it.
ARG VITE_SENTRY_DSN VITE_SENTRY_RELEASE SENTRY_ORG SENTRY_PROJECT
RUN --mount=type=secret,id=sentry_auth_token,env=SENTRY_AUTH_TOKEN pnpm build \
 && find web/dist -name '*.map' -delete

FROM golang:1.27-alpine AS bin
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
COPY web ./web
COPY --from=web /src/web/dist ./web/dist
# Empty for production. CI's e2e image passes GO_TAGS=testclock to compile in the test clock.
ARG GO_TAGS=
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -tags "$GO_TAGS" -o /zoop ./cmd/zoop

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=bin /zoop /zoop
ENV PORT=8080
EXPOSE 8080
ENTRYPOINT ["/zoop"]
