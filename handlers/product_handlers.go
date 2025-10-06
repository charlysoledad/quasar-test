// product_handlers.go
package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"quasar-ecommerce/api/db"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ProductHandler handles product-related requests
type ProductHandler struct {
	DB *pgxpool.Pool
}

func NewProductHandler(DB *pgxpool.Pool) *ProductHandler {
	return &ProductHandler{DB: DB}
}

func (h *ProductHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	var req db.CreateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	product := db.Product{
		ID:          uuid.New(),
		StoreID:     req.StoreID,
		SKU:         req.SKU,
		Name:        req.Name,
		Description: req.Description,
		PriceCents:  req.PriceCents,
		Currency:    req.Currency,
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	query := `
		INSERT INTO products (id, store_id, sku, name, description, price_cents, currency, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, store_id, sku, name, description, price_cents, currency, is_active, created_at, updated_at
	`

	err := h.DB.QueryRow(context.Background(), query,
		product.ID, product.StoreID, product.SKU, product.Name, product.Description,
		product.PriceCents, product.Currency, product.IsActive, product.CreatedAt, product.UpdatedAt,
	).Scan(&product.ID, &product.StoreID, &product.SKU, &product.Name, &product.Description,
		&product.PriceCents, &product.Currency, &product.IsActive, &product.CreatedAt, &product.UpdatedAt)

	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create product: "+err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, product)
}

func (h *ProductHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	storeID := r.URL.Query().Get("store_id")

	var query string
	var rows pgx.Rows
	var err error

	if storeID != "" {
		query = `SELECT id, store_id, sku, name, description, price_cents, currency, is_active, created_at, updated_at 
				FROM products WHERE store_id = $1 ORDER BY created_at DESC`
		rows, err = h.DB.Query(context.Background(), query, storeID)
	} else {
		query = `SELECT id, store_id, sku, name, description, price_cents, currency, is_active, created_at, updated_at 
				FROM products ORDER BY created_at DESC`
		rows, err = h.DB.Query(context.Background(), query)
	}

	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch products")
		return
	}
	defer rows.Close()

	var products []db.Product
	for rows.Next() {
		var p db.Product
		if err := rows.Scan(&p.ID, &p.StoreID, &p.SKU, &p.Name, &p.Description,
			&p.PriceCents, &p.Currency, &p.IsActive, &p.CreatedAt, &p.UpdatedAt); err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to scan product")
			return
		}
		products = append(products, p)
	}

	respondJSON(w, http.StatusOK, products)
}

func (h *ProductHandler) GetProduct(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid product ID")
		return
	}

	var product db.Product
	query := `SELECT id, store_id, sku, name, description, price_cents, currency, is_active, created_at, updated_at 
			  FROM products WHERE id = $1`
	err = h.DB.QueryRow(context.Background(), query, id).Scan(
		&product.ID, &product.StoreID, &product.SKU, &product.Name, &product.Description,
		&product.PriceCents, &product.Currency, &product.IsActive, &product.CreatedAt, &product.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		respondError(w, http.StatusNotFound, "Product not found")
		return
	} else if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch product")
		return
	}

	respondJSON(w, http.StatusOK, product)
}

func (h *ProductHandler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid product ID")
		return
	}

	var req db.CreateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	query := `
		UPDATE products SET sku = $1, name = $2, description = $3, price_cents = $4, currency = $5, updated_at = $6
		WHERE id = $7
		RETURNING id, store_id, sku, name, description, price_cents, currency, is_active, created_at, updated_at
	`

	var product db.Product
	err = h.DB.QueryRow(context.Background(), query, req.SKU, req.Name, req.Description,
		req.PriceCents, req.Currency, time.Now(), id).Scan(
		&product.ID, &product.StoreID, &product.SKU, &product.Name, &product.Description,
		&product.PriceCents, &product.Currency, &product.IsActive, &product.CreatedAt, &product.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		respondError(w, http.StatusNotFound, "Product not found")
		return
	} else if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update product")
		return
	}

	respondJSON(w, http.StatusOK, product)
}

func (h *ProductHandler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid product ID")
		return
	}

	query := `DELETE FROM products WHERE id = $1`
	result, err := h.DB.Exec(context.Background(), query, id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to delete product")
		return
	}

	if result.RowsAffected() == 0 {
		respondError(w, http.StatusNotFound, "Product not found")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Product deleted successfully"})
}

// StorageHandler handles warehouse/storage location requests
type StorageHandler struct {
	DB *pgxpool.Pool
}

func NewStorageHandler(DB *pgxpool.Pool) *StorageHandler {
	return &StorageHandler{DB: DB}
}

func (h *StorageHandler) CreateStorage(w http.ResponseWriter, r *http.Request) {
	var req db.CreateStorageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	storage := db.Storage{
		ID:         uuid.New(),
		StoreID:    req.StoreID,
		Name:       req.Name,
		Type:       req.Type,
		Address:    req.Address,
		City:       req.City,
		State:      req.State,
		Country:    req.Country,
		PostalCode: req.PostalCode,
		IsPrimary:  req.IsPrimary,
		IsActive:   true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	query := `
		INSERT INTO storages (id, store_id, name, type, address, city, state, country, postal_code, is_primary, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id, store_id, name, type, address, city, state, country, postal_code, is_primary, is_active, created_at, updated_at
	`

	err := h.DB.QueryRow(context.Background(), query,
		storage.ID, storage.StoreID, storage.Name, storage.Type, storage.Address,
		storage.City, storage.State, storage.Country, storage.PostalCode,
		storage.IsPrimary, storage.IsActive, storage.CreatedAt, storage.UpdatedAt,
	).Scan(&storage.ID, &storage.StoreID, &storage.Name, &storage.Type, &storage.Address,
		&storage.City, &storage.State, &storage.Country, &storage.PostalCode,
		&storage.IsPrimary, &storage.IsActive, &storage.CreatedAt, &storage.UpdatedAt)

	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create storage: "+err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, storage)
}

func (h *StorageHandler) ListStorages(w http.ResponseWriter, r *http.Request) {
	storeID := r.URL.Query().Get("store_id")

	var query string
	var rows pgx.Rows
	var err error

	if storeID != "" {
		query = `SELECT id, store_id, name, type, address, city, state, country, postal_code, is_primary, is_active, created_at, updated_at 
				FROM storages WHERE store_id = $1 ORDER BY created_at DESC`
		rows, err = h.DB.Query(context.Background(), query, storeID)
	} else {
		query = `SELECT id, store_id, name, type, address, city, state, country, postal_code, is_primary, is_active, created_at, updated_at 
				FROM storages ORDER BY created_at DESC`
		rows, err = h.DB.Query(context.Background(), query)
	}

	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch storages")
		return
	}
	defer rows.Close()

	var storages []db.Storage
	for rows.Next() {
		var s db.Storage
		if err := rows.Scan(&s.ID, &s.StoreID, &s.Name, &s.Type, &s.Address,
			&s.City, &s.State, &s.Country, &s.PostalCode,
			&s.IsPrimary, &s.IsActive, &s.CreatedAt, &s.UpdatedAt); err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to scan storage")
			return
		}
		storages = append(storages, s)
	}

	respondJSON(w, http.StatusOK, storages)
}

func (h *StorageHandler) GetStorage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid storage ID")
		return
	}

	var storage db.Storage
	query := `SELECT id, store_id, name, type, address, city, state, country, postal_code, is_primary, is_active, created_at, updated_at 
			  FROM storages WHERE id = $1`
	err = h.DB.QueryRow(context.Background(), query, id).Scan(
		&storage.ID, &storage.StoreID, &storage.Name, &storage.Type, &storage.Address,
		&storage.City, &storage.State, &storage.Country, &storage.PostalCode,
		&storage.IsPrimary, &storage.IsActive, &storage.CreatedAt, &storage.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		respondError(w, http.StatusNotFound, "Storage not found")
		return
	} else if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch storage")
		return
	}

	respondJSON(w, http.StatusOK, storage)
}

func (h *StorageHandler) UpdateStorage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid storage ID")
		return
	}

	var req db.CreateStorageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	query := `
		UPDATE storages SET name = $1, type = $2, address = $3, city = $4, state = $5, country = $6, postal_code = $7, updated_at = $8
		WHERE id = $9
		RETURNING id, store_id, name, type, address, city, state, country, postal_code, is_primary, is_active, created_at, updated_at
	`

	var storage db.Storage
	err = h.DB.QueryRow(context.Background(), query, req.Name, req.Type, req.Address,
		req.City, req.State, req.Country, req.PostalCode, time.Now(), id).Scan(
		&storage.ID, &storage.StoreID, &storage.Name, &storage.Type, &storage.Address,
		&storage.City, &storage.State, &storage.Country, &storage.PostalCode,
		&storage.IsPrimary, &storage.IsActive, &storage.CreatedAt, &storage.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		respondError(w, http.StatusNotFound, "Storage not found")
		return
	} else if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update storage")
		return
	}

	respondJSON(w, http.StatusOK, storage)
}
