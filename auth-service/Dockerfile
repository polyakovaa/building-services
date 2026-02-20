FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY auth-service/ ./auth-service/
COPY gen/ ./gen/ 

COPY auth-service/go.mod auth-service/go.sum ./auth-service/
WORKDIR /app/auth-service
RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o auth-service ./cmd/auth


FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/auth-service .
COPY auth-service/config/config.yaml ./config.yaml

EXPOSE 50051
CMD ["./auth-service"]