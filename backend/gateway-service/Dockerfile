FROM golang:1.25-alpine AS builder

WORKDIR /app/backend

COPY backend/go.mod backend/go.sum ./

RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY backend/. .

RUN --mount=type=cache,target=/go/pkg/mod go build -o /app/gateway-service ./gateway-service/cmd/main.go

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/gateway-service .

EXPOSE 8000

CMD ["./gateway-service"]
