FROM golang:1.26.6-alpine AS builder

RUN apk add --no-cache ca-certificates git

WORKDIR /app

COPY go.work ./
COPY pkg/go.mod pkg/go.sum ./pkg/
COPY services/dispatch/go.mod services/dispatch/go.sum ./services/dispatch/
COPY services/triage/go.mod services/triage/go.sum ./services/triage/
COPY services/data/go.mod services/data/go.sum* ./services/data/

WORKDIR /app/services/data
RUN go mod download

WORKDIR /app
COPY pkg ./pkg
COPY services/dispatch ./services/dispatch
COPY services/triage ./services/triage
COPY services/data ./services/data

WORKDIR /app/services/data
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /data-server ./cmd/server

FROM alpine:latest

RUN apk --no-cache add ca-certificates curl

RUN addgroup -g 1000 -S appuser && \
    adduser -u 1000 -S appuser -G appuser

WORKDIR /home/appuser

COPY --from=builder --chown=appuser:appuser /data-server .

USER appuser

EXPOSE 8083

CMD ["./data-server"]
