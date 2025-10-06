// models.go
package db

import (
	"time"

	"github.com/google/uuid"
)

// Tenant represents a merchant/organization
type Tenant struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Store represents a store within a tenant
type Store struct {
	ID              uuid.UUID `json:"id"`
	TenantID        uuid.UUID `json:"tenant_id"`
	Name            string    `json:"name"`
	Slug            string    `json:"slug"`
	DefaultCurrency string    `json:"default_currency"`
	Timezone        string    `json:"timezone"`
	IsActive        bool      `json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// Product represents a product in the store
type Product struct {
	ID          uuid.UUID `json:"id"`
	StoreID     uuid.UUID `json:"store_id"`
	SKU         string    `json:"sku"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	PriceCents  int       `json:"price_cents"`
	Currency    string    `json:"currency"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Storage represents a warehouse or branch location
type Storage struct {
	ID         uuid.UUID `json:"id"`
	StoreID    uuid.UUID `json:"store_id"`
	Name       string    `json:"name"`
	Type       string    `json:"type"`
	Address    string    `json:"address"`
	City       string    `json:"city"`
	State      string    `json:"state"`
	Country    string    `json:"country"`
	PostalCode string    `json:"postal_code"`
	IsPrimary  bool      `json:"is_primary"`
	IsActive   bool      `json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// StockItem represents product quantity at a specific storage location
type StockItem struct {
	ID               uuid.UUID `json:"id"`
	StorageID        uuid.UUID `json:"storage_id"`
	ProductID        uuid.UUID `json:"product_id"`
	Quantity         int       `json:"quantity"`
	ReservedQuantity int       `json:"reserved_quantity"`
}

// InventoryMovement tracks stock changes
type InventoryMovement struct {
	ID           uuid.UUID  `json:"id"`
	StorageID    *uuid.UUID `json:"storage_id"`
	ProductID    *uuid.UUID `json:"product_id"`
	MovementType string     `json:"movement_type"`
	Quantity     int        `json:"quantity"`
	ReferenceID  *uuid.UUID `json:"reference_id"`
	Note         string     `json:"note"`
	CreatedAt    time.Time  `json:"created_at"`
}

// Order represents a customer order
type Order struct {
	ID            uuid.UUID `json:"id"`
	StoreID       uuid.UUID `json:"store_id"`
	CustomerName  string    `json:"customer_name"`
	CustomerEmail string    `json:"customer_email"`
	TotalCents    int       `json:"total_cents"`
	Currency      string    `json:"currency"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// OrderItem represents a product in an order
type OrderItem struct {
	ID         uuid.UUID  `json:"id"`
	OrderID    uuid.UUID  `json:"order_id"`
	ProductID  *uuid.UUID `json:"product_id"`
	Quantity   int        `json:"quantity"`
	PriceCents int        `json:"price_cents"`
}

// FulfillmentOrder links orders to storage locations
type FulfillmentOrder struct {
	ID             uuid.UUID  `json:"id"`
	OrderID        uuid.UUID  `json:"order_id"`
	StorageID      *uuid.UUID `json:"storage_id"`
	Status         string     `json:"status"`
	TrackingNumber string     `json:"tracking_number"`
	ShippedAt      *time.Time `json:"shipped_at"`
	DeliveredAt    *time.Time `json:"delivered_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// User represents a system user
type User struct {
	ID              uuid.UUID `json:"id"`
	Email           string    `json:"email"`
	PasswordHash    string    `json:"-"` // Never expose password hash
	FullName        string    `json:"full_name"`
	IsPlatformAdmin bool      `json:"is_platform_admin"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// Plan represents a subscription plan
type Plan struct {
	ID         uuid.UUID              `json:"id"`
	Name       string                 `json:"name"`
	PriceCents int                    `json:"price_cents"`
	Currency   string                 `json:"currency"`
	Features   map[string]interface{} `json:"features"`
	CreatedAt  time.Time              `json:"created_at"`
}

// Subscription represents a tenant's subscription to a plan
type Subscription struct {
	ID                   uuid.UUID  `json:"id"`
	TenantID             uuid.UUID  `json:"tenant_id"`
	PlanID               uuid.UUID  `json:"plan_id"`
	StartDate            *time.Time `json:"start_date"`
	EndDate              *time.Time `json:"end_date"`
	Status               string     `json:"status"`
	StripeSubscriptionID string     `json:"stripe_subscription_id"`
	CreatedAt            time.Time  `json:"created_at"`
}

// CreateTenantRequest represents the request body for creating a tenant
type CreateTenantRequest struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// CreateStoreRequest represents the request body for creating a store
type CreateStoreRequest struct {
	TenantID        uuid.UUID `json:"tenant_id"`
	Name            string    `json:"name"`
	Slug            string    `json:"slug"`
	DefaultCurrency string    `json:"default_currency"`
	Timezone        string    `json:"timezone"`
}

// CreateProductRequest represents the request body for creating a product
type CreateProductRequest struct {
	StoreID     uuid.UUID `json:"store_id"`
	SKU         string    `json:"sku"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	PriceCents  int       `json:"price_cents"`
	Currency    string    `json:"currency"`
}

// CreateStorageRequest represents the request body for creating a storage location
type CreateStorageRequest struct {
	StoreID    uuid.UUID `json:"store_id"`
	Name       string    `json:"name"`
	Type       string    `json:"type"`
	Address    string    `json:"address"`
	City       string    `json:"city"`
	State      string    `json:"state"`
	Country    string    `json:"country"`
	PostalCode string    `json:"postal_code"`
	IsPrimary  bool      `json:"is_primary"`
}

// UpdateStockRequest represents the request body for updating stock
type UpdateStockRequest struct {
	StorageID    uuid.UUID `json:"storage_id"`
	ProductID    uuid.UUID `json:"product_id"`
	Quantity     int       `json:"quantity"`
	MovementType string    `json:"movement_type"`
	Note         string    `json:"note"`
}

// TransferStockRequest represents the request body for transferring stock
type TransferStockRequest struct {
	FromStorageID uuid.UUID `json:"from_storage_id"`
	ToStorageID   uuid.UUID `json:"to_storage_id"`
	ProductID     uuid.UUID `json:"product_id"`
	Quantity      int       `json:"quantity"`
	Note          string    `json:"note"`
}

// CreateOrderRequest represents the request body for creating an order
type CreateOrderRequest struct {
	StoreID       uuid.UUID         `json:"store_id"`
	CustomerName  string            `json:"customer_name"`
	CustomerEmail string            `json:"customer_email"`
	Items         []CreateOrderItem `json:"items"`
	StorageID     uuid.UUID         `json:"storage_id"`
}

// CreateOrderItem represents an item in a new order
type CreateOrderItem struct {
	ProductID uuid.UUID `json:"product_id"`
	Quantity  int       `json:"quantity"`
}
