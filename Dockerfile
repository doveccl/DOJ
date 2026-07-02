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
  CGO_ENABLED=0 go build -o /out/doj . && \
  CGO_ENABLED=0 go build -o /out/doj-jplag ./cmd/jplag

FROM eclipse-temurin:25-jre-alpine AS jplag

ARG JPLAG_VERSION=6.3.0
ADD "https://github.com/jplag/JPlag/releases/download/v${JPLAG_VERSION}/jplag-${JPLAG_VERSION}-jar-with-dependencies.jar" /app/jplag.jar

COPY --from=build /out/doj-jplag /usr/local/bin/doj-jplag

WORKDIR /app
EXPOSE 7979
CMD ["doj-jplag"]

FROM alpine:3 AS app

WORKDIR /app

RUN apk add --no-cache ca-certificates postgresql-client tzdata

COPY --from=build /out/doj /usr/local/bin/doj
COPY --from=web /src/dist /app/dist

EXPOSE 7974
CMD ["doj", "server"]
