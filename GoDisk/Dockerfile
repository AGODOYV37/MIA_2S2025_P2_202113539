# ===== Etapa de build =====
FROM golang:1.24.5-alpine AS builder
WORKDIR /src

ENV GOTOOLCHAIN=auto
RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .
ENV CGO_ENABLED=0
# Ajusta la ruta si tu main está en otro paquete
RUN go build -o /out/server ./main.go

# ===== Etapa de runtime =====
FROM alpine:3.20
WORKDIR /app

# Para healthcheck TCP y certificados
RUN apk add --no-cache curl ca-certificates netcat-openbsd

# 👇 Aquí usamos el stage llamado 'builder' de arriba
COPY --from=builder /out/server /app/server

ENV HTTP_ADDR=:8080
EXPOSE 8080
CMD ["/app/server", "-http", ":8080"]
