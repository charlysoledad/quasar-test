package main

import (
	"context"
	"log"

	"quasar-ecommerce/api/db"
	"quasar-ecommerce/api/db/repositories"
)

func main() {
	// Initialize DB connection (reads from .env)
	if err := db.Connect(); err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()

	tenantRepo := repositories.TenantRepo{DB: db.DB}
	storeRepo := repositories.StoreRepo{DB: db.DB}
	storageRepo := repositories.StorageRepo{DB: db.DB}

	tenantID, err := tenantRepo.Create(ctx, "ACME Corp", "acme")
	if err != nil {
		log.Fatal("create tenant:", err)
	}

	storeID, err := storeRepo.Create(ctx, tenantID, "ACME Online Store", "acme-online")
	if err != nil {
		log.Fatal("create store:", err)
	}

	_, _ = storageRepo.Create(ctx, storeID, "Main Warehouse", "Mexico City", true)
	_, _ = storageRepo.Create(ctx, storeID, "Sucursal Norte", "Monterrey", false)

	log.Println("✅ Setup complete.")
}
