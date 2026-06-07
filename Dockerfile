FROM golang:1.25.0-bookworm AS builder

RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY go.mod go.sum ./
# Note: we will generate go.sum locally when compiling
RUN go mod download || true

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -o notification-service ./cmd/main.go

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /root/

COPY --from=builder /app/notification-service .
COPY .env .
COPY sql/init.sql ./sql/init.sql
EXPOSE 10001

CMD ["./notification-service"]
