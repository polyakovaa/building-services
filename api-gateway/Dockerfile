FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY api-gateway/go.mod api-gateway/go.sum ./api-gateway/
COPY go.mod go.sum ./
COPY gen/ ./gen/
WORKDIR /app/api-gateway
RUN go mod download
COPY api-gateway/ ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o api-gateway ./cmd


FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/api-gateway/api-gateway .
COPY --from=builder /app/api-gateway/frontend ./frontend
COPY api-gateway/config/config.yaml ./config.yaml

EXPOSE 8080
CMD ["./api-gateway"]