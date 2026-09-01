FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.work go.work.sum ./
COPY pkg ./pkg
COPY services/dispatch ./services/dispatch
COPY services/triage ./services/triage
COPY services/data ./services/data

WORKDIR /app/services/data
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /data-service ./cmd/server

FROM alpine:3.20

RUN apk --no-cache add ca-certificates curl

WORKDIR /root/
COPY --from=builder /data-service .

EXPOSE 8083

ENTRYPOINT ["./data-service"]
