import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { api, Inventory } from '../api/client'

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

function Row({ inv }: { inv: Inventory }) {
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
      <td className="px-4 py-3 text-slate-400 text-xs">
        {new Date(inv.created_at).toLocaleString()}
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

  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">Inventories</h1>
      <div className="bg-white rounded-lg border border-slate-200 overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-slate-50 border-b border-slate-200">
            <tr>
              {['Name', 'Platform', 'Release', 'Type', 'Created'].map((h) => (
                <th key={h} className="px-4 py-3 text-left text-xs font-semibold text-slate-500 uppercase tracking-wide">
                  {h}
                </th>
              ))}
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-100">
            {data?.map((inv) => <Row key={inv.id} inv={inv} />)}
          </tbody>
        </table>
        {data?.length === 0 && (
          <p className="text-center text-slate-400 py-10">No inventories yet.</p>
        )}
      </div>
    </div>
  )
}
