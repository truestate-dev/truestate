package db

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"gitea.local.vjinx.de/truestate/truestate/internal/model"
)

// InventoryWithStats extends Inventory with denormalized counters for the list view.
type InventoryWithStats struct {
	model.Inventory
	PackageCount      int        `json:"package_count"`
	LastEvalID        string     `json:"last_eval_id,omitempty"`
	LastEvaluatedAt   *time.Time `json:"last_evaluated_at,omitempty"`
	LastFindingCount  int        `json:"last_finding_count"`
	LastFixableCount  int        `json:"last_fixable_count"`
	LastDriftCount    int        `json:"last_drift_count"`
}

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

// ListInventoriesWithStats returns all inventories with package count and
// the most recent evaluation summary (finding count, drift count, evaluated_at).
func ListInventoriesWithStats(ctx context.Context, pool *pgxpool.Pool) ([]InventoryWithStats, error) {
	rows, err := pool.Query(ctx, `
		SELECT
			i.id, i.name, i.type, i.platform, i.release, i.metadata,
			i.created_at, i.updated_at,
			COUNT(DISTINCT p.id)                       AS package_count,
			e.id                                       AS last_eval_id,
			e.evaluated_at                             AS last_evaluated_at,
			COALESCE(e.finding_count, 0)               AS last_finding_count,
			COALESCE(e.fixable_count, 0)               AS last_fixable_count,
			COALESCE(e.drift_count, 0)                 AS last_drift_count
		FROM inventories i
		LEFT JOIN packages p ON p.inventory_id = i.id
		LEFT JOIN LATERAL (
			SELECT ev.id, ev.evaluated_at,
			       COUNT(DISTINCT f.id)                                        AS finding_count,
			       COUNT(DISTINCT CASE WHEN f.status = 'fixed' THEN f.id END) AS fixable_count,
			       COUNT(DISTINCT d.id)                                        AS drift_count
			FROM evaluations ev
			LEFT JOIN findings    f ON f.evaluation_id = ev.id
			LEFT JOIN drift_items d ON d.evaluation_id = ev.id
			WHERE ev.inventory_id = i.id
			GROUP BY ev.id
			ORDER BY ev.evaluated_at DESC
			LIMIT 1
		) e ON true
		GROUP BY i.id, e.id, e.evaluated_at, e.finding_count, e.fixable_count, e.drift_count
		ORDER BY i.created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list inventories with stats: %w", err)
	}
	defer rows.Close()

	var out []InventoryWithStats
	for rows.Next() {
		var s InventoryWithStats
		var lastEvalID *string
		if err := rows.Scan(
			&s.ID, &s.Name, &s.Type, &s.Platform, &s.Release, &s.Metadata,
			&s.CreatedAt, &s.UpdatedAt,
			&s.PackageCount,
			&lastEvalID, &s.LastEvaluatedAt,
			&s.LastFindingCount, &s.LastFixableCount, &s.LastDriftCount,
		); err != nil {
			return nil, fmt.Errorf("list inventories with stats: scan: %w", err)
		}
		if lastEvalID != nil {
			s.LastEvalID = *lastEvalID
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// DeleteInventory removes an inventory and all its dependent data (packages,
// evaluations, findings, drift_items) via CASCADE.
func DeleteInventory(ctx context.Context, pool *pgxpool.Pool, id string) error {
	tag, err := pool.Exec(ctx, `DELETE FROM inventories WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete inventory: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("inventory not found: %s", id)
	}
	return nil
}
