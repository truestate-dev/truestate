// Package model defines the core domain types for TrueState.
package model

import "time"

// Platform identifies a Linux distribution.
type Platform string

const (
	PlatformDebian  Platform = "debian"
	PlatformUbuntu  Platform = "ubuntu"
	PlatformProxmox Platform = "proxmox"
)

// InventoryType distinguishes host snapshots from golden baselines.
type InventoryType string

const (
	InventoryTypeHost   InventoryType = "host"
	InventoryTypeGolden InventoryType = "golden"
)

// AssertionStatus is what a security source says about a package/CVE pair.
type AssertionStatus string

const (
	AssertionAffected    AssertionStatus = "affected"
	AssertionFixed       AssertionStatus = "fixed"
	AssertionNotAffected AssertionStatus = "not_affected"
	AssertionUnderReview AssertionStatus = "under_review"
)

// DriftKind describes the type of deviation from a golden inventory.
type DriftKind string

const (
	DriftMissingPackage  DriftKind = "missing_package"
	DriftExtraPackage    DriftKind = "extra_package"
	DriftVersionMismatch DriftKind = "version_mismatch"
)

// SourceID identifies a security data source.
type SourceID string

const (
	SourceCVEOrg  SourceID = "cve.org"
	SourceDebian  SourceID = "debian_tracker"
	SourceUbuntu  SourceID = "ubuntu_tracker"
	SourceProxmox SourceID = "proxmox"
	SourceBSI     SourceID = "bsi"
	SourceNVD     SourceID = "nvd"
)

// Inventory is a package + platform snapshot for a host or golden image.
type Inventory struct {
	ID       string        `db:"id"       json:"id"`
	Name     string        `db:"name"     json:"name"`
	Type     InventoryType `db:"type"     json:"type"`
	Platform Platform      `db:"platform" json:"platform"`
	Release  string        `db:"release"  json:"release"` // e.g. "bookworm", "jammy"
	Packages []Package     `db:"-"        json:"packages,omitempty"`
	Metadata map[string]string `db:"-"   json:"metadata,omitempty"`
	CreatedAt time.Time   `db:"created_at" json:"created_at"`
	UpdatedAt time.Time   `db:"updated_at" json:"updated_at"`
}

// Package is a single installed package within an inventory.
type Package struct {
	ID          string `db:"id"           json:"id"`
	InventoryID string `db:"inventory_id" json:"inventory_id"`
	Name        string `db:"name"         json:"name"`
	Version     string `db:"version"      json:"version"`
	Arch        string `db:"arch"         json:"arch"`
}

// InventoryRelation links a host inventory to a golden baseline.
type InventoryRelation struct {
	HostInventoryID   string `db:"host_inventory_id"   json:"host_inventory_id"`
	GoldenInventoryID string `db:"golden_inventory_id" json:"golden_inventory_id"`
}

// Vulnerability represents a CVE record from the canonical source.
type Vulnerability struct {
	CVEID        string    `db:"cve_id"        json:"cve_id"`
	Summary      string    `db:"summary"       json:"summary"`
	PublishedAt  time.Time `db:"published_at"  json:"published_at"`
	UpdatedAt    time.Time `db:"updated_at"    json:"updated_at"`
	References   []string  `db:"-"             json:"references,omitempty"`
	CVSSScore    float64   `db:"cvss_score"    json:"cvss_score,omitempty"`
	CVSSSeverity string    `db:"cvss_severity" json:"cvss_severity,omitempty"`
	CVSSVector   string    `db:"cvss_vector"   json:"cvss_vector,omitempty"`
}

// Assertion is what a specific source says about a package+CVE pair
// in the context of a platform and release.
type Assertion struct {
	ID           string          `db:"id"            json:"id"`
	Source       SourceID        `db:"source"        json:"source"`
	CVEID        string          `db:"cve_id"        json:"cve_id"`
	PackageName  string          `db:"package_name"  json:"package_name"`
	Platform     Platform        `db:"platform"      json:"platform"`
	Release      string          `db:"release"       json:"release"`
	Status       AssertionStatus `db:"status"        json:"status"`
	FixedVersion string          `db:"fixed_version" json:"fixed_version,omitempty"`
	AdvisoryRefs []string        `db:"-"             json:"advisory_refs,omitempty"`
	FetchedAt    time.Time       `db:"fetched_at"    json:"fetched_at"`
}

// Finding is a single vulnerability result for a package in an inventory.
type Finding struct {
	InventoryID  string          `json:"inventory_id"`
	PackageName  string          `json:"package_name"`
	Version      string          `json:"version"`
	CVEID        string          `json:"cve_id"`
	Status       AssertionStatus `json:"status"`
	FixedVersion string          `json:"fixed_version,omitempty"`
	Source       SourceID        `json:"source"`
	FetchedAt    time.Time       `json:"fetched_at"`
	AdvisoryRefs []string        `json:"advisory_refs,omitempty"`
	DriftRelated bool            `json:"drift_related"`
	CVSSScore    float64         `json:"cvss_score,omitempty"`
	CVSSSeverity string          `json:"cvss_severity,omitempty"`
}

// DriftItem describes one deviation between a host and its golden inventory.
type DriftItem struct {
	Kind            DriftKind `json:"kind"`
	PackageName     string    `json:"package_name"`
	HostVersion     string    `json:"host_version,omitempty"`
	BaselineVersion string    `json:"baseline_version,omitempty"`
}

// Evaluation is the result of assessing an inventory.
type Evaluation struct {
	ID                string      `json:"id"`
	InventoryID       string      `json:"inventory_id"`
	GoldenInventoryID string      `json:"golden_inventory_id,omitempty"`
	Findings          []Finding   `json:"findings"`
	DriftItems        []DriftItem `json:"drift_items,omitempty"`
	EvaluatedAt       time.Time   `json:"evaluated_at"`
}

// SourceStatus records the health and freshness of a data source.
type SourceStatus struct {
	Source      SourceID   `db:"source"       json:"source"`
	LastSyncAt  *time.Time `db:"last_sync_at" json:"last_sync_at"`
	RecordCount int        `db:"record_count" json:"record_count"`
	Error       string    `db:"error"        json:"error,omitempty"`
}
