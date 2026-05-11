FROM golang:1.25-alpine AS builder

ENV GOTOOLCHAIN=auto

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY analytics-service/ ./analytics-service/
COPY gen/ ./gen/ 

COPY analytics-service/go.mod analytics-service/go.sum ./analytics-service/
WORKDIR /app/analytics-service
RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o analytics-service ./cmd


FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/analytics-service .
COPY analytics-service/config/config.yaml ./config.yaml

EXPOSE 50053
CMD ["./analytics-service"]


