package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"wahanapark/domain"
)

type orderRepository struct{ db *sql.DB }

func NewOrderRepository(db *sql.DB) domain.OrderRepository { return &orderRepository{db: db} }

const orderColumns = `id, user_id, code, customer_name, customer_email, customer_phone, visit_date,
	status, total_amount, qris_payload, created_at, expires_at, paid_at`

func scanOrder(row interface{ Scan(...any) error }) (*domain.Order, error) {
	var o domain.Order
	var paidAt sql.NullTime
	err := row.Scan(&o.ID, &o.UserID, &o.Code, &o.CustomerName, &o.CustomerEmail, &o.CustomerPhone,
		&o.VisitDate, &o.Status, &o.TotalAmount, &o.QRISPayload, &o.CreatedAt, &o.ExpiresAt, &paidAt)
	if err != nil {
		return nil, err
	}
	if paidAt.Valid {
		t := paidAt.Time
		o.PaidAt = &t
	}
	return &o, nil
}

// Create menyimpan order beserta seluruh itemnya dalam satu transaksi, sehingga order
// tidak pernah tersimpan setengah jadi bila salah satu item gagal ditulis.
func (s *orderRepository) Create(ctx context.Context, o *domain.Order) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, `INSERT INTO orders
		(user_id, code, customer_name, customer_email, customer_phone, visit_date, status, total_amount, qris_payload, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		o.UserID, o.Code, o.CustomerName, o.CustomerEmail, o.CustomerPhone, o.VisitDate,
		o.Status, o.TotalAmount, o.QRISPayload, o.CreatedAt, o.ExpiresAt)
	if err != nil {
		return err
	}
	o.ID, _ = res.LastInsertId()

	for i := range o.Items {
		it := &o.Items[i]
		it.OrderID = o.ID
		r, err := tx.ExecContext(ctx, `INSERT INTO order_items
			(order_id, ride_id, ride_slug, ride_name, ride_emoji, unit_price, quantity, subtotal)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			it.OrderID, it.RideID, it.RideSlug, it.RideName, it.RideEmoji, it.UnitPrice, it.Quantity, it.Subtotal)
		if err != nil {
			return err
		}
		it.ID, _ = r.LastInsertId()
	}
	return tx.Commit()
}

func (s *orderRepository) GetByCode(ctx context.Context, code string) (*domain.Order, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+orderColumns+` FROM orders WHERE code = ?`, code)
	o, err := scanOrder(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrOrderNotFound
	}
	if err != nil {
		return nil, err
	}
	items, err := s.loadItems(ctx, o.ID)
	if err != nil {
		return nil, err
	}
	o.Items = items
	return o, nil
}

func (s *orderRepository) loadItems(ctx context.Context, orderID int64) ([]domain.OrderItem, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, order_id, ride_id, ride_slug, ride_name,
		ride_emoji, unit_price, quantity, subtotal FROM order_items WHERE order_id = ? ORDER BY id`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]domain.OrderItem, 0, 4)
	for rows.Next() {
		var it domain.OrderItem
		if err := rows.Scan(&it.ID, &it.OrderID, &it.RideID, &it.RideSlug, &it.RideName,
			&it.RideEmoji, &it.UnitPrice, &it.Quantity, &it.Subtotal); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

func (s *orderRepository) listWithItems(ctx context.Context, query string, args ...any) ([]domain.Order, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	orders := make([]domain.Order, 0, 16)
	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		orders = append(orders, *o)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	// Item dimuat setelah kursor utama ditutup, karena koneksi SQLite dibatasi satu.
	for i := range orders {
		items, err := s.loadItems(ctx, orders[i].ID)
		if err != nil {
			return nil, err
		}
		orders[i].Items = items
	}
	return orders, nil
}

func (s *orderRepository) ListByStatus(ctx context.Context, status string, limit int) ([]domain.Order, error) {
	return s.listWithItems(ctx, `SELECT `+orderColumns+` FROM orders WHERE status = ?
		ORDER BY created_at DESC LIMIT ?`, status, limit)
}

func (s *orderRepository) ListRecent(ctx context.Context, limit int) ([]domain.Order, error) {
	return s.listWithItems(ctx, `SELECT `+orderColumns+` FROM orders
		ORDER BY created_at DESC LIMIT ?`, limit)
}

// ListByUserID mengembalikan riwayat pemesanan milik satu akun pengunjung.
func (s *orderRepository) ListByUserID(ctx context.Context, userID int64, limit int) ([]domain.Order, error) {
	return s.listWithItems(ctx, `SELECT `+orderColumns+` FROM orders WHERE user_id = ?
		ORDER BY created_at DESC LIMIT ?`, userID, limit)
}

// MarkPaid hanya berhasil bila order masih PENDING. Kondisi status pada klausa WHERE
// membuat operasi ini aman dari pembayaran ganda walau dipanggil bersamaan.
func (s *orderRepository) MarkPaid(ctx context.Context, code string, paidAt time.Time) error {
	res, err := s.db.ExecContext(ctx, `UPDATE orders SET status = ?, paid_at = ?
		WHERE code = ? AND status = ?`, domain.StatusPaid, paidAt, code, domain.StatusPending)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.ErrOrderNotPending
	}
	return nil
}

func (s *orderRepository) UpdateStatus(ctx context.Context, code, status string) error {
	res, err := s.db.ExecContext(ctx, `UPDATE orders SET status = ? WHERE code = ?`, status, code)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.ErrOrderNotFound
	}
	return nil
}

func (s *orderRepository) Stats(ctx context.Context) (*domain.Stats, error) {
	var st domain.Stats
	q := func(dest any, query string, args ...any) error {
		return s.db.QueryRowContext(ctx, query, args...).Scan(dest)
	}
	if err := q(&st.TotalRides, `SELECT COUNT(*) FROM rides`); err != nil {
		return nil, err
	}
	if err := q(&st.TotalOrders, `SELECT COUNT(*) FROM orders`); err != nil {
		return nil, err
	}
	if err := q(&st.PaidOrders, `SELECT COUNT(*) FROM orders WHERE status = ?`, domain.StatusPaid); err != nil {
		return nil, err
	}
	if err := q(&st.PendingOrders, `SELECT COUNT(*) FROM orders WHERE status = ?`, domain.StatusPending); err != nil {
		return nil, err
	}
	if err := q(&st.TotalTickets, `SELECT COUNT(*) FROM tickets`); err != nil {
		return nil, err
	}
	if err := q(&st.Revenue, `SELECT COALESCE(SUM(total_amount), 0) FROM orders WHERE status = ?`, domain.StatusPaid); err != nil {
		return nil, err
	}
	return &st, nil
}
