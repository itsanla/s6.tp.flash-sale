# ---- Build stage ----
FROM golang:1.23-alpine AS builder

WORKDIR /app
RUN apk add --no-cache git

# Salin sumber lalu resolusi dependensi (go.sum di-generate saat build).
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o flashsale .

# ---- Runtime stage ----
FROM alpine:3.20

WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -u 10001 appuser

COPY --from=builder /app/flashsale ./flashsale
COPY --from=builder /app/web ./web

USER appuser
EXPOSE 8080
CMD ["./flashsale"]
