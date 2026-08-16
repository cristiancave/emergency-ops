# Se construye con contexto = raíz del repo (no services/triage/), porque
# triage importa el módulo local emergencyops/pkg vía go.work: sin la raíz
# en el contexto, Docker no puede ver pkg/.
#
#   docker build -f services/triage/Dockerfile -t triage-service:latest .

# Stage 1: Build stage
FROM golang:1.26.6-alpine AS builder

RUN apk add --no-cache ca-certificates git

WORKDIR /app

# go.work + go.mod/go.sum de los 3 módulos primero: permite cachear
# "go mod download" en una capa que no se invalida por cambios de código.
COPY go.work ./
COPY pkg/go.mod pkg/go.sum ./pkg/
COPY services/triage/go.mod services/triage/go.sum ./services/triage/
COPY services/dispatch/go.mod services/dispatch/go.sum ./services/dispatch/

WORKDIR /app/services/triage
RUN go mod download

WORKDIR /app
COPY pkg ./pkg
COPY services/triage ./services/triage

WORKDIR /app/services/triage
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o triage-server ./cmd/server

# Stage 2: Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

RUN addgroup -g 1000 -S appuser && \
    adduser -u 1000 -S appuser -G appuser

WORKDIR /home/appuser

COPY --from=builder --chown=appuser:appuser /app/services/triage/triage-server .

USER appuser

EXPOSE 8081

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8081/health || exit 1

CMD ["./triage-server"]
