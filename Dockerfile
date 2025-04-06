FROM golang:1.22-alpine AS builder
WORKDIR /app

RUN apk add --no-cache gcc musl-dev

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN go build -o blackjack-server

FROM alpine:3.19
WORKDIR /app

COPY --from=builder /app/blackjack-server .

EXPOSE 8080

ENTRYPOINT ["./blackjack-server"]
