### Dockerfile para backend de Go en Render o Railway

# Etapa 1: Build
FROM golang:1.22-alpine AS builder
WORKDIR /app

# Instalar dependencias necesarias para compilar
RUN apk add --no-cache gcc musl-dev

# Copiar módulos y descargar dependencias
COPY go.mod go.sum ./
RUN go mod download

# Copiar el código y compilar
COPY . ./
RUN go build -o blackjack-server

# Etapa 2: Run
FROM alpine:3.19
WORKDIR /app

# Copiar binario desde la etapa anterior
COPY --from=builder /app/blackjack-server .

# Exponer puerto
EXPOSE 8080

# Comando para ejecutar
ENTRYPOINT ["./blackjack-server"]
