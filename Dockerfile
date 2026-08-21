FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/task-manager ./cmd/app

FROM alpine:3.22

WORKDIR /app

COPY --from=builder /app/task-manager ./task-manager
COPY migrations ./migrations

EXPOSE 8080

CMD ["./task-manager"]