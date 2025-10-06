package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv" // for loading .env files
)

var DB *sql.DB

// Connect initializes the global DB connection
func Connect() error {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Println("⚠️  Warning: No .env file found (using system env vars instead)")
	}

	// Read env vars
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	pass := os.Getenv("DB_PASSWORD")
	name := os.Getenv("DB_NAME")

	if host == "" || user == "" || name == "" {
		return fmt.Errorf("missing required DB environment variables")
	}

	// Build connection string
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=verify-full",
		user, pass, host, port, name,
	)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("failed to open DB connection: %w", err)
	}

	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(time.Hour)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping DB: %w", err)
	}

	DB = db
	log.Printf("✅ Connected to PostgreSQL at %s:%s (%s)\n", host, port, name)
	return nil
}

func Close() {
	if DB != nil {
		DB.Close()
	}
}
