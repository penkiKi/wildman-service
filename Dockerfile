# syntax=docker/dockerfile:1

FROM oven/bun:1.3.5-debian AS frontend-build
WORKDIR /src/frontend
COPY frontend/package.json frontend/bun.lock ./
RUN bun install --frozen-lockfile
COPY frontend/ ./
RUN bun run build

FROM --platform=$BUILDPLATFORM golang:1.26-bookworm AS backend-build
ARG TARGETOS
ARG TARGETARCH
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
COPY --from=frontend-build /src/web/dist ./web/dist
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /out/wildman-service ./cmd/server
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /out/wildman-worker ./cmd/worker

FROM debian:bookworm-slim AS runtime
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates tzdata \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd --system --gid 10001 wildman \
    && useradd --system --uid 10001 --gid wildman --no-create-home --home-dir /nonexistent --shell /usr/sbin/nologin wildman

COPY --from=backend-build /out/wildman-service /usr/local/bin/wildman-service
COPY --from=backend-build /out/wildman-worker /usr/local/bin/wildman-worker

ENV WILDMAN_HOST=0.0.0.0 \
    WILDMAN_PORT=8080 \
    WILDMAN_ENV=production

USER wildman
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/wildman-service"]
