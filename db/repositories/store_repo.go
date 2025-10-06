package repositories

import (
	"context"
	"database/sql"
	"time"
)

type Store struct {
	ID        string
	TenantID  string
	Name      string
	Slug      string
	IsActive  bool
	CreatedAt time.Time
}

type StoreRepo struct {
	DB *sql.DB
}

func (r *StoreRepo) Create(ctx context.Context, tenantID, name, slug string) (string, error) {
	var id string
	query := `
		INSERT INTO stores (tenant_id, name, slug)
		VALUES ($1, $2, $3)
		RETURNING id;
	`
	err := r.DB.QueryRowContext(ctx, query, tenantID, name, slug).Scan(&id)
	return id, err
}

func (r *StoreRepo) ListByTenant(ctx context.Context, tenantID string) ([]Store, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT id, tenant_id, name, slug, is_active, created_at FROM stores WHERE tenant_id = $1`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stores []Store
	for rows.Next() {
		var s Store
		if err := rows.Scan(&s.ID, &s.TenantID, &s.Name, &s.Slug, &s.IsActive, &s.CreatedAt); err != nil {
			return nil, err
		}
		stores = append(stores, s)
	}
	return stores, nil
}
