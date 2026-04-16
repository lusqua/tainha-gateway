FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o tainha ./cmd/gateway

FROM alpine:3.19
COPY --from=builder /app/tainha /usr/local/bin/tainha
ENTRYPOINT ["tainha"]
