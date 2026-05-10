FROM golang:1.25-alpine AS builder

WORKDIR /app/backend

COPY backend/go.mod ./

RUN go mod download

COPY backend/. .

RUN go build -o /app/event-service ./event-service/cmd/main.go

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/event-service .

EXPOSE 8004

CMD ["./event-service"]
