FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o golem ./cmd/golem

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/golem .

COPY config.json /etc/golem/config.json

EXPOSE 8080

CMD ["./golem"]
