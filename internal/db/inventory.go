package db

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"gitea.local.vjinx.de/truestate/truestate/internal/model"
)

// CreateInventory inserts a new inventory and its packages.
func CreateInventory(ctx context.Context, pool *pgxpool.Pool, inv *model.Inventory) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	inv.ID = uuid.New().String()
	if inv.Metadata == nil {
		inv.Metadata = map[string]string{}
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO inventories (id, name, type, platform, release, metadata)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		inv.ID, inv.Name, inv.Type, inv.Platform, inv.Release, inv.Metadata,
	)
	if err != nil {
		return fmt.Errorf("insert inventory: %w", err)
	}

	for i := range inv.Packages {
		inv.Packages[i].ID = uuid.New().String()
		inv.Packages[i].InventoryID = inv.ID
		_, err = tx.Exec(ctx, `
			INSERT INTO packages (id, inventory_id, name, version, arch)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (inventory_id, name, arch) DO UPDATE SET version = EXCLUDED.version`,
			inv.Packages[i].ID,
			inv.Packages[i].InventoryID,
			inv.Packages[i].Name,
			inv.Packages[i].Version,
			inv.Packages[i].Arch,
		)
		if err != nil {
			return fmt.Errorf("insert package %s: %w", inv.Packages[i].Name, err)
		}
	}

	return tx.Commit(ctx)
}

// GetInventory fetches an inventory and its packages by ID.
func GetInventory(ctx context.Context, pool *pgxpool.Pool, id string) (*model.Inventory, error) {
	inv := &model.Inventory{}
	err := pool.QueryRow(ctx, `
		SELECT id, name, type, platform, release, metadata, created_at, updated_at
		FROM inventories WHERE id = $1`, id,
	).Scan(&inv.ID, &inv.Name, &inv.Type, &inv.Platform, &inv.Release,
		&inv.Metadata, &inv.CreatedAt, &inv.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get inventory: %w", err)
	}

	rows, err := pool.Query(ctx,
		`SELECT id, inventory_id, name, version, arch FROM packages WHERE inventory_id = $1`, id)
	if err != nil {
		return nil, fmt.Errorf("get packages: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var p model.Package
		if err := rows.Scan(&p.ID, &p.InventoryID, &p.Name, &p.Version, &p.Arch); err != nil {
			return nil, fmt.Errorf("scan package: %w", err)
		}
		inv.Packages = append(inv.Packages, p)
	}
	return inv, rows.Err()
}

// ListInventories returns all inventories (without packages).
func ListInventories(ctx context.Context, pool *pgxpool.Pool) ([]model.Inventory, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, name, type, platform, release, metadata, created_at, updated_at
		FROM inventories ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []model.Inventory
	for rows.Next() {
		var inv model.Inventory
		if err := rows.Scan(&inv.ID, &inv.Name, &inv.Type, &inv.Platform, &inv.Release,
			&inv.Metadata, &inv.CreatedAt, &inv.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, inv)
	}
	return out, rows.Err()
}

// SetInventoryRelation links a host inventory to a golden baseline.
func SetInventoryRelation(ctx context.Context, pool *pgxpool.Pool, hostID, goldenID string) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO inventory_relations (host_inventory_id, golden_inventory_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING`, hostID, goldenID)
	return err
}

// GetGoldenForHost returns the golden inventory linked to a host, if any.
func GetGoldenForHost(ctx context.Context, pool *pgxpool.Pool, hostID string) (*model.Inventory, error) {
	var goldenID string
	err := pool.QueryRow(ctx, `
		SELECT golden_inventory_id FROM inventory_relations
		WHERE host_inventory_id = $1 LIMIT 1`, hostID,
	).Scan(&goldenID)
	if err != nil {
		return nil, nil // no relation is not an error
	}
	return GetInventory(ctx, pool, goldenID)
}
