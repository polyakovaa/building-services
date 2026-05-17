FROM golang:1.25-alpine AS builder

ENV GOTOOLCHAIN=auto

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY notification-service/ ./notification-service/
COPY gen/ ./gen/

COPY notification-service/go.mod notification-service/go.sum ./notification-service/
WORKDIR /app/notification-service
RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o notification-service ./cmd/notification

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/notification-service/notification-service .
COPY notification-service/config/config.yaml ./config.yaml

EXPOSE 50054
CMD ["./notification-service"]
