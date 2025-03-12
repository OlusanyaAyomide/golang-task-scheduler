# Build stage
ARG GO_VERSION=1.21
FROM golang:${GO_VERSION}-bookworm AS builder

WORKDIR /app

# Copy and download dependencies
COPY go.mod ./
RUN go mod download

# Copy source code and build
COPY . .
RUN go build -o /run-app .

# Deploy stage
FROM debian:bookworm

COPY --from=builder /run-app /usr/local/bin/
CMD ["run-app"]