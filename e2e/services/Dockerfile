FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY main.go .
RUN go build -o mock-service main.go

FROM alpine:3.19
COPY --from=builder /app/mock-service /usr/local/bin/mock-service
ENTRYPOINT ["mock-service"]
