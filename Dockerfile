# syntax=docker/dockerfile:1.7

FROM node:24-alpine AS web

WORKDIR /src
RUN corepack enable
COPY package.json pnpm-lock.yaml ./
RUN --mount=type=cache,target=/root/.local/share/pnpm/store pnpm install --frozen-lockfile --prefer-offline

COPY api ./api
COPY web ./web
COPY index.html ./
COPY tsconfig.json vite.config.ts ./
RUN pnpm build

FROM golang:1.26-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
  CGO_ENABLED=0 go build -o /out/doj-server ./cmd/server.go && \
  CGO_ENABLED=0 go build -tags judger -o /out/doj-judger ./cmd/judger.go && \
  CGO_ENABLED=0 go build -tags runner -o /out/doj-runner ./cmd/runner.go

FROM alpine:3.23

RUN apk add --no-cache ca-certificates postgresql-client \
  && adduser -D -h /var/lib/doj doj \
  && mkdir -p /app /var/lib/doj \
  && chown -R doj:doj /app /var/lib/doj

COPY --from=build /out/doj-server /usr/local/bin/doj-server
COPY --from=build /out/doj-judger /usr/local/bin/doj-judger
COPY --from=build /out/doj-runner /usr/local/bin/doj-runner
COPY --from=web --chown=doj:doj /src/dist /app/dist

WORKDIR /app
USER doj
EXPOSE 7974
CMD ["doj-server"]
