import { useQuery } from '@tanstack/react-query'
import { api, SourceStatus } from '../api/client'

function Row({ s }: { s: SourceStatus }) {
  const syncedAgo = s.last_sync_at
    ? new Date(s.last_sync_at).toLocaleString()
    : 'Never'

  const statusDot = s.error
    ? 'bg-red-500'
    : s.last_sync_at
    ? 'bg-green-500'
    : 'bg-slate-300'

  return (
    <tr className="hover:bg-slate-50 transition-colors">
      <td className="px-4 py-3 font-medium text-sm flex items-center gap-2">
        <span className={`inline-block w-2 h-2 rounded-full ${statusDot}`} />
        {s.source}
      </td>
      <td className="px-4 py-3 text-slate-500 text-xs">{syncedAgo}</td>
      <td className="px-4 py-3 text-sm">{s.record_count.toLocaleString()}</td>
      <td className="px-4 py-3 text-xs text-red-600">{s.error || ''}</td>
    </tr>
  )
}

export default function Sources() {
  const { data, isLoading, error } = useQuery({
    queryKey: ['sources'],
    queryFn: api.listSources,
    refetchInterval: 30_000,
  })

  if (isLoading) return <p className="text-slate-500">Loading…</p>
  if (error) return <p className="text-red-600">{(error as Error).message}</p>

  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">Data Sources</h1>
      <div className="bg-white rounded-lg border border-slate-200 overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-slate-50 border-b border-slate-200">
            <tr>
              {['Source', 'Last Synced', 'Records', 'Error'].map((h) => (
                <th key={h} className="px-4 py-3 text-left text-xs font-semibold text-slate-500 uppercase tracking-wide">{h}</th>
              ))}
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-100">
            {data?.map((s) => <Row key={s.source} s={s} />)}
          </tbody>
        </table>
      </div>
    </div>
  )
}
