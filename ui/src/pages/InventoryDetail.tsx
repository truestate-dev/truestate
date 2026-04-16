import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useParams, Link, useNavigate } from 'react-router-dom'
import { api, EvaluationSummary } from '../api/client'

function EvalRow({ ev }: { ev: EvaluationSummary }) {
  const findingColor =
    ev.finding_count === 0 ? 'text-green-600' : 'text-red-600 font-semibold'
  return (
    <tr className="hover:bg-slate-50 transition-colors">
      <td className="px-4 py-3 text-slate-400 text-xs">
        {new Date(ev.evaluated_at).toLocaleString()}
      </td>
      <td className={`px-4 py-3 text-sm ${findingColor}`}>{ev.finding_count}</td>
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
      navigate(`/evaluations/${eval_.id}`)
    },
  })

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
        <button
          onClick={() => run.mutate()}
          disabled={run.isPending}
          className="bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white px-4 py-2 rounded text-sm font-medium transition-colors"
        >
          {run.isPending ? 'Running…' : 'Run Evaluation'}
        </button>
      </div>

      {run.isError && (
        <p className="text-red-600 text-sm">{(run.error as Error).message}</p>
      )}

      {/* Packages */}
      <section>
        <h2 className="text-lg font-semibold mb-3">
          Packages <span className="text-slate-400 font-normal text-sm">({inv.packages?.length ?? 0})</span>
        </h2>
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
              {inv.packages?.map((p) => (
                <tr key={p.id} className="hover:bg-slate-50">
                  <td className="px-4 py-2 font-mono text-xs">{p.name}</td>
                  <td className="px-4 py-2 font-mono text-xs text-slate-500">{p.version}</td>
                  <td className="px-4 py-2 text-xs text-slate-400">{p.arch || '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>

      {/* Evaluation history */}
      <section>
        <h2 className="text-lg font-semibold mb-3">Evaluation History</h2>
        <div className="bg-white rounded-lg border border-slate-200 overflow-hidden">
          <table className="w-full text-sm">
            <thead className="bg-slate-50 border-b border-slate-200">
              <tr>
                {['Evaluated At', 'Findings', 'Drift Items', ''].map((h) => (
                  <th key={h} className="px-4 py-3 text-left text-xs font-semibold text-slate-500 uppercase tracking-wide">{h}</th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {evals?.map((ev) => <EvalRow key={ev.id} ev={ev} />)}
            </tbody>
          </table>
          {(evals?.length ?? 0) === 0 && (
            <p className="text-center text-slate-400 py-8 text-sm">No evaluations yet. Click "Run Evaluation" to start.</p>
          )}
        </div>
      </section>
    </div>
  )
}
