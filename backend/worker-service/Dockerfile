FROM golang:1.25-alpine AS builder

WORKDIR /app/backend

COPY backend/go.mod ./

RUN go mod download

COPY backend/. .

RUN go build -o /app/worker-service ./worker-service/cmd/main.go

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/worker-service .

EXPOSE 8003

CMD ["./worker-service"]
