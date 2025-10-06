// main.go
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"quasar-ecommerce/api/handlers"
	"time"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Database connection
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer pool.Close()

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("Unable to ping database: %v\n", err)
	}
	log.Println("Successfully connected to database")

	// Initialize router
	r := mux.NewRouter()

	// Middleware
	r.Use(loggingMiddleware)
	r.Use(corsMiddleware)

	// API routes
	api := r.PathPrefix("/api/v1").Subrouter()

	// Health check
	r.HandleFunc("/health", healthCheckHandler(pool)).Methods("GET")

	// Initialize handlers
	tenantHandler := &handlers.TenantHandler{DB: pool}
	storeHandler := &handlers.StoreHandler{DB: pool}
	productHandler := &handlers.ProductHandler{DB: pool}
	storageHandler := &handlers.StorageHandler{DB: pool}
	stockHandler := &handlers.StockHandler{DB: pool}
	orderHandler := &handlers.OrderHandler{DB: pool}

	// Tenant routes
	api.HandleFunc("/tenants", tenantHandler.CreateTenant).Methods("POST")
	api.HandleFunc("/tenants", tenantHandler.ListTenants).Methods("GET")
	api.HandleFunc("/tenants/{id}", tenantHandler.GetTenant).Methods("GET")
	api.HandleFunc("/tenants/{id}", tenantHandler.UpdateTenant).Methods("PUT")
	api.HandleFunc("/tenants/{id}", tenantHandler.DeleteTenant).Methods("DELETE")

	// Store routes
	api.HandleFunc("/stores", storeHandler.CreateStore).Methods("POST")
	api.HandleFunc("/stores", storeHandler.ListStores).Methods("GET")
	api.HandleFunc("/stores/{id}", storeHandler.GetStore).Methods("GET")
	api.HandleFunc("/stores/{id}", storeHandler.UpdateStore).Methods("PUT")

	// Product routes
	api.HandleFunc("/products", productHandler.CreateProduct).Methods("POST")
	api.HandleFunc("/products", productHandler.ListProducts).Methods("GET")
	api.HandleFunc("/products/{id}", productHandler.GetProduct).Methods("GET")
	api.HandleFunc("/products/{id}", productHandler.UpdateProduct).Methods("PUT")
	api.HandleFunc("/products/{id}", productHandler.DeleteProduct).Methods("DELETE")

	// Storage (warehouse) routes
	api.HandleFunc("/storages", storageHandler.CreateStorage).Methods("POST")
	api.HandleFunc("/storages", storageHandler.ListStorages).Methods("GET")
	api.HandleFunc("/storages/{id}", storageHandler.GetStorage).Methods("GET")
	api.HandleFunc("/storages/{id}", storageHandler.UpdateStorage).Methods("PUT")

	// Stock routes
	api.HandleFunc("/stock", stockHandler.GetStockByProduct).Methods("GET")
	api.HandleFunc("/stock", stockHandler.UpdateStock).Methods("POST")
	api.HandleFunc("/stock/transfer", stockHandler.TransferStock).Methods("POST")

	// Order routes
	api.HandleFunc("/orders", orderHandler.CreateOrder).Methods("POST")
	api.HandleFunc("/orders", orderHandler.ListOrders).Methods("GET")
	api.HandleFunc("/orders/{id}", orderHandler.GetOrder).Methods("GET")
	api.HandleFunc("/orders/{id}/status", orderHandler.UpdateOrderStatus).Methods("PUT")

	// Server configuration
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		log.Printf("Server starting on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c

	// Shutdown gracefully
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	log.Println("Shutting down server...")
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	log.Println("Server stopped")
}

func healthCheckHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := pool.Ping(r.Context()); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"error","message":"database unavailable"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","message":"service healthy"}`))
	}
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("%s %s %s", r.Method, r.RequestURI, time.Since(start))
		next.ServeHTTP(w, r)
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
