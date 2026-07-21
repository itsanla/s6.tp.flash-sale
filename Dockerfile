# =============================================================================
# Aplikasi monolit: antarmuka React, backend Go, dan basis data SQLite menyatu
# dalam satu image. Hasil build React ditanamkan langsung ke dalam binary Go,
# sehingga container ini tidak memerlukan server berkas statis terpisah.
# =============================================================================

# ---- Tahap 1: build antarmuka React ----
FROM node:20-alpine AS frontend

WORKDIR /frontend
COPY web/package.json ./
RUN npm install

COPY web/ ./
RUN npm run build

# ---- Tahap 2: build binary Go ----
FROM golang:1.23-alpine AS backend

WORKDIR /app
RUN apk add --no-cache git

COPY . .
# Hasil build React menggantikan folder dist kosong sebelum proses embed berjalan.
COPY --from=frontend /frontend/dist ./web/dist

# Driver SQLite yang dipakai murni Go, sehingga binary tetap statis tanpa CGO.
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o wahanapark .

# ---- Tahap 3: image akhir ----
FROM alpine:3.20

WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -u 10001 appuser && \
    mkdir -p /app/data && chown -R appuser:appuser /app

COPY --from=backend /app/wahanapark ./wahanapark

USER appuser
EXPOSE 8080
VOLUME ["/app/data"]

CMD ["./wahanapark"]
