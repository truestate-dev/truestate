// Package engine implements vulnerability matching and drift evaluation.
package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"gitea.local.vjinx.de/truestate/truestate/internal/db"
	"gitea.local.vjinx.de/truestate/truestate/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Evaluate runs a full evaluation for an inventory.
// It fetches matching assertions, optionally compares against a golden baseline,
// and returns an Evaluation with findings and drift items.
func Evaluate(ctx context.Context, pool *pgxpool.Pool, inventoryID string) (*model.Evaluation, error) {
	inv, err := db.GetInventory(ctx, pool, inventoryID)
	if err != nil {
		return nil, fmt.Errorf("evaluate: get inventory: %w", err)
	}

	golden, err := db.GetGoldenForHost(ctx, pool, inventoryID)
	if err != nil {
		return nil, fmt.Errorf("evaluate: get golden: %w", err)
	}

	eval := &model.Evaluation{
		ID:          uuid.New().String(),
		InventoryID: inventoryID,
		EvaluatedAt: time.Now().UTC(),
	}
	if golden != nil {
		eval.GoldenInventoryID = golden.ID
	}

	// Build a drift set if we have a golden baseline.
	driftPackages := map[string]bool{}
	if golden != nil {
		eval.DriftItems = computeDrift(inv.Packages, golden.Packages)
		for _, d := range eval.DriftItems {
			driftPackages[d.PackageName] = true
		}
	}

	// Match each package in the inventory against stored assertions.
	for _, pkg := range inv.Packages {
		assertions, err := db.GetAssertionsForPackage(ctx, pool,
			pkg.Name, string(inv.Platform), inv.Release)
		if err != nil {
			return nil, fmt.Errorf("evaluate: get assertions for %s: %w", pkg.Name, err)
		}

		for _, a := range assertions {
			if a.Status == model.AssertionNotAffected {
				continue
			}
			// If fixed, only report if the installed version is still vulnerable.
			if a.Status == model.AssertionFixed && a.FixedVersion != "" {
				if versionGTE(pkg.Version, a.FixedVersion) {
					continue // package is at or above the fix — not vulnerable
				}
			}

			eval.Findings = append(eval.Findings, model.Finding{
				InventoryID:  inventoryID,
				PackageName:  pkg.Name,
				Version:      pkg.Version,
				CVEID:        a.CVEID,
				Status:       a.Status,
				FixedVersion: a.FixedVersion,
				Source:       a.Source,
				FetchedAt:    a.FetchedAt,
				DriftRelated: driftPackages[pkg.Name],
			})
		}
	}

	return eval, nil
}

// computeDrift returns the list of package deviations between host and golden.
func computeDrift(host, golden []model.Package) []model.DriftItem {
	goldenMap := make(map[string]string, len(golden))
	for _, p := range golden {
		goldenMap[p.Name] = p.Version
	}

	hostMap := make(map[string]string, len(host))
	for _, p := range host {
		hostMap[p.Name] = p.Version
	}

	var items []model.DriftItem

	// Packages in host but not in golden, or different version.
	for _, p := range host {
		goldenVersion, inGolden := goldenMap[p.Name]
		if !inGolden {
			items = append(items, model.DriftItem{
				Kind:        model.DriftExtraPackage,
				PackageName: p.Name,
				HostVersion: p.Version,
			})
		} else if p.Version != goldenVersion {
			items = append(items, model.DriftItem{
				Kind:            model.DriftVersionMismatch,
				PackageName:     p.Name,
				HostVersion:     p.Version,
				BaselineVersion: goldenVersion,
			})
		}
	}

	// Packages in golden but missing from host.
	for _, p := range golden {
		if _, inHost := hostMap[p.Name]; !inHost {
			items = append(items, model.DriftItem{
				Kind:            model.DriftMissingPackage,
				PackageName:     p.Name,
				BaselineVersion: p.Version,
			})
		}
	}

	return items
}

// versionGTE returns true if installed >= fixed using a simple lexicographic
// comparison. For proper Debian epoch:upstream-revision comparison, this
// should be replaced with a dpkg-version-aware comparator.
// TODO: replace with proper Debian version comparison (dpkg algorithm).
func versionGTE(installed, fixed string) bool {
	return installed >= fixed
}
