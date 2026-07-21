package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"wahanapark/domain"
)

type userRepository struct{ db *sql.DB }

func NewUserRepository(db *sql.DB) domain.UserRepository { return &userRepository{db: db} }

const userColumns = `id, name, email, phone, password_hash, created_at`

func scanUser(row interface{ Scan(...any) error }) (*domain.User, error) {
	var u domain.User
	err := row.Scan(&u.ID, &u.Name, &u.Email, &u.Phone, &u.PasswordHash, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *userRepository) Create(ctx context.Context, u *domain.User) error {
	res, err := s.db.ExecContext(ctx, `INSERT INTO users
		(name, email, phone, password_hash, created_at) VALUES (?, ?, ?, ?, ?)`,
		u.Name, u.Email, u.Phone, u.PasswordHash, u.CreatedAt)
	if err != nil {
		// Kolom email bersifat unik, sehingga pendaftaran ulang ditolak basis data.
		if strings.Contains(strings.ToUpper(err.Error()), "UNIQUE") {
			return domain.ErrEmailTaken
		}
		return err
	}
	u.ID, _ = res.LastInsertId()
	return nil
}

func (s *userRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+userColumns+` FROM users WHERE email = ?`, email)
	u, err := scanUser(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}
	return u, err
}

func (s *userRepository) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+userColumns+` FROM users WHERE id = ?`, id)
	u, err := scanUser(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrUserNotFound
	}
	return u, err
}

func (s *userRepository) UpdateProfile(ctx context.Context, id int64, name, phone string) error {
	res, err := s.db.ExecContext(ctx, `UPDATE users SET name = ?, phone = ? WHERE id = ?`, name, phone, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}
