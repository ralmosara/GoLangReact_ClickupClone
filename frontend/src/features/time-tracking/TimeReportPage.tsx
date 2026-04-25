import { useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useParams } from 'react-router-dom'
import { api } from '../../lib/api'
import { PageSpinner } from '../../components/ui'
import type { TimeReportBucket } from '../../types'
import { formatHMS } from './components/TimeTracker'

export function TimeReportPage() {
  const { workspaceId } = useParams<{ workspaceId: string }>()

  // Default to last 30 days.
  const defaults = useMemo(() => {
    const to = new Date()
    const from = new Date(to.getTime() - 30 * 24 * 3600 * 1000)
    return { from: from.toISOString().slice(0, 10), to: to.toISOString().slice(0, 10) }
  }, [])
  const [from, setFrom] = useState(defaults.from)
  const [to, setTo] = useState(defaults.to)

  const { data: rows = [], isLoading } = useQuery({
    queryKey: ['time-report', workspaceId, from, to],
    queryFn: () =>
      api
        .get(`workspaces/${workspaceId}/time-report?from=${new Date(from).toISOString()}&to=${new Date(to).toISOString()}`)
        .json<TimeReportBucket[]>(),
    enabled: !!workspaceId,
  })

  const grandTotal = rows.reduce((s, r) => s + r.total_s, 0)
  const grandBillable = rows.reduce((s, r) => s + r.billable_s, 0)

  return (
    <div className="max-w-3xl mx-auto px-6 py-10 animate-slide-up">
      <h1 className="text-xl font-bold text-ink-1 mb-2">Time report</h1>
      <p className="text-sm text-ink-4 mb-6">Hours logged across this workspace grouped by teammate.</p>

      <div className="bg-surface border border-ink-5/30 rounded-2xl p-4 shadow-card mb-4 flex items-end gap-3">
        <div>
          <label className="block text-xs font-medium text-ink-2 mb-1">From</label>
          <input
            type="date"
            value={from}
            onChange={(e) => setFrom(e.target.value)}
            className="h-9 bg-surface border border-ink-5/40 rounded-lg px-3 text-sm focus:outline-none focus:ring-2 focus:ring-brand-400/40"
          />
        </div>
        <div>
          <label className="block text-xs font-medium text-ink-2 mb-1">To</label>
          <input
            type="date"
            value={to}
            onChange={(e) => setTo(e.target.value)}
            className="h-9 bg-surface border border-ink-5/40 rounded-lg px-3 text-sm focus:outline-none focus:ring-2 focus:ring-brand-400/40"
          />
        </div>
        <div className="ml-auto text-right">
          <p className="text-[10px] font-semibold uppercase tracking-wider text-ink-4">Total</p>
          <p className="text-2xl font-bold text-ink-1 tabular-nums">{formatHMS(grandTotal)}</p>
          {grandBillable > 0 && <p className="text-[11px] text-ink-4">{formatHMS(grandBillable)} billable</p>}
        </div>
      </div>

      {isLoading && <PageSpinner />}

      {!isLoading && rows.length === 0 && (
        <div className="text-center py-14 bg-surface/50 border border-ink-5/20 rounded-2xl text-sm text-ink-4">
          No time logged in this window.
        </div>
      )}

      {!isLoading && rows.length > 0 && (
        <div className="bg-surface border border-ink-5/30 rounded-2xl shadow-card overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-ink-5/20 bg-canvas/40 text-[10px] font-bold uppercase tracking-wider text-ink-4">
                <th className="text-left px-4 py-2">Teammate</th>
                <th className="text-right px-4 py-2">Tasks</th>
                <th className="text-right px-4 py-2">Billable</th>
                <th className="text-right px-4 py-2">Total</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((r) => (
                <tr key={r.user_id} className="border-b border-ink-5/10 last:border-b-0">
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-2">
                      <div className="w-7 h-7 rounded-full bg-brand-gradient text-[11px] font-bold text-white flex items-center justify-center">
                        {(r.name || r.email || '?').slice(0, 1).toUpperCase()}
                      </div>
                      <div className="min-w-0">
                        <p className="text-sm font-medium text-ink-1 truncate">{r.name || r.email || r.user_id.slice(0, 8)}</p>
                        {r.email && r.name && <p className="text-[11px] text-ink-4 truncate">{r.email}</p>}
                      </div>
                    </div>
                  </td>
                  <td className="text-right px-4 py-3 text-xs text-ink-2 tabular-nums">{r.task_count}</td>
                  <td className="text-right px-4 py-3 text-xs text-ink-2 tabular-nums">{formatHMS(r.billable_s)}</td>
                  <td className="text-right px-4 py-3 text-sm font-semibold text-ink-1 tabular-nums">{formatHMS(r.total_s)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
