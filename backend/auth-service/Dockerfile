FROM golang:1.25-alpine AS builder

WORKDIR /app/backend

COPY backend/go.mod ./

RUN go mod download

COPY backend/. .

RUN go build -o /app/auth-service ./auth-service/cmd/main.go

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/auth-service .

EXPOSE 8001

CMD ["./auth-service"]
