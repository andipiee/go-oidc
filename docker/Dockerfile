FROM golang:1.21-alpine AS builder

RUN apk add --no-cache git

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server ./cmd/server

FROM alpine:3.19

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /build/server .
COPY --from=builder /build/configs ./configs
COPY --from=builder /build/web ./web

EXPOSE 8080

CMD ["./server"]
