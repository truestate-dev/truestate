// Typed API client — matches the Go model types.

export type Platform = 'debian' | 'ubuntu' | 'proxmox'
export type InventoryType = 'host' | 'golden'
export type AssertionStatus = 'affected' | 'fixed' | 'not_affected' | 'under_review'
export type DriftKind = 'missing_package' | 'extra_package' | 'version_mismatch'
export type SourceID = 'cve.org' | 'debian_tracker' | 'ubuntu_tracker' | 'proxmox' | 'bsi' | 'nvd'
export type CVSSSeverity = 'CRITICAL' | 'HIGH' | 'MEDIUM' | 'LOW' | 'NONE' | ''

export interface Inventory {
  id: string
  name: string
  type: InventoryType
  platform: Platform
  release: string
  packages?: Package[]
  metadata?: Record<string, string>
  created_at: string
  updated_at: string
}

export interface Package {
  id: string
  inventory_id: string
  name: string
  version: string
  arch: string
}

export interface Finding {
  inventory_id: string
  package_name: string
  version: string
  cve_id: string
  status: AssertionStatus
  fixed_version?: string
  source: SourceID
  fetched_at: string
  drift_related: boolean
  cvss_score?: number
  cvss_severity?: CVSSSeverity
}

export interface DriftItem {
  kind: DriftKind
  package_name: string
  host_version?: string
  baseline_version?: string
}

export interface Evaluation {
  id: string
  inventory_id: string
  golden_inventory_id?: string
  findings: Finding[]
  drift_items?: DriftItem[]
  evaluated_at: string
}

export interface EvaluationSummary {
  id: string
  inventory_id: string
  golden_inventory_id?: string
  finding_count: number
  drift_count: number
  evaluated_at: string
}

export interface InventoryWithStats extends Inventory {
  package_count: number
  last_eval_id?: string
  last_evaluated_at?: string
  last_finding_count: number
  last_drift_count: number
}

export interface SourceStatus {
  source: SourceID
  last_sync_at?: string
  record_count: number
  error?: string
}

const BASE = '/api/v1'

async function req<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  })
  if (!res.ok) {
    const body = await res.json().catch(() => ({}))
    throw new Error(body.error ?? `HTTP ${res.status}`)
  }
  return res.json()
}

export const api = {
  listInventories: () => req<InventoryWithStats[]>('/inventories'),
  getInventory: (id: string) => req<Inventory>(`/inventories/${id}`),
  deleteInventory: (id: string) =>
    fetch(`${BASE}/inventories/${id}`, { method: 'DELETE' }).then((r) => {
      if (!r.ok && r.status !== 204) throw new Error(`HTTP ${r.status}`)
    }),
  listEvaluations: (inventoryId: string) =>
    req<EvaluationSummary[]>(`/inventories/${inventoryId}/evaluations`),
  runEvaluation: (inventoryId: string) =>
    req<Evaluation>(`/evaluate/${inventoryId}`, { method: 'POST' }),
  getEvaluation: (id: string) => req<Evaluation>(`/evaluations/${id}`),
  listSources: () => req<SourceStatus[]>('/sources'),
}
