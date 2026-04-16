import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useState, useMemo } from 'react'
import { useParams, Link, useNavigate } from 'react-router-dom'
import { api, EvaluationSummary } from '../api/client'
import TrendChart from '../components/TrendChart'

const PAGE_SIZE = 100

function EvalRow({ ev }: { ev: EvaluationSummary }) {
  const findingColor =
    ev.finding_count === 0 ? 'text-green-600' : 'text-red-600 font-semibold'
  return (
    <tr className="hover:bg-slate-50 transition-colors">
      <td className="px-4 py-3 text-slate-400 text-xs">
        {new Date(ev.evaluated_at).toLocaleString()}
      </td>
      <td className={`px-4 py-3 text-sm ${findingColor}`}>{ev.finding_count}</td>
      <td className="px-4 py-3 text-sm">
        {ev.fixable_count > 0
          ? <span className="text-amber-600 font-medium">{ev.fixable_count}</span>
          : <span className="text-slate-300">—</span>}
      </td>
      <td className="px-4 py-3 text-sm text-slate-600">{ev.drift_count}</td>
      <td className="px-4 py-3">
        <Link to={`/evaluations/${ev.id}`} className="text-blue-600 hover:underline text-sm">
          View →
        </Link>
      </td>
    </tr>
  )
}

export default function InventoryDetail() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const qc = useQueryClient()

  const [pkgSearch, setPkgSearch] = useState('')
  const [pkgPage, setPkgPage] = useState(0)
  const [confirmDelete, setConfirmDelete] = useState(false)

  const { data: inv, isLoading } = useQuery({
    queryKey: ['inventory', id],
    queryFn: () => api.getInventory(id!),
  })

  const { data: evals } = useQuery({
    queryKey: ['evaluations', id],
    queryFn: () => api.listEvaluations(id!),
  })

  const run = useMutation({
    mutationFn: () => api.runEvaluation(id!),
    onSuccess: (eval_) => {
      qc.invalidateQueries({ queryKey: ['evaluations', id] })
      qc.invalidateQueries({ queryKey: ['inventories'] })
      navigate(`/evaluations/${eval_.id}`)
    },
  })

  const del = useMutation({
    mutationFn: () => api.deleteInventory(id!),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['inventories'] })
      navigate('/')
    },
  })

  const filteredPackages = useMemo(() => {
    const pkgs = inv?.packages ?? []
    if (!pkgSearch.trim()) return pkgs
    const q = pkgSearch.trim().toLowerCase()
    return pkgs.filter((p) => p.name.includes(q) || p.version.includes(q))
  }, [inv?.packages, pkgSearch])

  const pkgPageCount = Math.ceil(filteredPackages.length / PAGE_SIZE)
  const pkgSlice = filteredPackages.slice(pkgPage * PAGE_SIZE, (pkgPage + 1) * PAGE_SIZE)

  if (isLoading) return <p className="text-slate-500">Loading…</p>
  if (!inv) return <p className="text-red-600">Inventory not found.</p>

  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="flex items-start justify-between">
        <div>
          <Link to="/" className="text-slate-400 hover:text-slate-600 text-sm">← Inventories</Link>
          <h1 className="text-2xl font-bold mt-1">{inv.name}</h1>
          <p className="text-slate-500 text-sm mt-1">
            {inv.platform} / {inv.release} · {inv.type}
          </p>
        </div>
        <div className="flex items-center gap-3">
          {confirmDelete ? (
            <>
              <span className="text-xs text-slate-500">Delete this inventory?</span>
              <button
                onClick={() => del.mutate()}
                disabled={del.isPending}
                className="bg-red-600 hover:bg-red-700 disabled:opacity-50 text-white px-3 py-2 rounded text-sm font-medium"
              >
                {del.isPending ? 'Deleting…' : 'Confirm'}
              </button>
              <button
                onClick={() => setConfirmDelete(false)}
                className="text-slate-400 hover:text-slate-600 text-sm px-2 py-2"
              >
                Cancel
              </button>
            </>
          ) : (
            <button
              onClick={() => setConfirmDelete(true)}
              className="text-slate-400 hover:text-red-600 text-sm px-2 py-2 transition-colors"
              title="Delete inventory"
            >
              Delete
            </button>
          )}
          <button
            onClick={() => run.mutate()}
            disabled={run.isPending}
            className="bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white px-4 py-2 rounded text-sm font-medium transition-colors"
          >
            {run.isPending ? 'Running…' : 'Run Evaluation'}
          </button>
        </div>
      </div>

      {run.isError && (
        <p className="text-red-600 text-sm">{(run.error as Error).message}</p>
      )}
      {del.isError && (
        <p className="text-red-600 text-sm">{(del.error as Error).message}</p>
      )}

      {/* Packages */}
      <section>
        <div className="flex items-center justify-between mb-3">
          <h2 className="text-lg font-semibold">
            Packages
            <span className="text-slate-400 font-normal text-sm ml-2">
              {filteredPackages.length !== (inv.packages?.length ?? 0)
                ? `${filteredPackages.length} of ${inv.packages?.length ?? 0}`
                : (inv.packages?.length ?? 0).toLocaleString()}
            </span>
          </h2>
          <input
            type="text"
            placeholder="Search packages…"
            value={pkgSearch}
            onChange={(e) => { setPkgSearch(e.target.value); setPkgPage(0) }}
            className="text-sm px-3 py-1.5 border border-slate-200 rounded focus:outline-none focus:ring-1 focus:ring-slate-400 w-56"
          />
        </div>

        <div className="bg-white rounded-lg border border-slate-200 overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-slate-50 border-b border-slate-200">
              <tr>
                {['Name', 'Version', 'Arch'].map((h) => (
                  <th key={h} className="px-4 py-3 text-left text-xs font-semibold text-slate-500 uppercase tracking-wide">{h}</th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {pkgSlice.map((p) => (
                <tr key={p.id} className="hover:bg-slate-50">
                  <td className="px-4 py-2 font-mono text-xs">{p.name}</td>
                  <td className="px-4 py-2 font-mono text-xs text-slate-500">{p.version}</td>
                  <td className="px-4 py-2 text-xs text-slate-400">{p.arch || '—'}</td>
                </tr>
              ))}
              {pkgSlice.length === 0 && (
                <tr>
                  <td colSpan={3} className="px-4 py-6 text-center text-slate-400 text-sm">
                    {pkgSearch ? 'No packages match the search.' : 'No packages.'}
                  </td>
                </tr>
              )}
            </tbody>
          </table>

          {/* Pagination */}
          {pkgPageCount > 1 && (
            <div className="flex items-center justify-between px-4 py-2 border-t border-slate-100 bg-slate-50 text-xs text-slate-500">
              <span>
                {pkgPage * PAGE_SIZE + 1}–{Math.min((pkgPage + 1) * PAGE_SIZE, filteredPackages.length)} of {filteredPackages.length}
              </span>
              <div className="flex gap-2">
                <button
                  onClick={() => setPkgPage((p) => Math.max(0, p - 1))}
                  disabled={pkgPage === 0}
                  className="px-2 py-1 rounded border border-slate-200 disabled:opacity-40 hover:bg-slate-100"
                >
                  ← Prev
                </button>
                <button
                  onClick={() => setPkgPage((p) => Math.min(pkgPageCount - 1, p + 1))}
                  disabled={pkgPage >= pkgPageCount - 1}
                  className="px-2 py-1 rounded border border-slate-200 disabled:opacity-40 hover:bg-slate-100"
                >
                  Next →
                </button>
              </div>
            </div>
          )}
        </div>
      </section>

      {/* Trend chart */}
      {(evals?.length ?? 0) >= 2 && (
        <section>
          <h2 className="text-lg font-semibold mb-3">Finding Trend</h2>
          <div className="bg-white rounded-lg border border-slate-200 p-4">
            <TrendChart data={evals!} />
          </div>
        </section>
      )}

      {/* Evaluation history */}
      <section>
        <h2 className="text-lg font-semibold mb-3">Evaluation History</h2>
        <div className="bg-white rounded-lg border border-slate-200 overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-slate-50 border-b border-slate-200">
              <tr>
                {['Evaluated At', 'Findings', 'Fixable', 'Drift Items', ''].map((h) => (
                  <th key={h} className="px-4 py-3 text-left text-xs font-semibold text-slate-500 uppercase tracking-wide">{h}</th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {evals?.map((ev) => <EvalRow key={ev.id} ev={ev} />)}
            </tbody>
          </table>
          {(evals?.length ?? 0) === 0 && (
            <p className="text-center text-slate-400 py-8 text-sm">
              No evaluations yet. Click "Run Evaluation" to start.
            </p>
          )}
        </div>
      </section>
    </div>
  )
}
