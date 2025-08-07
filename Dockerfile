FROM golang:1.24-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY migrations/ ./migrations/
COPY integrations/ ./integrations/

COPY . .
RUN go mod tidy
RUN go build -o server ./cmd/server

FROM alpine:latest
WORKDIR /app

RUN apk add --no-cache curl ca-certificates

COPY --from=builder /app/server .
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /app/integrations ./integrations

EXPOSE 8080
ENTRYPOINT ["./server"]

