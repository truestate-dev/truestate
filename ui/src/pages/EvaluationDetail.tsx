import { useQuery } from '@tanstack/react-query'
import { useParams, Link } from 'react-router-dom'
import { api, Finding, DriftItem, AssertionStatus, CVSSSeverity } from '../api/client'

const statusStyle: Record<AssertionStatus, string> = {
  fixed: 'bg-amber-100 text-amber-800',
  affected: 'bg-red-100 text-red-800',
  not_affected: 'bg-green-100 text-green-700',
  under_review: 'bg-slate-100 text-slate-600',
}

const driftStyle: Record<string, string> = {
  missing_package: 'bg-red-100 text-red-800',
  extra_package: 'bg-blue-100 text-blue-800',
  version_mismatch: 'bg-amber-100 text-amber-800',
}

const severityStyle: Record<string, string> = {
  CRITICAL: 'bg-red-700 text-white',
  HIGH: 'bg-orange-500 text-white',
  MEDIUM: 'bg-yellow-400 text-gray-900',
  LOW: 'bg-blue-400 text-white',
}

function severityOrder(s: CVSSSeverity | undefined): number {
  switch (s) {
    case 'CRITICAL': return 0
    case 'HIGH': return 1
    case 'MEDIUM': return 2
    case 'LOW': return 3
    default: return 4
  }
}

function Badge({ label, cls }: { label: string; cls: string }) {
  return (
    <span className={`inline-flex px-2 py-0.5 rounded text-xs font-medium ${cls}`}>
      {label}
    </span>
  )
}

function SeverityBadge({ severity, score }: { severity?: CVSSSeverity; score?: number }) {
  if (!severity || !score) return <span className="text-slate-300 text-xs">—</span>
  const cls = severityStyle[severity] ?? 'bg-slate-100 text-slate-600'
  return (
    <span className={`inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium ${cls}`}>
      {severity}
      <span className="opacity-75 font-normal">{score.toFixed(1)}</span>
    </span>
  )
}

function FindingRow({ f }: { f: Finding }) {
  return (
    <tr className={`hover:bg-slate-50 transition-colors ${f.drift_related ? 'bg-amber-50/40' : ''}`}>
      <td className="px-3 py-2">
        <SeverityBadge severity={f.cvss_severity} score={f.cvss_score} />
      </td>
      <td className="px-3 py-2 font-mono text-xs font-medium">{f.package_name}</td>
      <td className="px-3 py-2 font-mono text-xs text-slate-500">{f.version}</td>
      <td className="px-3 py-2 text-xs">
        <a
          href={`https://www.cve.org/CVERecord?id=${f.cve_id}`}
          target="_blank" rel="noopener noreferrer"
          className="text-blue-600 hover:underline"
        >
          {f.cve_id}
        </a>
      </td>
      <td className="px-3 py-2">
        <Badge label={f.status} cls={statusStyle[f.status]} />
      </td>
      <td className="px-3 py-2 font-mono text-xs text-slate-500">
        {f.fixed_version || '—'}
      </td>
      <td className="px-3 py-2 text-xs text-slate-400">{f.source}</td>
      <td className="px-3 py-2 text-center">
        {f.drift_related && <span title="Drift-related" className="text-amber-500 text-xs">⚠</span>}
      </td>
    </tr>
  )
}

function DriftRow({ d }: { d: DriftItem }) {
  return (
    <tr className="hover:bg-slate-50">
      <td className="px-4 py-2 font-mono text-xs font-medium">{d.package_name}</td>
      <td className="px-4 py-2">
        <Badge label={d.kind.replace('_', ' ')} cls={driftStyle[d.kind] ?? ''} />
      </td>
      <td className="px-4 py-2 font-mono text-xs text-slate-500">{d.host_version || '—'}</td>
      <td className="px-4 py-2 font-mono text-xs text-slate-500">{d.baseline_version || '—'}</td>
    </tr>
  )
}

export default function EvaluationDetail() {
  const { id } = useParams<{ id: string }>()

  const { data: ev, isLoading, error } = useQuery({
    queryKey: ['evaluation', id],
    queryFn: () => api.getEvaluation(id!),
  })

  if (isLoading) return <p className="text-slate-500">Loading…</p>
  if (error) return <p className="text-red-600">{(error as Error).message}</p>
  if (!ev) return null

  const findings = [...(ev.findings ?? [])].sort((a, b) => {
    const so = severityOrder(a.cvss_severity) - severityOrder(b.cvss_severity)
    if (so !== 0) return so
    return (b.cvss_score ?? 0) - (a.cvss_score ?? 0)
  })
  const drift = ev.drift_items ?? []
  const driftRelated = findings.filter((f) => f.drift_related).length
  const critical = findings.filter((f) => f.cvss_severity === 'CRITICAL').length
  const high = findings.filter((f) => f.cvss_severity === 'HIGH').length

  return (
    <div className="space-y-8">
      {/* Header */}
      <div>
        <Link to={`/inventories/${ev.inventory_id}`} className="text-slate-400 hover:text-slate-600 text-sm">
          ← Inventory
        </Link>
        <h1 className="text-2xl font-bold mt-1">Evaluation</h1>
        <p className="text-slate-500 text-sm mt-1">
          {new Date(ev.evaluated_at).toLocaleString()}
        </p>
      </div>

      {/* Summary cards */}
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
        <div className="bg-white rounded-lg border border-slate-200 p-4">
          <div className={`text-3xl font-bold ${findings.length > 0 ? 'text-red-600' : 'text-green-600'}`}>
            {findings.length}
          </div>
          <div className="text-slate-500 text-sm mt-1">Total Findings</div>
        </div>
        <div className="bg-white rounded-lg border border-slate-200 p-4">
          <div className={`text-3xl font-bold ${critical > 0 ? 'text-red-700' : 'text-slate-300'}`}>
            {critical}
          </div>
          <div className="text-slate-500 text-sm mt-1">Critical</div>
        </div>
        <div className="bg-white rounded-lg border border-slate-200 p-4">
          <div className={`text-3xl font-bold ${high > 0 ? 'text-orange-500' : 'text-slate-300'}`}>
            {high}
          </div>
          <div className="text-slate-500 text-sm mt-1">High</div>
        </div>
        <div className="bg-white rounded-lg border border-slate-200 p-4">
          <div className={`text-3xl font-bold ${drift.length > 0 ? 'text-amber-600' : 'text-green-600'}`}>
            {drift.length}
          </div>
          <div className="text-slate-500 text-sm mt-1">Drift Items</div>
        </div>
      </div>

      {/* Findings table */}
      <section>
        <h2 className="text-lg font-semibold mb-3">
          Findings
          {driftRelated > 0 && (
            <span className="ml-2 text-sm text-amber-600 font-normal">⚠ {driftRelated} drift-related</span>
          )}
        </h2>
        <div className="bg-white rounded-lg border border-slate-200 overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-slate-50 border-b border-slate-200">
              <tr>
                {['Severity', 'Package', 'Installed', 'CVE', 'Status', 'Fixed in', 'Source', ''].map((h) => (
                  <th key={h} className="px-3 py-3 text-left text-xs font-semibold text-slate-500 uppercase tracking-wide">{h}</th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {findings.map((f, i) => <FindingRow key={i} f={f} />)}
            </tbody>
          </table>
          {findings.length === 0 && (
            <p className="text-center text-green-600 py-10 font-medium">No vulnerabilities found.</p>
          )}
        </div>
      </section>

      {/* Drift panel */}
      {drift.length > 0 && (
        <section>
          <h2 className="text-lg font-semibold mb-3">Drift from Golden Baseline</h2>
          <div className="bg-white rounded-lg border border-slate-200 overflow-hidden">
            <table className="w-full text-sm">
              <thead className="bg-slate-50 border-b border-slate-200">
                <tr>
                  {['Package', 'Kind', 'Host Version', 'Baseline Version'].map((h) => (
                    <th key={h} className="px-4 py-3 text-left text-xs font-semibold text-slate-500 uppercase tracking-wide">{h}</th>
                  ))}
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100">
                {drift.map((d, i) => <DriftRow key={i} d={d} />)}
              </tbody>
            </table>
          </div>
        </section>
      )}
    </div>
  )
}
