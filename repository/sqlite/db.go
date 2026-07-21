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
CREATE TABLE IF NOT EXISTS users (
	id            INTEGER PRIMARY KEY AUTOINCREMENT,
	name          TEXT    NOT NULL,
	email         TEXT    NOT NULL UNIQUE,
	phone         TEXT    NOT NULL DEFAULT '',
	password_hash TEXT    NOT NULL,
	created_at    DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS rides (
	id            INTEGER PRIMARY KEY AUTOINCREMENT,
	slug          TEXT    NOT NULL UNIQUE,
	name          TEXT    NOT NULL,
	category      TEXT    NOT NULL,
	tagline       TEXT    NOT NULL DEFAULT '',
	description   TEXT    NOT NULL DEFAULT '',
	emoji         TEXT    NOT NULL DEFAULT '',
	image_url     TEXT    NOT NULL DEFAULT '',
	price         INTEGER NOT NULL DEFAULT 0,
	duration_min  INTEGER NOT NULL DEFAULT 0,
	min_height_cm INTEGER NOT NULL DEFAULT 0,
	thrill_level  INTEGER NOT NULL DEFAULT 1,
	daily_quota   INTEGER NOT NULL DEFAULT 0,
	is_active     INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS orders (
	id             INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id        INTEGER NOT NULL DEFAULT 0,
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
	if err := addMissingColumns(db); err != nil {
		return err
	}
	// Index yang bergantung pada kolom tambahan dibuat setelah kolomnya dipastikan ada,
	// supaya basis data lama tidak gagal saat dimigrasikan.
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_orders_user ON orders(user_id)`); err != nil {
		return err
	}
	if err := seedRides(db); err != nil {
		return err
	}
	return backfillRideImages(db)
}

// addMissingColumns menambahkan kolom yang baru diperkenalkan pada versi berikutnya,
// supaya basis data yang sudah berisi data lama tetap dapat dipakai tanpa dihapus.
func addMissingColumns(db *sql.DB) error {
	needed := []struct{ table, column, definition string }{
		{"rides", "image_url", "TEXT NOT NULL DEFAULT ''"},
		{"orders", "user_id", "INTEGER NOT NULL DEFAULT 0"},
	}
	for _, n := range needed {
		exists, err := columnExists(db, n.table, n.column)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		stmt := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", n.table, n.column, n.definition)
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("gagal menambah kolom %s.%s: %w", n.table, n.column, err)
		}
	}
	return nil
}

func columnExists(db *sql.DB, table, column string) (bool, error) {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid        int
			name, ctyp string
			notNull    int
			dfltValue  sql.NullString
			pk         int
		)
		if err := rows.Scan(&cid, &name, &ctyp, &notNull, &dfltValue, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}

// backfillRideImages mengisi gambar wahana bawaan yang masih kosong, misalnya pada basis
// data yang dibuat sebelum kolom gambar diperkenalkan.
func backfillRideImages(db *sql.DB) error {
	stmt, err := db.Prepare(`UPDATE rides SET image_url = ? WHERE slug = ? AND image_url = ''`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, r := range seedData {
		if r.ImageURL == "" {
			continue
		}
		if _, err := stmt.Exec(r.ImageURL, r.Slug); err != nil {
			return err
		}
	}
	return nil
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
		(slug, name, category, tagline, description, emoji, image_url, price, duration_min, min_height_cm, thrill_level, daily_quota, is_active)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, r := range seedData {
		if _, err := stmt.Exec(r.Slug, r.Name, r.Category, r.Tagline, r.Description, r.Emoji,
			r.ImageURL, r.Price, r.DurationMin, r.MinHeightCm, r.ThrillLevel, r.DailyQuota); err != nil {
			return err
		}
	}
	return tx.Commit()
}
