# Taman Wahana Nusantara

Aplikasi web penjualan tiket wahana taman hiburan. Dibangun sebagai proyek mata kuliah
Topik Khusus untuk mendemonstrasikan penerapan **Redis** dan **Message Queue (RabbitMQ)**
pada sebuah sistem yang utuh, lengkap dengan antarmuka pengguna dan basis data.

Antarmuka React, backend Go, dan basis data SQLite dikemas menjadi **satu container
monolit**. Redis dan RabbitMQ berjalan sebagai container pendamping.

---

## Peran Setiap Teknologi

| Teknologi | Peran pada sistem |
|-----------|-------------------|
| **Redis** | Kuota tiket harian per wahana yang dikurangi secara **atomik** lewat Lua script, sehingga tiket tidak pernah terjual melebihi kuota. Sekaligus menjadi cache katalog wahana. |
| **RabbitMQ** | Menjalankan tiga pekerjaan asinkron: penerbitan tiket setelah pembayaran, notifikasi ke pengunjung, dan pembatalan otomatis order yang tidak dibayar (TTL + Dead Letter Queue). |
| **SQLite** | Penyimpanan permanen katalog wahana, order, item order, dan tiket. Berkas basis data disimpan pada volume Docker. |
| **Go (Gin)** | Logika backend dengan pemisahan lapisan domain, repository, usecase, handler, dan worker. |
| **React (Vite)** | Antarmuka pengguna satu halaman bergaya iOS modern, hanya mode terang. |

## Arsitektur

```
                          Satu container aplikasi
   ┌───────────────────────────────────────────────────────────┐
   │  React (hasil build ditanam ke binary lewat go:embed)      │
   │                          │                                 │
   │                    Go (Gin) API                            │
   │                    │            │                          │
   │              SQLite            worker consumer             │
   └────────────────────┼────────────────┼─────────────────────┘
                        │                │
              ┌─────────▼──────┐  ┌──────▼───────────────────┐
              │     Redis      │  │        RabbitMQ          │
              │ kuota atomik   │  │ notifikasi, terbit tiket │
              │ cache katalog  │  │ TTL + Dead Letter Queue  │
              └────────────────┘  └──────────────────────────┘
```

## Katalog Wahana

Terdapat **32 wahana** yang dikelompokkan ke dalam enam kategori:

| Kategori | Contoh wahana |
|----------|---------------|
| **Ekstrem** | Halilintar Petir, Ular Besi Terbalik, Menara Hysteria, Tornado Spin, Kora Kora Samudra, Ayunan Langit |
| **Keluarga** | Bianglala Cakrawala, Komidi Putar Kencana, Mobil Tabrakan, Kereta Wisata Keliling, Perahu Angsa Danau, Rumah Cermin Labirin |
| **Anak** | Istana Balon Ceria, Kolam Bola Pelangi, Kuda Poni Mini, Kereta Mini Anak |
| **Wahana Air** | Seluncur Air Raksasa, Kolam Ombak Samudra, Arung Jeram Log Flume, Sungai Santai, Ember Tumpah Raksasa |
| **Petualangan** | Flying Fox Lintas Danau, Jembatan Gantung Sky Bridge, Panjat Tebing Buatan, Lintasan ATV Off Road, Arena Paintball, Taman Tali Tinggi |
| **Indoor** | Rumah Hantu Nusantara, Bioskop 4D Petualangan, Zona Realitas Virtual, Arena Arcade dan Game, Planetarium Mini |

Setiap wahana memiliki harga, durasi, syarat tinggi minimum, tingkat tantangan, dan
kuota harian tersendiri.

## Alur Pemesanan Tiket

1. Pengunjung memilih wahana dan tanggal kunjungan, lalu memasukkannya ke keranjang.
2. Saat checkout, kuota setiap wahana **direservasi atomik di Redis**. Bila salah satu
   wahana kehabisan kuota, seluruh reservasi pada pesanan itu dikembalikan.
3. Order tersimpan di SQLite berstatus `PENDING`, kode **QRIS** terbit, dan pesan
   penjadwalan kedaluwarsa dikirim ke antrean ber-TTL.
4. Pembayaran diselesaikan lewat halaman **`/test/qris-list`** (lihat catatan di bawah).
5. Setelah lunas, pekerjaan penerbitan tiket dikirim ke RabbitMQ. Worker menerbitkan
   tiket satu per satu, sehingga respons pembayaran tetap cepat. Halaman tiket akan
   memperbarui sendiri saat tiket selesai dibuat.
6. Bila tidak dibayar sampai batas waktu, pesan pada antrean penunda di-*dead letter*
   ke antrean pemroses, worker menandai order `EXPIRED` dan mengembalikan kuotanya.

### Catatan mengenai pembayaran QRIS

Kode QRIS yang dihasilkan mengikuti format EMVCo yang benar (termasuk perhitungan
CRC16), namun memakai identitas merchant **simulasi** dan tidak terhubung ke rekening
mana pun. Kode ini dibuat murni untuk keperluan demonstrasi akademik dan tidak dapat
memproses uang sungguhan.

Karena itu tidak ada notifikasi pembayaran dari penyedia pembayaran. Pelunasan
dilakukan lewat halaman **`/test/qris-list`** yang menampilkan seluruh pesanan yang
menunggu pembayaran beserta tombol untuk menyelesaikannya.

## Halaman Aplikasi

| Rute | Keterangan |
|------|-----------|
| `/` | Beranda: kategori wahana, wahana populer, panduan memesan |
| `/wahana` | Katalog lengkap dengan pencarian, filter kategori, dan pengurutan |
| `/wahana/:slug` | Detail wahana, pemilihan tanggal dan jumlah tiket |
| `/keranjang` | Keranjang, data pemesan, dan checkout |
| `/pembayaran/:code` | Tampilan QRIS, hitung mundur, pemantauan status pembayaran |
| `/tiket` dan `/tiket/:code` | Pencarian pesanan dan daftar tiket elektronik |
| `/test/qris-list` | Halaman uji pembayaran serta pemantauan kedalaman antrean |
| `/admin` | Login admin, statistik penjualan, pengelolaan katalog wahana |

## Endpoint API

| Method | Path | Auth | Keterangan |
|--------|------|------|-----------|
| `GET` | `/health` | - | Health check |
| `GET` | `/api/v1/categories` | - | Daftar kategori beserta jumlah wahana |
| `GET` | `/api/v1/rides` | - | Katalog wahana, mendukung `?category=` dan `?date=` |
| `GET` | `/api/v1/rides/:slug` | - | Detail satu wahana |
| `POST` | `/api/v1/orders` | - | Checkout, membuat order dan kode QRIS |
| `GET` | `/api/v1/orders/:code` | - | Status order beserta gambar QRIS |
| `POST` | `/api/v1/orders/:code/cancel` | - | Membatalkan order dan mengembalikan kuota |
| `GET` | `/api/v1/orders/:code/tickets` | - | Daftar tiket sebuah order |
| `POST` | `/api/v1/tickets/:code/scan` | - | Menandai tiket sudah dipakai di gerbang |
| `GET` | `/api/v1/test/pending-orders` | - | Order yang menunggu pembayaran (halaman uji) |
| `POST` | `/api/v1/test/orders/:code/settle` | - | Menyelesaikan pembayaran (halaman uji) |
| `GET` | `/api/v1/test/system` | - | Kedalaman antrean RabbitMQ |
| `POST` | `/api/v1/auth/login` | - | Login admin, menerbitkan JWT |
| `GET` | `/api/v1/admin/stats` | Admin | Statistik penjualan |
| `GET` | `/api/v1/admin/orders` | Admin | Pesanan terbaru |
| `POST` `PUT` `DELETE` | `/api/v1/admin/rides...` | Admin | Kelola katalog wahana |

## Environment Variable

| Variabel | Default | Keterangan |
|----------|---------|-----------|
| `APP_MODE` | `all` | `server`, `worker`, atau `all` |
| `PORT` | `8080` | Port HTTP di dalam container |
| `DATABASE_PATH` | `data/wahana.db` | Lokasi berkas SQLite |
| `REDIS_ADDR` | `localhost:6379` | Alamat Redis |
| `RABBITMQ_URL` | `amqp://guest:guest@localhost:5672/` | URL RabbitMQ |
| `PAYMENT_TTL_MINUTES` | `10` | Batas waktu pembayaran sebelum order kedaluwarsa |
| `CACHE_TTL_SECONDS` | `60` | Masa berlaku cache katalog |
| `QUOTA_TTL_DAYS` | `45` | Masa simpan kunci kuota harian di Redis |
| `MERCHANT_NAME` | `TAMAN WAHANA SIMULASI` | Nama merchant pada payload QRIS |
| `ADMIN_USERNAME` | `admin` | Username admin |
| `ADMIN_PASSWORD` | `admin123` | Password admin, di-hash bcrypt saat startup |
| `JWT_SECRET` | `wahana-dev-secret-change-me` | Kunci penandatanganan JWT, wajib diganti di produksi |

## Menjalankan

Seluruh image diambil dari registry, tidak ada proses build pada `docker-compose.yml`.

```bash
docker compose up -d
docker compose logs -f app     # memantau worker antrean dan notifikasi
docker compose down            # menghentikan
```

Image aplikasi: [`itsanla/s6.tp.flash-sale`](https://hub.docker.com/r/itsanla/s6.tp.flash-sale) (`v2.0.0`, `latest`).

### Akses

| Layanan | Port langsung | Domain (HTTPS otomatis) |
|---------|---------------|-------------------------|
| Aplikasi | `http://<ip>:6003` | https://topik-khusus.akademik.anla.works |
| RabbitMQ management | `http://<ip>:6004` | https://rabbitmq.akademik.anla.works |
| Redis | `<ip>:6005` | - |

## Struktur Project

```
.
├── domain/          entity, kontrak repository, pesan antrean, error
├── repository/
│   ├── sqlite/      skema, seed katalog, repository wahana/order/tiket
│   └── rediscache/  kuota atomik (Lua) dan cache
├── queue/           topologi RabbitMQ dan publisher
├── usecase/         logika katalog serta pemesanan tiket
├── handler/         rute HTTP, penyajian aplikasi React
├── worker/          consumer notifikasi, penerbitan tiket, kedaluwarsa
├── qris/            penyusun payload QRIS dan perender gambar QR
├── auth/            JWT admin
├── web/             sumber React (Vite) dan hasil build yang di-embed
├── main.go
├── Dockerfile       build tiga tahap: Node, Go, image akhir
└── docker-compose.yml
```

## Stack

Go 1.25 · Gin · modernc SQLite (tanpa CGO) · go-redis v9 · amqp091-go · golang-jwt ·
React 18 · Vite 5 · React Router 6 · Redis 7 · RabbitMQ 3 · Docker Compose
