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

Satu container app (`APP_MODE=all`) menjalankan HTTP API + UI **dan** worker consumer
(notifikasi & auto-expire) sekaligus. Total 3 container: **app**, **redis**, **rabbitmq**.

> Mode dapat dipisah (`server` / `worker`) bila ingin scale consumer terpisah.

## Alur Zero-Oversell

1. User menekan **Beli** → `POST /api/v1/checkout`.
2. Redis menjalankan Lua script: cek stok lalu `DECRBY` dalam satu operasi atomik. Jika stok kurang → ditolak (`409`), **tidak ada order dibuat**.
3. Order dibuat berstatus `PENDING` (batas bayar 60 detik), lalu:
   - publish notifikasi ke fanout exchange (`worker` mensimulasikan kirim email),
   - publish pesan ke **wait queue** ber-TTL 60 detik.
4. Jika dibayar (`POST /orders/:id/pay`) sebelum kedaluwarsa → status `PAID`.
5. Jika **tidak** dibayar → setelah 60 detik pesan di **wait queue** di-*dead-letter* ke `expiry.process` → `worker` meng-expire order & mengembalikan stok (`INCRBY`).

## Uji Beban (Load Test) — Bukti RabbitMQ & Redis Bekerja di Bawah Beban Tinggi

Panel "⚡ Uji Beban" di UI mendemokan **decoupling** lewat message queue: client mengirim N pesanan sekaligus (mis. 10.000), langsung dapat respons sukses instan, sementara pemrosesan nyata terjadi asinkron di belakang layar.

Alur `POST /api/v1/loadtest {"quantity": N}`:
1. Stok ditambah sejumlah N (`INCRBY`) — supaya seluruh batch bisa berhasil; fokus uji ini adalah **throughput queue**, bukan zero-oversell (yang sudah dibuktikan lewat checkout normal & uji 50-paralel di atas).
2. Sebuah `batch_id` dibuat & tracker progres diinisialisasi di Redis (hash: `requested/submitted/processed/success/failed`).
3. Respons **202** dikembalikan segera ke client.
4. Di background (goroutine terpisah), N pesan dipublish ke antrean `flashsale.bulk.queue`.
5. Beberapa worker paralel (`LOADTEST_CONCURRENCY`) mengonsumsi antrean tsb secara *competing consumers* — tiap pesan memicu **checkout + auto-pay nyata** (reservasi atomik Redis, buat order, publish notifikasi), lalu progres di-update (`HINCRBY`).
6. UI polling `GET /api/v1/loadtest/:batch_id` tiap 400ms → progress bar & "Riwayat Order" terlihat bertambah bertahap sampai selesai.

```bash
curl -X POST http://localhost:6003/api/v1/loadtest -H "Content-Type: application/json" -d '{"quantity": 10000}'
curl http://localhost:6003/api/v1/loadtest/BATCH-xxxx
```

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
| `POST` | `/api/v1/loadtest` | Mulai uji beban N pesanan (async via RabbitMQ) |
| `GET`  | `/api/v1/loadtest/:batch_id` | Progres real-time sebuah batch uji beban |

### Contoh Request

```bash
# Checkout 2 tiket
curl -X POST http://localhost:6003/api/v1/checkout \
  -H "Content-Type: application/json" -d '{"quantity": 2}'

# Bayar
curl -X POST http://localhost:6003/api/v1/orders/ORD-xxxx/pay
```

## Environment Variables

| Variabel | Default | Keterangan |
|----------|---------|------------|
| `APP_MODE` | `all` | `server`, `worker`, atau `all` |
| `PORT` | `8080` | Port HTTP (di-map ke `6003` di host) |
| `REDIS_ADDR` | `localhost:6379` | Alamat Redis |
| `RABBITMQ_URL` | `amqp://guest:guest@localhost:5672/` | URL RabbitMQ |
| `ORDER_TTL_SECONDS` | `60` | Batas waktu bayar sebelum auto-expire |
| `PRODUCT_NAME` | `Tiket Flash Sale EventHub 2026` | Nama produk |
| `PRODUCT_STOCK` | `20` | Stok awal (di-seed sekali) |
| `LOADTEST_MAX_QUANTITY` | `50000` | Batas aman jumlah pesanan per batch uji beban |
| `LOADTEST_CONCURRENCY` | `20` | Jumlah worker paralel pemroses antrean bulk |
| `LOADTEST_DELAY_MS` | `15` | Simulasi waktu proses per pesanan (ms) |

## Menjalankan (Docker Compose)

Semua service (app, redis, rabbitmq) diambil dari **image registry** — tidak ada build.
Cukup butuh Docker + Docker Compose di host.

```bash
docker compose up -d     # tarik image & jalankan 3 container
docker compose logs -f app   # lihat notifikasi & proses auto-expire
docker compose down          # hentikan & hapus container
```

Image aplikasi: [`itsanla/s6.tp.flash-sale`](https://hub.docker.com/r/itsanla/s6.tp.flash-sale) (`v1.1.0`, `latest`).

### Akses

| Layanan | Port langsung | Domain (Traefik, HTTPS + Let's Encrypt) |
|---------|---------------|------------------------------------------|
| Aplikasi (UI/API) | `http://<ip>:6003` | https://topik-khusus.akademik.anla.works |
| RabbitMQ mgmt UI | `http://<ip>:6004` | https://rabbitmq.akademik.anla.works |
| Redis | `<ip>:6005` | — |

Akses via port langsung tetap HTTP polos (tanpa TLS); akses via domain otomatis diarahkan ke HTTPS.

RabbitMQ UI login: `guest` / `guest`.

### Menguji Zero-Oversell

Stok awal 20. Tembak 50 checkout paralel — hanya 20 yang sukses, sisanya `409 stok habis`:

```bash
seq 50 | xargs -P50 -I{} curl -s -o /dev/null -w "%{http_code}\n" \
  -X POST http://localhost:6003/api/v1/checkout \
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
