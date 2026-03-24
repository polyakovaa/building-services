FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY project-service/ ./project-service/
COPY gen/ ./gen/ 

COPY project-service/go.mod project-service/go.sum ./project-service/
WORKDIR /app/project-service
RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o project-service ./cmd


FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/project-service .
COPY project-service/config/config.yaml ./config.yaml

EXPOSE 50052
CMD ["./project-service"]


