FROM golang:1.25-alpine AS builder

WORKDIR /app/backend

COPY backend/go.mod ./

RUN go mod download

COPY backend/. .

RUN go build -o /app/gateway-service ./gateway-service/cmd/main.go

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/gateway-service .

EXPOSE 8000

CMD ["./gateway-service"]
