# Stage 1: Build stage
FROM golang:1.26.6-alpine AS builder

# Instalar certificados SSL y git
RUN apk add --no-cache ca-certificates git

# Establecer el directorio de trabajo
WORKDIR /app

# Copiar go.mod (obligatorio)
COPY go.mod .

# Copiar go.sum si existe
COPY go.* ./

# Descargar dependencias
RUN go mod download

# Copiar todo el código fuente
COPY . .

# Compilar el binario
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o triage-server ./cmd/server

# Stage 2: Runtime stage
FROM alpine:latest

# Instalar certificados SSL en la imagen final
RUN apk --no-cache add ca-certificates

# Crear un usuario no-root por seguridad
RUN addgroup -g 1000 -S appuser && \
    adduser -u 1000 -S appuser -G appuser

WORKDIR /home/appuser

# Copiar el binario compilado desde el stage anterior
COPY --from=builder --chown=appuser:appuser /app/triage-server .

# Cambiar al usuario no-root
USER appuser

# Exponer el puerto (default 8081)
EXPOSE 8081

# Configurar healthcheck
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8081/health || exit 1

# Ejecutar el servidor
CMD ["./triage-server"]
