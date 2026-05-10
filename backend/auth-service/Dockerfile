FROM golang:1.25-alpine AS builder

WORKDIR /app/backend

COPY backend/go.mod backend/go.sum ./

RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY backend/. .

RUN --mount=type=cache,target=/go/pkg/mod go build -o ./bin/auth-service ./auth-service/cmd/main.go

FROM alpine:3.20

WORKDIR /root/

COPY --from=builder /app/backend/bin/auth-service ./

EXPOSE 8001

CMD ["./auth-service"]
