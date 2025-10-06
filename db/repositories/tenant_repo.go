package repositories

import (
	"context"
	"database/sql"
	"time"
)

type Tenant struct {
	ID        string
	Name      string
	Slug      string
	Status    string
	CreatedAt time.Time
}

type TenantRepo struct {
	DB *sql.DB
}

func (r *TenantRepo) Create(ctx context.Context, name, slug string) (string, error) {
	var id string
	query := `
		INSERT INTO tenants (name, slug)
		VALUES ($1, $2)
		RETURNING id;
	`
	err := r.DB.QueryRowContext(ctx, query, name, slug).Scan(&id)
	return id, err
}

func (r *TenantRepo) GetByID(ctx context.Context, id string) (*Tenant, error) {
	row := r.DB.QueryRowContext(ctx, `SELECT id, name, slug, status, created_at FROM tenants WHERE id = $1`, id)
	var t Tenant
	err := row.Scan(&t.ID, &t.Name, &t.Slug, &t.Status, &t.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &t, err
}
