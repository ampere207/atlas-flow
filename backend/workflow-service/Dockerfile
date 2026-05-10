FROM golang:1.25-alpine AS builder

WORKDIR /app/backend

COPY backend/go.mod ./

RUN go mod download

COPY backend/. .

RUN go build -o /app/workflow-service ./workflow-service/cmd/main.go

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/workflow-service .

EXPOSE 8002

CMD ["./workflow-service"]
