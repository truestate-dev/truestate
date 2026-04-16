package db

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"gitea.local.vjinx.de/truestate/truestate/internal/model"
)

// SaveEvaluation persists an Evaluation and all its Findings and DriftItems.
// It sets eval.ID to a fresh UUID before inserting.
func SaveEvaluation(ctx context.Context, pool *pgxpool.Pool, eval *model.Evaluation) error {
	eval.ID = uuid.New().String()

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("save evaluation: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO evaluations (id, inventory_id, golden_inventory_id, evaluated_at)
		VALUES ($1, $2, $3, $4)`,
		eval.ID,
		eval.InventoryID,
		nullableString(eval.GoldenInventoryID),
		eval.EvaluatedAt,
	)
	if err != nil {
		return fmt.Errorf("save evaluation: insert evaluation: %w", err)
	}

	// Bulk-insert findings via CopyFrom.
	if len(eval.Findings) > 0 {
		findingRows := make([][]any, len(eval.Findings))
		for i, f := range eval.Findings {
			findingRows[i] = []any{
				uuid.New().String(),
				eval.ID,
				f.InventoryID,
				f.PackageName,
				f.Version,
				f.CVEID,
				string(f.Status),
				f.FixedVersion,
				string(f.Source),
				f.FetchedAt,
				f.DriftRelated,
			}
		}
		_, err = tx.CopyFrom(ctx,
			pgx.Identifier{"findings"},
			[]string{"id", "evaluation_id", "inventory_id", "package_name", "version",
				"cve_id", "status", "fixed_version", "source", "fetched_at", "drift_related"},
			pgx.CopyFromRows(findingRows),
		)
		if err != nil {
			return fmt.Errorf("save evaluation: copy findings: %w", err)
		}
	}

	// Bulk-insert drift items.
	if len(eval.DriftItems) > 0 {
		driftRows := make([][]any, len(eval.DriftItems))
		for i, d := range eval.DriftItems {
			driftRows[i] = []any{
				uuid.New().String(),
				eval.ID,
				string(d.Kind),
				d.PackageName,
				d.HostVersion,
				d.BaselineVersion,
			}
		}
		_, err = tx.CopyFrom(ctx,
			pgx.Identifier{"drift_items"},
			[]string{"id", "evaluation_id", "kind", "package_name", "host_version", "baseline_version"},
			pgx.CopyFromRows(driftRows),
		)
		if err != nil {
			return fmt.Errorf("save evaluation: copy drift items: %w", err)
		}
	}

	return tx.Commit(ctx)
}

// GetEvaluation fetches a stored evaluation by ID, including findings and drift items.
func GetEvaluation(ctx context.Context, pool *pgxpool.Pool, id string) (*model.Evaluation, error) {
	eval := &model.Evaluation{}
	var goldenID *string
	err := pool.QueryRow(ctx, `
		SELECT id, inventory_id, golden_inventory_id, evaluated_at
		FROM evaluations WHERE id = $1`, id,
	).Scan(&eval.ID, &eval.InventoryID, &goldenID, &eval.EvaluatedAt)
	if err != nil {
		return nil, fmt.Errorf("get evaluation: %w", err)
	}
	if goldenID != nil {
		eval.GoldenInventoryID = *goldenID
	}

	rows, err := pool.Query(ctx, `
		SELECT inventory_id, package_name, version, cve_id, status,
		       fixed_version, source, fetched_at, drift_related
		FROM findings WHERE evaluation_id = $1
		ORDER BY package_name, cve_id`, id)
	if err != nil {
		return nil, fmt.Errorf("get evaluation: query findings: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var f model.Finding
		if err := rows.Scan(&f.InventoryID, &f.PackageName, &f.Version, &f.CVEID,
			&f.Status, &f.FixedVersion, &f.Source, &f.FetchedAt, &f.DriftRelated); err != nil {
			return nil, fmt.Errorf("get evaluation: scan finding: %w", err)
		}
		eval.Findings = append(eval.Findings, f)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	driftRows, err := pool.Query(ctx, `
		SELECT kind, package_name, host_version, baseline_version
		FROM drift_items WHERE evaluation_id = $1
		ORDER BY package_name`, id)
	if err != nil {
		return nil, fmt.Errorf("get evaluation: query drift items: %w", err)
	}
	defer driftRows.Close()
	for driftRows.Next() {
		var d model.DriftItem
		if err := driftRows.Scan(&d.Kind, &d.PackageName, &d.HostVersion, &d.BaselineVersion); err != nil {
			return nil, fmt.Errorf("get evaluation: scan drift item: %w", err)
		}
		eval.DriftItems = append(eval.DriftItems, d)
	}
	return eval, driftRows.Err()
}

// EvaluationSummary is a lightweight view of a stored evaluation (no findings).
type EvaluationSummary struct {
	ID                string    `json:"id"`
	InventoryID       string    `json:"inventory_id"`
	GoldenInventoryID string    `json:"golden_inventory_id,omitempty"`
	FindingCount      int       `json:"finding_count"`
	DriftCount        int       `json:"drift_count"`
	EvaluatedAt       time.Time `json:"evaluated_at"`
}

// ListEvaluationsForInventory returns summaries of all evaluations for an inventory,
// newest first.
func ListEvaluationsForInventory(ctx context.Context, pool *pgxpool.Pool, inventoryID string) ([]EvaluationSummary, error) {
	rows, err := pool.Query(ctx, `
		SELECT
			e.id,
			e.inventory_id,
			COALESCE(e.golden_inventory_id::text, ''),
			COUNT(DISTINCT f.id)  AS finding_count,
			COUNT(DISTINCT d.id)  AS drift_count,
			e.evaluated_at
		FROM evaluations e
		LEFT JOIN findings   f ON f.evaluation_id = e.id
		LEFT JOIN drift_items d ON d.evaluation_id = e.id
		WHERE e.inventory_id = $1
		GROUP BY e.id
		ORDER BY e.evaluated_at DESC`, inventoryID)
	if err != nil {
		return nil, fmt.Errorf("list evaluations: %w", err)
	}
	defer rows.Close()

	var out []EvaluationSummary
	for rows.Next() {
		var s EvaluationSummary
		if err := rows.Scan(&s.ID, &s.InventoryID, &s.GoldenInventoryID,
			&s.FindingCount, &s.DriftCount, &s.EvaluatedAt); err != nil {
			return nil, fmt.Errorf("list evaluations: scan: %w", err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// nullableString returns nil for an empty string (maps to SQL NULL).
func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
