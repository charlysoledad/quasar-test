package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"quasar-ecommerce/api/db"
)

// TenantHandler handles tenant-related requests
type TenantHandler struct {
	DB *pgxpool.Pool
}

func (h *TenantHandler) CreateTenant(w http.ResponseWriter, r *http.Request) {
	var req db.CreateTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	tenant := db.Tenant{
		ID:        uuid.New(),
		Name:      req.Name,
		Slug:      req.Slug,
		Status:    "active",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	query := `
		INSERT INTO tenants (id, name, slug, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, name, slug, status, created_at, updated_at
	`

	err := h.DB.QueryRow(context.Background(), query,
		tenant.ID, tenant.Name, tenant.Slug, tenant.Status, tenant.CreatedAt, tenant.UpdatedAt,
	).Scan(&tenant.ID, &tenant.Name, &tenant.Slug, &tenant.Status, &tenant.CreatedAt, &tenant.UpdatedAt)

	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create tenant: "+err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, tenant)
}

func (h *TenantHandler) ListTenants(w http.ResponseWriter, r *http.Request) {
	query := `SELECT id, name, slug, status, created_at, updated_at FROM tenants ORDER BY created_at DESC`
	rows, err := h.DB.Query(context.Background(), query)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch tenants")
		return
	}
	defer rows.Close()

	var tenants []db.Tenant
	for rows.Next() {
		var t db.Tenant
		if err := rows.Scan(&t.ID, &t.Name, &t.Slug, &t.Status, &t.CreatedAt, &t.UpdatedAt); err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to scan tenant")
			return
		}
		tenants = append(tenants, t)
	}

	respondJSON(w, http.StatusOK, tenants)
}

func (h *TenantHandler) GetTenant(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid tenant ID")
		return
	}

	var tenant db.Tenant
	query := `SELECT id, name, slug, status, created_at, updated_at FROM tenants WHERE id = $1`
	err = h.DB.QueryRow(context.Background(), query, id).Scan(
		&tenant.ID, &tenant.Name, &tenant.Slug, &tenant.Status, &tenant.CreatedAt, &tenant.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		respondError(w, http.StatusNotFound, "Tenant not found")
		return
	} else if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch tenant")
		return
	}

	respondJSON(w, http.StatusOK, tenant)
}

func (h *TenantHandler) UpdateTenant(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid tenant ID")
		return
	}

	var req db.CreateTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	query := `
		UPDATE tenants SET name = $1, slug = $2, updated_at = $3
		WHERE id = $4
		RETURNING id, name, slug, status, created_at, updated_at
	`

	var tenant db.Tenant
	err = h.DB.QueryRow(context.Background(), query, req.Name, req.Slug, time.Now(), id).Scan(
		&tenant.ID, &tenant.Name, &tenant.Slug, &tenant.Status, &tenant.CreatedAt, &tenant.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		respondError(w, http.StatusNotFound, "Tenant not found")
		return
	} else if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update tenant")
		return
	}

	respondJSON(w, http.StatusOK, tenant)
}

func (h *TenantHandler) DeleteTenant(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid tenant ID")
		return
	}

	query := `DELETE FROM tenants WHERE id = $1`
	result, err := h.DB.Exec(context.Background(), query, id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to delete tenant")
		return
	}

	if result.RowsAffected() == 0 {
		respondError(w, http.StatusNotFound, "Tenant not found")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Tenant deleted successfully"})
}

// StoreHandler handles store-related requests
type StoreHandler struct {
	DB *pgxpool.Pool
}

func NewStoreHandler(db *pgxpool.Pool) *StoreHandler {
	return &StoreHandler{DB: db}
}

func (h *StoreHandler) CreateStore(w http.ResponseWriter, r *http.Request) {
	var req db.CreateStoreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	store := db.Store{
		ID:              uuid.New(),
		TenantID:        req.TenantID,
		Name:            req.Name,
		Slug:            req.Slug,
		DefaultCurrency: req.DefaultCurrency,
		Timezone:        req.Timezone,
		IsActive:        true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	query := `
		INSERT INTO stores (id, tenant_id, name, slug, default_currency, timezone, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, tenant_id, name, slug, default_currency, timezone, is_active, created_at, updated_at
	`

	err := h.DB.QueryRow(context.Background(), query,
		store.ID, store.TenantID, store.Name, store.Slug, store.DefaultCurrency,
		store.Timezone, store.IsActive, store.CreatedAt, store.UpdatedAt,
	).Scan(&store.ID, &store.TenantID, &store.Name, &store.Slug, &store.DefaultCurrency,
		&store.Timezone, &store.IsActive, &store.CreatedAt, &store.UpdatedAt)

	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create store: "+err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, store)
}

func (h *StoreHandler) ListStores(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")

	var query string
	var rows pgx.Rows
	var err error

	if tenantID != "" {
		query = `SELECT id, tenant_id, name, slug, default_currency, timezone, is_active, created_at, updated_at 
				FROM stores WHERE tenant_id = $1 ORDER BY created_at DESC`
		rows, err = h.DB.Query(context.Background(), query, tenantID)
	} else {
		query = `SELECT id, tenant_id, name, slug, default_currency, timezone, is_active, created_at, updated_at 
				FROM stores ORDER BY created_at DESC`
		rows, err = h.DB.Query(context.Background(), query)
	}

	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch stores")
		return
	}
	defer rows.Close()

	var stores []db.Store
	for rows.Next() {
		var s db.Store
		if err := rows.Scan(&s.ID, &s.TenantID, &s.Name, &s.Slug, &s.DefaultCurrency,
			&s.Timezone, &s.IsActive, &s.CreatedAt, &s.UpdatedAt); err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to scan store")
			return
		}
		stores = append(stores, s)
	}

	respondJSON(w, http.StatusOK, stores)
}

func (h *StoreHandler) GetStore(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid store ID")
		return
	}

	var store db.Store
	query := `SELECT id, tenant_id, name, slug, default_currency, timezone, is_active, created_at, updated_at 
			  FROM stores WHERE id = $1`
	err = h.DB.QueryRow(context.Background(), query, id).Scan(
		&store.ID, &store.TenantID, &store.Name, &store.Slug, &store.DefaultCurrency,
		&store.Timezone, &store.IsActive, &store.CreatedAt, &store.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		respondError(w, http.StatusNotFound, "Store not found")
		return
	} else if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch store")
		return
	}

	respondJSON(w, http.StatusOK, store)
}

func (h *StoreHandler) UpdateStore(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid store ID")
		return
	}

	var req db.CreateStoreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	query := `
		UPDATE stores SET name = $1, slug = $2, default_currency = $3, timezone = $4, updated_at = $5
		WHERE id = $6
		RETURNING id, tenant_id, name, slug, default_currency, timezone, is_active, created_at, updated_at
	`

	var store db.Store
	err = h.DB.QueryRow(context.Background(), query, req.Name, req.Slug, req.DefaultCurrency,
		req.Timezone, time.Now(), id).Scan(
		&store.ID, &store.TenantID, &store.Name, &store.Slug, &store.DefaultCurrency,
		&store.Timezone, &store.IsActive, &store.CreatedAt, &store.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		respondError(w, http.StatusNotFound, "Store not found")
		return
	} else if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update store")
		return
	}

	respondJSON(w, http.StatusOK, store)
}

// Helper functions
func respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}
