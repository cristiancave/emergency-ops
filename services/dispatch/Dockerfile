# Se construye con contexto = raíz del repo (no services/dispatch/), porque
# dispatch importa el módulo local emergencyops/pkg vía go.work: sin la raíz
# en el contexto, Docker no puede ver pkg/.
#
#   docker build -f services/dispatch/Dockerfile -t dispatch-service:latest .

# Stage 1: Build stage
FROM golang:1.26.6-alpine AS builder

RUN apk add --no-cache ca-certificates git

WORKDIR /app

# go.work + go.mod/go.sum de los 3 módulos primero: permite cachear
# "go mod download" en una capa que no se invalida por cambios de código.
COPY go.work ./
COPY pkg/go.mod pkg/go.sum ./pkg/
COPY services/dispatch/go.mod services/dispatch/go.sum ./services/dispatch/
COPY services/triage/go.mod services/triage/go.sum ./services/triage/

WORKDIR /app/services/dispatch
RUN go mod download

WORKDIR /app
COPY pkg ./pkg
COPY services/dispatch ./services/dispatch

WORKDIR /app/services/dispatch
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o dispatch-server ./cmd/server

# Stage 2: Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

RUN addgroup -g 1000 -S appuser && \
    adduser -u 1000 -S appuser -G appuser

WORKDIR /home/appuser

COPY --from=builder --chown=appuser:appuser /app/services/dispatch/dispatch-server .

USER appuser

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

CMD ["./dispatch-server"]
