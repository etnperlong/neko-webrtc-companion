FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN apk add --no-cache git && \
  go mod download

COPY . .

RUN set -eux; \
  CGO_ENABLED=0 go build -o /app/neko-turn-refresh ./cmd/neko-turn-refresh

FROM gcr.io/distroless/static-debian13

COPY --from=builder /app/neko-turn-refresh /usr/local/bin/neko-turn-refresh

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/neko-turn-refresh"]
