FROM golang:1.24-alpine AS builder

WORKDIR /app

# Копируем go.mod сервиса (который уже содержит replace)
COPY api-gateway/go.mod api-gateway/go.sum ./api-gateway/

# Копируем корневой go.mod (нужен для replace)
COPY go.mod go.sum ./

# Копируем gen папку (нужна для сборки)
COPY gen/ ./gen/

# Переходим в папку сервиса
WORKDIR /app/api-gateway

# Скачиваем зависимости (replace сработает)
RUN go mod download

# Копируем остальной код сервиса
COPY api-gateway/ ./

# Собираем
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o api-gateway ./cmd


FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/api-gateway/api-gateway .
COPY api-gateway/config/config.yaml ./config.yaml

EXPOSE 8080
CMD ["./api-gateway"]