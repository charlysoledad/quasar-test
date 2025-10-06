// stock_handlers.go
package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"quasar-ecommerce/api/db"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// StockHandler handles inventory/stock operations
type StockHandler struct {
	DB *pgxpool.Pool
}

func NewStockHandler(DB *pgxpool.Pool) *StockHandler {
	return &StockHandler{DB: DB}
}

func (h *StockHandler) GetStockByProduct(w http.ResponseWriter, r *http.Request) {
	productID := r.URL.Query().Get("product_id")
	if productID == "" {
		respondError(w, http.StatusBadRequest, "product_id is required")
		return
	}

	query := `
		SELECT si.id, si.storage_id, si.product_id, si.quantity, si.reserved_quantity,
			   s.name as storage_name, p.name as product_name, p.sku
		FROM stock_items si
		JOIN storages s ON si.storage_id = s.id
		JOIN products p ON si.product_id = p.id
		WHERE si.product_id = $1
	`

	rows, err := h.DB.Query(context.Background(), query, productID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch stock")
		return
	}
	defer rows.Close()

	type StockWithDetails struct {
		db.StockItem
		StorageName string `json:"storage_name"`
		ProductName string `json:"product_name"`
		SKU         string `json:"sku"`
	}

	var stocks []StockWithDetails
	for rows.Next() {
		var s StockWithDetails
		if err := rows.Scan(&s.ID, &s.StorageID, &s.ProductID, &s.Quantity, &s.ReservedQuantity,
			&s.StorageName, &s.ProductName, &s.SKU); err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to scan stock")
			return
		}
		stocks = append(stocks, s)
	}

	respondJSON(w, http.StatusOK, stocks)
}

func (h *StockHandler) UpdateStock(w http.ResponseWriter, r *http.Request) {
	var req db.UpdateStockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Start transaction
	tx, err := h.DB.Begin(context.Background())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to start transaction")
		return
	}
	defer tx.Rollback(context.Background())

	// Check if stock item exists
	var stockItem db.StockItem
	query := `SELECT id, storage_id, product_id, quantity, reserved_quantity 
			  FROM stock_items WHERE storage_id = $1 AND product_id = $2`
	err = tx.QueryRow(context.Background(), query, req.StorageID, req.ProductID).Scan(
		&stockItem.ID, &stockItem.StorageID, &stockItem.ProductID, &stockItem.Quantity, &stockItem.ReservedQuantity,
	)

	if err == pgx.ErrNoRows {
		// Create new stock item
		stockItem = db.StockItem{
			ID:               uuid.New(),
			StorageID:        req.StorageID,
			ProductID:        req.ProductID,
			Quantity:         req.Quantity,
			ReservedQuantity: 0,
		}

		insertQuery := `INSERT INTO stock_items (id, storage_id, product_id, quantity, reserved_quantity)
						VALUES ($1, $2, $3, $4, $5)`
		_, err = tx.Exec(context.Background(), insertQuery,
			stockItem.ID, stockItem.StorageID, stockItem.ProductID, stockItem.Quantity, stockItem.ReservedQuantity)
	} else if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch stock item")
		return
	} else {
		// Update existing stock item
		newQuantity := stockItem.Quantity + req.Quantity
		if newQuantity < 0 {
			respondError(w, http.StatusBadRequest, "Insufficient stock")
			return
		}

		updateQuery := `UPDATE stock_items SET quantity = $1 WHERE id = $2`
		_, err = tx.Exec(context.Background(), updateQuery, newQuantity, stockItem.ID)
		stockItem.Quantity = newQuantity
	}

	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update stock")
		return
	}

	// Record inventory movement
	movement := db.InventoryMovement{
		ID:           uuid.New(),
		StorageID:    &req.StorageID,
		ProductID:    &req.ProductID,
		MovementType: req.MovementType,
		Quantity:     req.Quantity,
		Note:         req.Note,
		CreatedAt:    time.Now(),
	}

	movementQuery := `INSERT INTO inventory_movements (id, storage_id, product_id, movement_type, quantity, note, created_at)
					  VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err = tx.Exec(context.Background(), movementQuery,
		movement.ID, movement.StorageID, movement.ProductID, movement.MovementType,
		movement.Quantity, movement.Note, movement.CreatedAt)

	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to record movement")
		return
	}

	// Commit transaction
	if err := tx.Commit(context.Background()); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to commit transaction")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"stock_item": stockItem,
		"movement":   movement,
	})
}

func (h *StockHandler) TransferStock(w http.ResponseWriter, r *http.Request) {
	var req db.TransferStockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Quantity <= 0 {
		respondError(w, http.StatusBadRequest, "Quantity must be positive")
		return
	}

	// Start transaction
	tx, err := h.DB.Begin(context.Background())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to start transaction")
		return
	}
	defer tx.Rollback(context.Background())

	// Decrease stock at source storage
	decreaseQuery := `UPDATE stock_items SET quantity = quantity - $1 
					  WHERE storage_id = $2 AND product_id = $3 AND quantity >= $1
					  RETURNING quantity`
	var newFromQuantity int
	err = tx.QueryRow(context.Background(), decreaseQuery, req.Quantity, req.FromStorageID, req.ProductID).Scan(&newFromQuantity)
	if err == pgx.ErrNoRows {
		respondError(w, http.StatusBadRequest, "Insufficient stock at source storage")
		return
	} else if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to decrease stock")
		return
	}

	// Increase stock at destination storage (or create if doesn't exist)
	upsertQuery := `
		INSERT INTO stock_items (id, storage_id, product_id, quantity, reserved_quantity)
		VALUES ($1, $2, $3, $4, 0)
		ON CONFLICT (storage_id, product_id) 
		DO UPDATE SET quantity = stock_items.quantity + $4
		RETURNING quantity
	`
	var newToQuantity int
	err = tx.QueryRow(context.Background(), upsertQuery, uuid.New(), req.ToStorageID, req.ProductID, req.Quantity).Scan(&newToQuantity)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to increase stock")
		return
	}

	// Record transfer out movement
	transferOutID := uuid.New()
	movementOutQuery := `INSERT INTO inventory_movements (id, storage_id, product_id, movement_type, quantity, reference_id, note, created_at)
						 VALUES ($1, $2, $3, 'transfer_out', $4, $5, $6, $7)`
	_, err = tx.Exec(context.Background(), movementOutQuery,
		uuid.New(), req.FromStorageID, req.ProductID, -req.Quantity, transferOutID, req.Note, time.Now())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to record transfer out")
		return
	}

	// Record transfer in movement
	movementInQuery := `INSERT INTO inventory_movements (id, storage_id, product_id, movement_type, quantity, reference_id, note, created_at)
						VALUES ($1, $2, $3, 'transfer_in', $4, $5, $6, $7)`
	_, err = tx.Exec(context.Background(), movementInQuery,
		uuid.New(), req.ToStorageID, req.ProductID, req.Quantity, transferOutID, req.Note, time.Now())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to record transfer in")
		return
	}

	// Commit transaction
	if err := tx.Commit(context.Background()); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to commit transaction")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message":              "Stock transferred successfully",
		"transfer_id":          transferOutID,
		"from_new_quantity":    newFromQuantity,
		"to_new_quantity":      newToQuantity,
		"transferred_quantity": req.Quantity,
	})
}

// OrderHandler handles order operations
type OrderHandler struct {
	DB *pgxpool.Pool
}

func NewOrderHandler(DB *pgxpool.Pool) *OrderHandler {
	return &OrderHandler{DB: DB}
}

func (h *OrderHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	var req db.CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Start transaction
	tx, err := h.DB.Begin(context.Background())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to start transaction")
		return
	}
	defer tx.Rollback(context.Background())

	// Calculate total and validate products
	var totalCents int
	for _, item := range req.Items {
		var product db.Product
		query := `SELECT id, price_cents, currency FROM products WHERE id = $1 AND is_active = true`
		err := tx.QueryRow(context.Background(), query, item.ProductID).Scan(&product.ID, &product.PriceCents, &product.Currency)
		if err == pgx.ErrNoRows {
			respondError(w, http.StatusBadRequest, "Product not found or inactive")
			return
		} else if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to fetch product")
			return
		}
		totalCents += product.PriceCents * item.Quantity
	}

	// Create order
	order := db.Order{
		ID:            uuid.New(),
		StoreID:       req.StoreID,
		CustomerName:  req.CustomerName,
		CustomerEmail: req.CustomerEmail,
		TotalCents:    totalCents,
		Currency:      "USD",
		Status:        "pending",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	orderQuery := `INSERT INTO orders (id, store_id, customer_name, customer_email, total_cents, currency, status, created_at, updated_at)
				   VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err = tx.Exec(context.Background(), orderQuery,
		order.ID, order.StoreID, order.CustomerName, order.CustomerEmail,
		order.TotalCents, order.Currency, order.Status, order.CreatedAt, order.UpdatedAt)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create order")
		return
	}

	// Create order items and reserve stock
	for _, item := range req.Items {
		var product db.Product
		productQuery := `SELECT id, price_cents FROM products WHERE id = $1`
		err := tx.QueryRow(context.Background(), productQuery, item.ProductID).Scan(&product.ID, &product.PriceCents)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to fetch product price")
			return
		}

		// Insert order item
		orderItem := db.OrderItem{
			ID:         uuid.New(),
			OrderID:    order.ID,
			ProductID:  &item.ProductID,
			Quantity:   item.Quantity,
			PriceCents: product.PriceCents,
		}

		itemQuery := `INSERT INTO order_items (id, order_id, product_id, quantity, price_cents)
					  VALUES ($1, $2, $3, $4, $5)`
		_, err = tx.Exec(context.Background(), itemQuery,
			orderItem.ID, orderItem.OrderID, orderItem.ProductID, orderItem.Quantity, orderItem.PriceCents)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to create order item")
			return
		}

		// Reserve stock
		reserveQuery := `UPDATE stock_items SET reserved_quantity = reserved_quantity + $1
						 WHERE storage_id = $2 AND product_id = $3 AND (quantity - reserved_quantity) >= $1`
		result, err := tx.Exec(context.Background(), reserveQuery, item.Quantity, req.StorageID, item.ProductID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to reserve stock")
			return
		}
		if result.RowsAffected() == 0 {
			respondError(w, http.StatusBadRequest, "Insufficient stock available")
			return
		}
	}

	// Create fulfillment order
	fulfillment := db.FulfillmentOrder{
		ID:        uuid.New(),
		OrderID:   order.ID,
		StorageID: &req.StorageID,
		Status:    "pending",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	fulfillmentQuery := `INSERT INTO fulfillment_orders (id, order_id, storage_id, status, created_at, updated_at)
						 VALUES ($1, $2, $3, $4, $5, $6)`
	_, err = tx.Exec(context.Background(), fulfillmentQuery,
		fulfillment.ID, fulfillment.OrderID, fulfillment.StorageID, fulfillment.Status,
		fulfillment.CreatedAt, fulfillment.UpdatedAt)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create fulfillment order")
		return
	}

	// Commit transaction
	if err := tx.Commit(context.Background()); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to commit transaction")
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"order":       order,
		"fulfillment": fulfillment,
	})
}

func (h *OrderHandler) ListOrders(w http.ResponseWriter, r *http.Request) {
	storeID := r.URL.Query().Get("store_id")

	var query string
	var rows pgx.Rows
	var err error

	if storeID != "" {
		query = `SELECT id, store_id, customer_name, customer_email, total_cents, currency, status, created_at, updated_at 
				FROM orders WHERE store_id = $1 ORDER BY created_at DESC`
		rows, err = h.DB.Query(context.Background(), query, storeID)
	} else {
		query = `SELECT id, store_id, customer_name, customer_email, total_cents, currency, status, created_at, updated_at 
				FROM orders ORDER BY created_at DESC`
		rows, err = h.DB.Query(context.Background(), query)
	}

	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch orders")
		return
	}
	defer rows.Close()

	var orders []db.Order
	for rows.Next() {
		var o db.Order
		if err := rows.Scan(&o.ID, &o.StoreID, &o.CustomerName, &o.CustomerEmail,
			&o.TotalCents, &o.Currency, &o.Status, &o.CreatedAt, &o.UpdatedAt); err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to scan order")
			return
		}
		orders = append(orders, o)
	}

	respondJSON(w, http.StatusOK, orders)
}

func (h *OrderHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid order ID")
		return
	}

	var order db.Order
	orderQuery := `SELECT id, store_id, customer_name, customer_email, total_cents, currency, status, created_at, updated_at 
				   FROM orders WHERE id = $1`
	err = h.DB.QueryRow(context.Background(), orderQuery, id).Scan(
		&order.ID, &order.StoreID, &order.CustomerName, &order.CustomerEmail,
		&order.TotalCents, &order.Currency, &order.Status, &order.CreatedAt, &order.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		respondError(w, http.StatusNotFound, "Order not found")
		return
	} else if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch order")
		return
	}

	// Fetch order items
	itemsQuery := `SELECT oi.id, oi.order_id, oi.product_id, oi.quantity, oi.price_cents, p.name, p.sku
				   FROM order_items oi
				   LEFT JOIN products p ON oi.product_id = p.id
				   WHERE oi.order_id = $1`
	rows, err := h.DB.Query(context.Background(), itemsQuery, id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch order items")
		return
	}
	defer rows.Close()

	type OrderItemWithProduct struct {
		db.OrderItem
		ProductName string `json:"product_name"`
		SKU         string `json:"sku"`
	}

	var items []OrderItemWithProduct
	for rows.Next() {
		var item OrderItemWithProduct
		if err := rows.Scan(&item.ID, &item.OrderID, &item.ProductID, &item.Quantity,
			&item.PriceCents, &item.ProductName, &item.SKU); err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to scan order item")
			return
		}
		items = append(items, item)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"order": order,
		"items": items,
	})
}

func (h *OrderHandler) UpdateOrderStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid order ID")
		return
	}

	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Start transaction
	tx, err := h.DB.Begin(context.Background())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to start transaction")
		return
	}
	defer tx.Rollback(context.Background())

	// Update order status
	query := `UPDATE orders SET status = $1, updated_at = $2 WHERE id = $3
			  RETURNING id, store_id, customer_name, customer_email, total_cents, currency, status, created_at, updated_at`
	var order db.Order
	err = tx.QueryRow(context.Background(), query, req.Status, time.Now(), id).Scan(
		&order.ID, &order.StoreID, &order.CustomerName, &order.CustomerEmail,
		&order.TotalCents, &order.Currency, &order.Status, &order.CreatedAt, &order.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		respondError(w, http.StatusNotFound, "Order not found")
		return
	} else if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update order")
		return
	}

	// If order is canceled, release reserved stock
	if req.Status == "canceled" {
		// Get order items
		itemsQuery := `SELECT product_id, quantity FROM order_items WHERE order_id = $1`
		rows, err := tx.Query(context.Background(), itemsQuery, id)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to fetch order items")
			return
		}
		defer rows.Close()

		// Get fulfillment storage
		var storageID uuid.UUID
		fulfillmentQuery := `SELECT storage_id FROM fulfillment_orders WHERE order_id = $1 LIMIT 1`
		err = tx.QueryRow(context.Background(), fulfillmentQuery, id).Scan(&storageID)
		if err != nil && err != pgx.ErrNoRows {
			respondError(w, http.StatusInternalServerError, "Failed to fetch fulfillment")
			return
		}

		// Release reserved stock
		for rows.Next() {
			var productID uuid.UUID
			var quantity int
			if err := rows.Scan(&productID, &quantity); err != nil {
				respondError(w, http.StatusInternalServerError, "Failed to scan item")
				return
			}

			releaseQuery := `UPDATE stock_items SET reserved_quantity = reserved_quantity - $1
							 WHERE storage_id = $2 AND product_id = $3`
			_, err := tx.Exec(context.Background(), releaseQuery, quantity, storageID, productID)
			if err != nil {
				respondError(w, http.StatusInternalServerError, "Failed to release stock")
				return
			}
		}
	}

	// If order is shipped, deduct from actual stock
	if req.Status == "shipped" {
		// Get order items
		itemsQuery := `SELECT product_id, quantity FROM order_items WHERE order_id = $1`
		rows, err := tx.Query(context.Background(), itemsQuery, id)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to fetch order items")
			return
		}
		defer rows.Close()

		// Get fulfillment storage
		var storageID uuid.UUID
		fulfillmentQuery := `SELECT storage_id FROM fulfillment_orders WHERE order_id = $1 LIMIT 1`
		err = tx.QueryRow(context.Background(), fulfillmentQuery, id).Scan(&storageID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to fetch fulfillment")
			return
		}

		// Deduct from stock and reserved quantity
		for rows.Next() {
			var productID uuid.UUID
			var quantity int
			if err := rows.Scan(&productID, &quantity); err != nil {
				respondError(w, http.StatusInternalServerError, "Failed to scan item")
				return
			}

			deductQuery := `UPDATE stock_items 
							SET quantity = quantity - $1, reserved_quantity = reserved_quantity - $1
							WHERE storage_id = $2 AND product_id = $3`
			_, err := tx.Exec(context.Background(), deductQuery, quantity, storageID, productID)
			if err != nil {
				respondError(w, http.StatusInternalServerError, "Failed to deduct stock")
				return
			}

			// Record inventory movement
			movementQuery := `INSERT INTO inventory_movements (id, storage_id, product_id, movement_type, quantity, reference_id, created_at)
							  VALUES ($1, $2, $3, 'sale', $4, $5, $6)`
			_, err = tx.Exec(context.Background(), movementQuery,
				uuid.New(), storageID, productID, -quantity, id, time.Now())
			if err != nil {
				respondError(w, http.StatusInternalServerError, "Failed to record movement")
				return
			}
		}

		// Update fulfillment order
		updateFulfillmentQuery := `UPDATE fulfillment_orders 
								   SET status = 'shipped', shipped_at = $1, updated_at = $2
								   WHERE order_id = $3`
		_, err = tx.Exec(context.Background(), updateFulfillmentQuery, time.Now(), time.Now(), id)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to update fulfillment")
			return
		}
	}

	// Commit transaction
	if err := tx.Commit(context.Background()); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to commit transaction")
		return
	}

	respondJSON(w, http.StatusOK, order)
}
