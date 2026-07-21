package sqlite

import (
	"context"
	"database/sql"
	"errors"

	"wahanapark/domain"
)

type rideRepository struct{ db *sql.DB }

func NewRideRepository(db *sql.DB) domain.RideRepository { return &rideRepository{db: db} }

const rideColumns = `id, slug, name, category, tagline, description, emoji, image_url, price,
	duration_min, min_height_cm, thrill_level, daily_quota, is_active`

func scanRide(row interface{ Scan(...any) error }) (*domain.Ride, error) {
	var r domain.Ride
	var active int
	err := row.Scan(&r.ID, &r.Slug, &r.Name, &r.Category, &r.Tagline, &r.Description, &r.Emoji,
		&r.ImageURL, &r.Price, &r.DurationMin, &r.MinHeightCm, &r.ThrillLevel, &r.DailyQuota, &active)
	if err != nil {
		return nil, err
	}
	r.IsActive = active == 1
	return &r, nil
}

func (s *rideRepository) List(ctx context.Context, category string, activeOnly bool) ([]domain.Ride, error) {
	query := `SELECT ` + rideColumns + ` FROM rides WHERE 1=1`
	args := []any{}
	if category != "" {
		query += ` AND category = ?`
		args = append(args, category)
	}
	if activeOnly {
		query += ` AND is_active = 1`
	}
	query += ` ORDER BY thrill_level DESC, name ASC`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rides := make([]domain.Ride, 0, 32)
	for rows.Next() {
		r, err := scanRide(rows)
		if err != nil {
			return nil, err
		}
		rides = append(rides, *r)
	}
	return rides, rows.Err()
}

func (s *rideRepository) GetBySlug(ctx context.Context, slug string) (*domain.Ride, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+rideColumns+` FROM rides WHERE slug = ?`, slug)
	r, err := scanRide(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrRideNotFound
	}
	return r, err
}

func (s *rideRepository) GetByID(ctx context.Context, id int64) (*domain.Ride, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+rideColumns+` FROM rides WHERE id = ?`, id)
	r, err := scanRide(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrRideNotFound
	}
	return r, err
}

func (s *rideRepository) Create(ctx context.Context, r *domain.Ride) error {
	res, err := s.db.ExecContext(ctx, `INSERT INTO rides
		(slug, name, category, tagline, description, emoji, image_url, price, duration_min, min_height_cm, thrill_level, daily_quota, is_active)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.Slug, r.Name, r.Category, r.Tagline, r.Description, r.Emoji, r.ImageURL, r.Price,
		r.DurationMin, r.MinHeightCm, r.ThrillLevel, r.DailyQuota, boolToInt(r.IsActive))
	if err != nil {
		return err
	}
	r.ID, _ = res.LastInsertId()
	return nil
}

func (s *rideRepository) Update(ctx context.Context, r *domain.Ride) error {
	res, err := s.db.ExecContext(ctx, `UPDATE rides SET
		name = ?, category = ?, tagline = ?, description = ?, emoji = ?, image_url = ?, price = ?,
		duration_min = ?, min_height_cm = ?, thrill_level = ?, daily_quota = ?, is_active = ?
		WHERE id = ?`,
		r.Name, r.Category, r.Tagline, r.Description, r.Emoji, r.ImageURL, r.Price,
		r.DurationMin, r.MinHeightCm, r.ThrillLevel, r.DailyQuota, boolToInt(r.IsActive), r.ID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.ErrRideNotFound
	}
	return nil
}

func (s *rideRepository) Delete(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM rides WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.ErrRideNotFound
	}
	return nil
}

// CountOrderItems dipakai untuk memproteksi penghapusan wahana yang sudah pernah dipesan,
// supaya riwayat order dan tiket lama tidak kehilangan acuan wahananya.
func (s *rideRepository) CountOrderItems(ctx context.Context, rideID int64) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM order_items WHERE ride_id = ?`, rideID).Scan(&n)
	return n, err
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
