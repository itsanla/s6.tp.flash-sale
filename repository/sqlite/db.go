package sqlite

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite" // driver SQLite murni Go, tidak butuh CGO
)

// Open membuka koneksi SQLite, menyiapkan folder data, lalu menjalankan migrasi.
//
// Catatan konkurensi: SQLite hanya mengizinkan satu penulis pada satu waktu. Aplikasi
// ini punya dua sumber tulisan (HTTP handler dan worker RabbitMQ), sehingga jumlah
// koneksi dibatasi satu agar seluruh tulisan berbaris rapi dan tidak pernah menemui
// error "database is locked". Mode WAL tetap dipakai supaya pembacaan tetap cepat.
func Open(path string) (*sql.DB, error) {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("gagal menyiapkan folder database: %w", err)
		}
	}

	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("gagal terhubung ke SQLite: %w", err)
	}
	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("gagal menjalankan migrasi: %w", err)
	}
	return db, nil
}

func migrate(db *sql.DB) error {
	schema := `
CREATE TABLE IF NOT EXISTS rides (
	id            INTEGER PRIMARY KEY AUTOINCREMENT,
	slug          TEXT    NOT NULL UNIQUE,
	name          TEXT    NOT NULL,
	category      TEXT    NOT NULL,
	tagline       TEXT    NOT NULL DEFAULT '',
	description   TEXT    NOT NULL DEFAULT '',
	emoji         TEXT    NOT NULL DEFAULT '',
	price         INTEGER NOT NULL DEFAULT 0,
	duration_min  INTEGER NOT NULL DEFAULT 0,
	min_height_cm INTEGER NOT NULL DEFAULT 0,
	thrill_level  INTEGER NOT NULL DEFAULT 1,
	daily_quota   INTEGER NOT NULL DEFAULT 0,
	is_active     INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS orders (
	id             INTEGER PRIMARY KEY AUTOINCREMENT,
	code           TEXT    NOT NULL UNIQUE,
	customer_name  TEXT    NOT NULL,
	customer_email TEXT    NOT NULL DEFAULT '',
	customer_phone TEXT    NOT NULL DEFAULT '',
	visit_date     TEXT    NOT NULL,
	status         TEXT    NOT NULL,
	total_amount   INTEGER NOT NULL DEFAULT 0,
	qris_payload   TEXT    NOT NULL DEFAULT '',
	created_at     DATETIME NOT NULL,
	expires_at     DATETIME NOT NULL,
	paid_at        DATETIME
);
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);
CREATE INDEX IF NOT EXISTS idx_orders_created ON orders(created_at DESC);

CREATE TABLE IF NOT EXISTS order_items (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	order_id   INTEGER NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
	ride_id    INTEGER NOT NULL,
	ride_slug  TEXT    NOT NULL DEFAULT '',
	ride_name  TEXT    NOT NULL,
	ride_emoji TEXT    NOT NULL DEFAULT '',
	unit_price INTEGER NOT NULL,
	quantity   INTEGER NOT NULL,
	subtotal   INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_items_order ON order_items(order_id);
CREATE INDEX IF NOT EXISTS idx_items_ride ON order_items(ride_id);

CREATE TABLE IF NOT EXISTS tickets (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	code       TEXT    NOT NULL UNIQUE,
	order_id   INTEGER NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
	order_code TEXT    NOT NULL,
	ride_id    INTEGER NOT NULL,
	ride_name  TEXT    NOT NULL,
	ride_emoji TEXT    NOT NULL DEFAULT '',
	visit_date TEXT    NOT NULL,
	status     TEXT    NOT NULL,
	issued_at  DATETIME NOT NULL,
	used_at    DATETIME
);
CREATE INDEX IF NOT EXISTS idx_tickets_order ON tickets(order_code);
`
	if _, err := db.Exec(schema); err != nil {
		return err
	}
	return seedRides(db)
}

// seedRides mengisi katalog wahana bila tabel masih kosong. Data hanya ditulis sekali
// sehingga perubahan yang dilakukan admin lewat aplikasi tidak akan tertimpa saat restart.
func seedRides(db *sql.DB) error {
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM rides`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT INTO rides
		(slug, name, category, tagline, description, emoji, price, duration_min, min_height_cm, thrill_level, daily_quota, is_active)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, r := range seedData {
		if _, err := stmt.Exec(r.Slug, r.Name, r.Category, r.Tagline, r.Description, r.Emoji,
			r.Price, r.DurationMin, r.MinHeightCm, r.ThrillLevel, r.DailyQuota); err != nil {
			return err
		}
	}
	return tx.Commit()
}
