package repositories

import (
	"context"
	"database/sql"
	"time"
)

type Storage struct {
	ID        string
	StoreID   string
	Name      string
	City      string
	IsPrimary bool
	IsActive  bool
	CreatedAt time.Time
}

type StorageRepo struct {
	DB *sql.DB
}

func (r *StorageRepo) Create(ctx context.Context, storeID, name, city string, isPrimary bool) (string, error) {
	var id string
	query := `
		INSERT INTO storages (store_id, name, city, is_primary)
		VALUES ($1, $2, $3, $4)
		RETURNING id;
	`
	err := r.DB.QueryRowContext(ctx, query, storeID, name, city, isPrimary).Scan(&id)
	return id, err
}

func (r *StorageRepo) ListByStore(ctx context.Context, storeID string) ([]Storage, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT id, store_id, name, city, is_primary, is_active, created_at FROM storages WHERE store_id = $1`, storeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var storages []Storage
	for rows.Next() {
		var s Storage
		if err := rows.Scan(&s.ID, &s.StoreID, &s.Name, &s.City, &s.IsPrimary, &s.IsActive, &s.CreatedAt); err != nil {
			return nil, err
		}
		storages = append(storages, s)
	}
	return storages, nil
}
