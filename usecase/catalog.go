package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"wahanapark/domain"
	"wahanapark/repository/rediscache"
)

// CatalogUsecase melayani pembacaan katalog wahana dan pengelolaannya oleh admin.
type CatalogUsecase struct {
	rides    domain.RideRepository
	quota    domain.QuotaStore
	cache    domain.Cache
	cacheTTL time.Duration
}

func NewCatalogUsecase(rides domain.RideRepository, quota domain.QuotaStore, cache domain.Cache, cacheTTL time.Duration) *CatalogUsecase {
	return &CatalogUsecase{rides: rides, quota: quota, cache: cache, cacheTTL: cacheTTL}
}

// List mengembalikan katalog wahana. Daftar dasar diambil dari cache Redis bila tersedia,
// sedangkan sisa kuota selalu dibaca langsung supaya angkanya tetap akurat.
func (u *CatalogUsecase) List(ctx context.Context, category, date string, activeOnly bool) ([]domain.Ride, error) {
	var rides []domain.Ride

	cacheKey := fmt.Sprintf("%s:%s:%t", rediscache.KeyRideCatalog, category, activeOnly)
	if raw, ok := u.cache.Get(ctx, cacheKey); ok {
		if err := json.Unmarshal(raw, &rides); err != nil {
			rides = nil
		}
	}
	if rides == nil {
		var err error
		rides, err = u.rides.List(ctx, category, activeOnly)
		if err != nil {
			return nil, err
		}
		if raw, err := json.Marshal(rides); err == nil {
			u.cache.Set(ctx, cacheKey, raw, u.cacheTTL)
		}
	}

	if date != "" {
		for i := range rides {
			avail, err := u.quota.Available(ctx, rides[i].ID, date, rides[i].DailyQuota)
			if err != nil {
				avail = int64(rides[i].DailyQuota)
			}
			rides[i].Available = avail
		}
	} else {
		for i := range rides {
			rides[i].Available = int64(rides[i].DailyQuota)
		}
	}
	return rides, nil
}

// GetBySlug mengembalikan detail satu wahana beserta sisa kuota pada tanggal tertentu.
func (u *CatalogUsecase) GetBySlug(ctx context.Context, slug, date string) (*domain.Ride, error) {
	ride, err := u.rides.GetBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if date != "" {
		if avail, err := u.quota.Available(ctx, ride.ID, date, ride.DailyQuota); err == nil {
			ride.Available = avail
		} else {
			ride.Available = int64(ride.DailyQuota)
		}
	} else {
		ride.Available = int64(ride.DailyQuota)
	}
	return ride, nil
}

// Categories mengembalikan daftar kategori beserta jumlah wahana pada masing masing kategori.
func (u *CatalogUsecase) Categories(ctx context.Context) ([]map[string]any, error) {
	rides, err := u.rides.List(ctx, "", true)
	if err != nil {
		return nil, err
	}
	counter := map[string]int{}
	for _, r := range rides {
		counter[r.Category]++
	}
	order := []string{domain.CategoryEkstrem, domain.CategoryKeluarga, domain.CategoryAnak,
		domain.CategoryAir, domain.CategoryPetualangan, domain.CategoryIndoor}

	out := make([]map[string]any, 0, len(order))
	for _, slug := range order {
		out = append(out, map[string]any{
			"slug":  slug,
			"label": domain.CategoryLabels[slug],
			"count": counter[slug],
		})
	}
	return out, nil
}

func (u *CatalogUsecase) invalidate(ctx context.Context) {
	keys := []string{rediscache.KeyStats}
	cats := []string{"", domain.CategoryEkstrem, domain.CategoryKeluarga, domain.CategoryAnak,
		domain.CategoryAir, domain.CategoryPetualangan, domain.CategoryIndoor}
	for _, c := range cats {
		keys = append(keys,
			fmt.Sprintf("%s:%s:%t", rediscache.KeyRideCatalog, c, true),
			fmt.Sprintf("%s:%s:%t", rediscache.KeyRideCatalog, c, false))
	}
	u.cache.Delete(ctx, keys...)
}

// Create menambah wahana baru ke katalog (admin).
func (u *CatalogUsecase) Create(ctx context.Context, r *domain.Ride) error {
	if r.Name == "" || r.Slug == "" || r.Price < 0 || r.DailyQuota < 0 {
		return domain.ErrInvalidInput
	}
	if _, ok := domain.CategoryLabels[r.Category]; !ok {
		return domain.ErrInvalidInput
	}
	if err := u.rides.Create(ctx, r); err != nil {
		return err
	}
	u.invalidate(ctx)
	return nil
}

// Update memperbarui data wahana (admin).
func (u *CatalogUsecase) Update(ctx context.Context, r *domain.Ride) error {
	if r.Name == "" || r.Price < 0 || r.DailyQuota < 0 {
		return domain.ErrInvalidInput
	}
	if _, ok := domain.CategoryLabels[r.Category]; !ok {
		return domain.ErrInvalidInput
	}
	if err := u.rides.Update(ctx, r); err != nil {
		return err
	}
	u.invalidate(ctx)
	return nil
}

// Delete menghapus wahana, kecuali bila wahana tersebut sudah pernah masuk order.
func (u *CatalogUsecase) Delete(ctx context.Context, id int64) error {
	used, err := u.rides.CountOrderItems(ctx, id)
	if err != nil {
		return err
	}
	if used > 0 {
		return domain.ErrRideHasOrders
	}
	if err := u.rides.Delete(ctx, id); err != nil {
		return err
	}
	u.invalidate(ctx)
	return nil
}

func (u *CatalogUsecase) GetByID(ctx context.Context, id int64) (*domain.Ride, error) {
	return u.rides.GetByID(ctx, id)
}
