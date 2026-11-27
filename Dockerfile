# One image: the Vite app built and embedded into the Go binary, nothing else.

FROM node:24-slim AS web
WORKDIR /src
RUN corepack enable
COPY package.json pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY tsconfig.json vite.config.ts ./
COPY web ./web
RUN pnpm build

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
