# Build stage
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
COPY migrations/ ./migrations/
RUN go mod download
COPY . .
RUN go build -o server ./cmd/server

# Final image
FROM alpine:latest
WORKDIR /root/
RUN apk add curl
COPY --from=builder /app/server .
COPY --from=builder /app/migrations /app/migrations
EXPOSE 8080
ENTRYPOINT ["./server"]

