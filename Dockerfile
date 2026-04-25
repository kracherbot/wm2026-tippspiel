FROM golang:1.24-bookworm AS builder

WORKDIR /app

COPY go.mod ./
COPY *.go ./
COPY templates/ ./templates/
COPY static/ ./static/

RUN go mod tidy && CGO_ENABLED=1 GOOS=linux go build -o tippspiel .

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    libsqlite3-0 \
    tzdata \
    && rm -rf /var/lib/apt/lists/*

ENV TZ=Europe/Zurich

WORKDIR /app

COPY --from=builder /app/tippspiel .

RUN mkdir -p /data

EXPOSE 8080

CMD ["./tippspiel"]