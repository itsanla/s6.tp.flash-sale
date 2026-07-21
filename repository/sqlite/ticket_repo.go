package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"wahanapark/domain"
)

type ticketRepository struct{ db *sql.DB }

func NewTicketRepository(db *sql.DB) domain.TicketRepository { return &ticketRepository{db: db} }

func (s *ticketRepository) CreateBatch(ctx context.Context, tickets []domain.Ticket) error {
	if len(tickets) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `INSERT INTO tickets
		(code, order_id, order_code, ride_id, ride_name, ride_emoji, visit_date, status, issued_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, t := range tickets {
		if _, err := stmt.ExecContext(ctx, t.Code, t.OrderID, t.OrderCode, t.RideID,
			t.RideName, t.RideEmoji, t.VisitDate, t.Status, t.IssuedAt); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *ticketRepository) ListByOrderCode(ctx context.Context, orderCode string) ([]domain.Ticket, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, code, order_id, order_code, ride_id, ride_name,
		ride_emoji, visit_date, status, issued_at, used_at FROM tickets WHERE order_code = ? ORDER BY id`, orderCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tickets := make([]domain.Ticket, 0, 8)
	for rows.Next() {
		t, err := scanTicket(rows)
		if err != nil {
			return nil, err
		}
		tickets = append(tickets, *t)
	}
	return tickets, rows.Err()
}

// CountByOrderID dipakai worker untuk memastikan tiket sebuah order hanya terbit sekali,
// walau pesan penerbitan tiket sempat dikirim ulang oleh RabbitMQ.
func (s *ticketRepository) CountByOrderID(ctx context.Context, orderID int64) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM tickets WHERE order_id = ?`, orderID).Scan(&n)
	return n, err
}

func (s *ticketRepository) GetByCode(ctx context.Context, code string) (*domain.Ticket, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, code, order_id, order_code, ride_id, ride_name,
		ride_emoji, visit_date, status, issued_at, used_at FROM tickets WHERE code = ?`, code)
	t, err := scanTicket(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrTicketNotFound
	}
	return t, err
}

func (s *ticketRepository) MarkUsed(ctx context.Context, code string, usedAt time.Time) error {
	res, err := s.db.ExecContext(ctx, `UPDATE tickets SET status = ?, used_at = ?
		WHERE code = ? AND status = ?`, domain.TicketUsed, usedAt, code, domain.TicketIssued)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.ErrTicketUsed
	}
	return nil
}

func scanTicket(row interface{ Scan(...any) error }) (*domain.Ticket, error) {
	var t domain.Ticket
	var usedAt sql.NullTime
	err := row.Scan(&t.ID, &t.Code, &t.OrderID, &t.OrderCode, &t.RideID, &t.RideName,
		&t.RideEmoji, &t.VisitDate, &t.Status, &t.IssuedAt, &usedAt)
	if err != nil {
		return nil, err
	}
	if usedAt.Valid {
		u := usedAt.Time
		t.UsedAt = &u
	}
	return &t, nil
}
