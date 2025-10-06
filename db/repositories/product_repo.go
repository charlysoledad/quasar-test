package repositories

import (
	"context"
	"database/sql"
)

type ProductStock struct {
	ProductID string
	StorageID string
	Product   string
	Quantity  int
}

type ProductRepo struct {
	DB *sql.DB
}

// Get available stock across storages
func (r *ProductRepo) GetStockByProduct(ctx context.Context, productID string) ([]ProductStock, error) {
	query := `
		SELECT si.product_id, si.storage_id, p.name, (si.quantity - si.reserved_quantity) AS available
		FROM stock_items si
		JOIN products p ON si.product_id = p.id
		WHERE si.product_id = $1
	`
	rows, err := r.DB.QueryContext(ctx, query, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stocks []ProductStock
	for rows.Next() {
		var ps ProductStock
		if err := rows.Scan(&ps.ProductID, &ps.StorageID, &ps.Product, &ps.Quantity); err != nil {
			return nil, err
		}
		stocks = append(stocks, ps)
	}
	return stocks, nil
}
