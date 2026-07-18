# Flash Sale Mini — Redis + RabbitMQ

Sistem mini penjualan tiket **flash sale** yang menggabungkan dua topik Topik Khusus:

- **m1 — Redis:** reservasi stok tiket secara **atomik** (Lua script `DECRBY`) sehingga dijamin **tidak pernah oversell** walau ratusan orang checkout bersamaan.
- **m2 — Message Queue (RabbitMQ):** pemrosesan **asinkron** — notifikasi order dan **auto-expire** order yang tidak dibayar memakai mekanisme **TTL + Dead Letter Queue**.

Terinspirasi dari SKPL EventHub (rekan), tetapi dipersempit ke inti integrasi Redis + MQ, ditulis dengan Go + Gin (senada gaya `ms1.track-method`) dan UI HTML/JS tanpa framework.

---

## Arsitektur

```
                    ┌──────────────┐   GET /product (polling)
   Browser (UI) ───▶│   app         │◀──────────────────────────┐
   Beli / Bayar     │  (Gin HTTP)   │                            │
                    └──────┬───────┘                            │
                           │                                     │
         reservasi atomik  │ (Lua DECRBY)         baca stok/order│
                           ▼                                     │
                    ┌──────────────┐                             │
                    │    Redis      │  stok, order, index ───────┘
                    └──────────────┘
                           ▲
              publish      │ notify + expiry(TTL)
                           ▼
                    ┌──────────────┐
                    │   RabbitMQ    │
                    │  ┌─────────┐  │  notify.queue ─────▶ [worker] kirim email (simulasi)
                    │  │ TTL 60s │  │
                    │  │wait queue│─┼─(TTL habis)─▶ DLX ─▶ expiry.process ─▶ [worker] expire + kembalikan stok
                    │  └─────────┘  │
                    └──────────────┘
```

Dua service dari satu image (mirip publisher/consumer di `ms2`):
- **app** (`APP_MODE=server`) — HTTP API + UI.
- **worker** (`APP_MODE=worker`) — consumer notifikasi & auto-expire.

## Alur Zero-Oversell

1. User menekan **Beli** → `POST /api/v1/checkout`.
2. Redis menjalankan Lua script: cek stok lalu `DECRBY` dalam satu operasi atomik. Jika stok kurang → ditolak (`409`), **tidak ada order dibuat**.
3. Order dibuat berstatus `PENDING` (batas bayar 60 detik), lalu:
   - publish notifikasi ke fanout exchange (`worker` mensimulasikan kirim email),
   - publish pesan ke **wait queue** ber-TTL 60 detik.
4. Jika dibayar (`POST /orders/:id/pay`) sebelum kedaluwarsa → status `PAID`.
5. Jika **tidak** dibayar → setelah 60 detik pesan di **wait queue** di-*dead-letter* ke `expiry.process` → `worker` meng-expire order & mengembalikan stok (`INCRBY`).

## Endpoints API

| Method | Path | Deskripsi |
|--------|------|-----------|
| `GET`  | `/health` | Health check |
| `GET`  | `/api/v1/config` | Info produk & TTL (untuk UI) |
| `GET`  | `/api/v1/product` | Produk + stok tersisa (real-time dari Redis) |
| `POST` | `/api/v1/checkout` | Reservasi stok & buat order PENDING |
| `POST` | `/api/v1/orders/:id/pay` | Bayar order (PENDING → PAID) |
| `POST` | `/api/v1/orders/:id/cancel` | Batalkan order & kembalikan stok |
| `GET`  | `/api/v1/orders/:id` | Detail satu order |
| `GET`  | `/api/v1/orders` | 50 order terbaru |

### Contoh Request

```bash
# Checkout 2 tiket
curl -X POST http://localhost:8095/api/v1/checkout \
  -H "Content-Type: application/json" -d '{"quantity": 2}'

# Bayar
curl -X POST http://localhost:8095/api/v1/orders/ORD-xxxx/pay
```

## Environment Variables

| Variabel | Default | Keterangan |
|----------|---------|------------|
| `APP_MODE` | `all` | `server`, `worker`, atau `all` |
| `PORT` | `8080` | Port HTTP (di-map ke `8095` di host) |
| `REDIS_ADDR` | `localhost:6379` | Alamat Redis |
| `RABBITMQ_URL` | `amqp://guest:guest@localhost:5672/` | URL RabbitMQ |
| `ORDER_TTL_SECONDS` | `60` | Batas waktu bayar sebelum auto-expire |
| `PRODUCT_NAME` | `Tiket Flash Sale EventHub 2026` | Nama produk |
| `PRODUCT_STOCK` | `20` | Stok awal (di-seed sekali) |

## Menjalankan (Docker Compose)

Cukup butuh Docker + Docker Compose di host. Redis & RabbitMQ ikut di-provision.

```bash
docker compose up -d --build

# UI:                http://<host>:8095
# RabbitMQ UI:       http://<host>:15673  (user/pass: guest/guest)

docker compose logs -f worker   # lihat notifikasi & proses auto-expire
docker compose down             # hentikan
docker compose down -v          # hentikan + hapus data
```

### Menguji Zero-Oversell

Stok awal 20. Tembak 50 checkout paralel — hanya 20 yang sukses, sisanya `409 stok habis`:

```bash
seq 50 | xargs -P50 -I{} curl -s -o /dev/null -w "%{http_code}\n" \
  -X POST http://localhost:8095/api/v1/checkout \
  -H "Content-Type: application/json" -d '{"quantity":1}' | sort | uniq -c
```

## Struktur Project

```
s6.tp.flash-sale/
├── config/        # baca env var
├── domain/        # entity, kontrak (interface), error
├── repository/    # Redis: stok atomik (Lua) + order
├── queue/         # RabbitMQ: topologi TTL + DLX, publisher
├── usecase/       # logika bisnis flash sale
├── handler/       # HTTP handler (Gin)
├── worker/        # consumer notifikasi & auto-expire
├── middleware/    # logger & CORS
├── web/           # UI (HTML/CSS/JS tanpa framework)
├── main.go
├── Dockerfile
└── docker-compose.yml
```

## Stack

Go 1.22 · Gin · go-redis v9 · amqp091-go · Redis 7 · RabbitMQ 3 · Docker Compose
