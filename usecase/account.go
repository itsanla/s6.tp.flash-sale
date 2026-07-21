package usecase

import (
	"context"
	"strings"
	"time"

	"wahanapark/domain"

	"golang.org/x/crypto/bcrypt"
)

// AccountUsecase menangani pendaftaran, masuk akun, dan profil pengunjung.
type AccountUsecase struct {
	users  domain.UserRepository
	orders domain.OrderRepository
}

func NewAccountUsecase(users domain.UserRepository, orders domain.OrderRepository) *AccountUsecase {
	return &AccountUsecase{users: users, orders: orders}
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// Register membuat akun baru. Password disimpan dalam bentuk hash bcrypt, tidak pernah
// disimpan apa adanya.
func (u *AccountUsecase) Register(ctx context.Context, name, email, phone, password string) (*domain.User, error) {
	name = strings.TrimSpace(name)
	email = normalizeEmail(email)
	if name == "" || email == "" || !strings.Contains(email, "@") {
		return nil, domain.ErrInvalidInput
	}
	if len(password) < 6 {
		return nil, domain.ErrPasswordTooShort
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	user := &domain.User{
		Name:         name,
		Email:        email,
		Phone:        strings.TrimSpace(phone),
		PasswordHash: string(hash),
		CreatedAt:    time.Now(),
	}
	if err := u.users.Create(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

// Login memeriksa kredensial pengunjung. Pesan galat sengaja dibuat sama untuk email
// yang tidak terdaftar maupun password keliru, agar tidak membocorkan email mana yang ada.
func (u *AccountUsecase) Login(ctx context.Context, email, password string) (*domain.User, error) {
	user, err := u.users.GetByEmail(ctx, normalizeEmail(email))
	if err == domain.ErrUserNotFound {
		return nil, domain.ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, domain.ErrInvalidCredentials
	}
	return user, nil
}

func (u *AccountUsecase) Profile(ctx context.Context, id int64) (*domain.User, error) {
	return u.users.GetByID(ctx, id)
}

func (u *AccountUsecase) UpdateProfile(ctx context.Context, id int64, name, phone string) (*domain.User, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, domain.ErrInvalidInput
	}
	if err := u.users.UpdateProfile(ctx, id, name, strings.TrimSpace(phone)); err != nil {
		return nil, err
	}
	return u.users.GetByID(ctx, id)
}

// MyOrders mengembalikan riwayat pemesanan milik akun tersebut.
func (u *AccountUsecase) MyOrders(ctx context.Context, id int64, limit int) ([]domain.Order, error) {
	return u.orders.ListByUserID(ctx, id, limit)
}
