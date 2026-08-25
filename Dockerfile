FROM golang:1.26-alpine AS build

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o main ./cmd/main.go

FROM alpine:latest

RUN adduser -D appuser

WORKDIR /app
COPY --from=build --chown=appuser:appuser /app/main ./

USER appuser

CMD ["./main"]