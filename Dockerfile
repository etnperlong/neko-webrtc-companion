FROM --platform=$BUILDPLATFORM golang:1.25.0-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN apk add --no-cache git && \
    go mod download

COPY . .

ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG TARGETVARIANT
ENV CGO_ENABLED=0

RUN set -eux; \
    export GOOS="${TARGETOS}"; \
    export GOARCH="${TARGETARCH}"; \
    if [ "${TARGETARCH}" = "arm" ] && [ -n "${TARGETVARIANT}" ]; then export GOARM="${TARGETVARIANT#v}"; fi; \
    go build -o /app/neko-turn-refresh ./cmd/neko-turn-refresh

FROM gcr.io/distroless/static-debian12

COPY --from=builder /app/neko-turn-refresh /usr/local/bin/neko-turn-refresh

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/neko-turn-refresh"]
