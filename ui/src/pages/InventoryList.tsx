import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { api, InventoryWithStats } from '../api/client'

const platformBadge: Record<string, string> = {
  ubuntu: 'bg-orange-100 text-orange-800',
  debian: 'bg-red-100 text-red-800',
  proxmox: 'bg-blue-100 text-blue-800',
}

const typeBadge: Record<string, string> = {
  host: 'bg-slate-100 text-slate-700',
  golden: 'bg-green-100 text-green-800',
}

function Badge({ label, cls }: { label: string; cls: string }) {
  return (
    <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${cls}`}>
      {label}
    </span>
  )
}

function FindingBadge({ count, fixable, evalId }: { count: number; fixable: number; evalId?: string }) {
  if (!evalId) return <span className="text-slate-300 text-xs">—</span>
  const cls = count === 0
    ? 'text-green-600 font-medium'
    : count >= 10 ? 'text-red-600 font-semibold' : 'text-amber-600 font-medium'
  return (
    <Link to={`/evaluations/${evalId}`} className="text-xs hover:underline flex items-center gap-1.5">
      <span className={cls}>{count}</span>
      {fixable > 0 && (
        <span className="text-amber-500 font-medium" title={`${fixable} fixable (upgrade available)`}>
          {fixable} fix
        </span>
      )}
    </Link>
  )
}

function Row({ inv }: { inv: InventoryWithStats }) {
  return (
    <tr className="hover:bg-slate-50 transition-colors">
      <td className="px-4 py-3 font-medium">
        <Link to={`/inventories/${inv.id}`} className="text-blue-600 hover:underline">
          {inv.name}
        </Link>
      </td>
      <td className="px-4 py-3">
        <Badge label={inv.platform} cls={platformBadge[inv.platform] ?? 'bg-slate-100 text-slate-700'} />
      </td>
      <td className="px-4 py-3 text-slate-600 text-sm">{inv.release}</td>
      <td className="px-4 py-3">
        <Badge label={inv.type} cls={typeBadge[inv.type] ?? ''} />
      </td>
      <td className="px-4 py-3 text-slate-500 text-xs tabular-nums">
        {inv.package_count.toLocaleString()}
      </td>
      <td className="px-4 py-3">
        <FindingBadge count={inv.last_finding_count} fixable={inv.last_fixable_count} evalId={inv.last_eval_id} />
      </td>
      <td className="px-4 py-3 text-slate-400 text-xs">
        {inv.last_evaluated_at
          ? new Date(inv.last_evaluated_at).toLocaleString()
          : <span className="text-slate-300">Never</span>}
      </td>
      <td className="px-4 py-3 text-slate-300 text-xs">
        {new Date(inv.created_at).toLocaleDateString()}
      </td>
    </tr>
  )
}

export default function InventoryList() {
  const { data, isLoading, error } = useQuery({
    queryKey: ['inventories'],
    queryFn: api.listInventories,
  })

  if (isLoading) return <p className="text-slate-500">Loading…</p>
  if (error) return <p className="text-red-600">Error: {(error as Error).message}</p>

  const hosts = data?.filter((i) => i.type === 'host') ?? []
  const goldens = data?.filter((i) => i.type === 'golden') ?? []

  return (
    <div className="space-y-8">
      <h1 className="text-2xl font-bold">Inventories</h1>

      {hosts.length > 0 && (
        <section>
          <h2 className="text-sm font-semibold text-slate-500 uppercase tracking-wide mb-2">Hosts</h2>
          <InventoryTable rows={hosts} />
        </section>
      )}

      {goldens.length > 0 && (
        <section>
          <h2 className="text-sm font-semibold text-slate-500 uppercase tracking-wide mb-2">Golden Baselines</h2>
          <InventoryTable rows={goldens} />
        </section>
      )}

      {(data?.length ?? 0) === 0 && (
        <p className="text-slate-400">No inventories yet. Use <code className="bg-slate-100 px-1 rounded">ingestion/collect.sh</code> to register a host.</p>
      )}
    </div>
  )
}

function InventoryTable({ rows }: { rows: InventoryWithStats[] }) {
  return (
    <div className="bg-white rounded-lg border border-slate-200 overflow-hidden">
      <table className="w-full text-sm">
        <thead className="bg-slate-50 border-b border-slate-200">
          <tr>
            {['Name', 'Platform', 'Release', 'Type', 'Packages', 'Findings', 'Last Evaluated', 'Created'].map((h) => (
              <th key={h} className="px-4 py-3 text-left text-xs font-semibold text-slate-500 uppercase tracking-wide">
                {h}
              </th>
            ))}
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-100">
          {rows.map((inv) => <Row key={inv.id} inv={inv} />)}
        </tbody>
      </table>
    </div>
  )
}
