FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o migrate ./cmd/migrate

FROM alpine:3.19

RUN apk --no-cache add ca-certificates curl

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /build/server .
COPY --from=builder /build/migrate .
COPY --from=builder /build/migrations ./migrations
COPY --from=builder /build/configs/config.yaml.example ./configs/config.yaml
COPY --from=builder /build/web ./web
COPY docker/entrypoint.sh .
RUN chmod +x /app/entrypoint.sh && chown -R appuser:appgroup /app

USER appuser

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD curl -f http://localhost:8080/health || exit 1

ENTRYPOINT ["/app/entrypoint.sh"]
