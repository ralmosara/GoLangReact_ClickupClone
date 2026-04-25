import type { TimeReportBucket } from '../../../types'
import { formatHMS } from '../../time-tracking/components/TimeTracker'

export function TimePerUserWidget({ data }: { data: TimeReportBucket[] }) {
  if (!data || data.length === 0) {
    return <p className="text-xs text-ink-4 h-48 flex items-center justify-center">No time logged yet.</p>
  }
  const max = Math.max(...data.map((r) => r.total_s), 1)
  return (
    <div className="space-y-2 max-h-52 overflow-y-auto">
      {data.map((r) => (
        <div key={r.user_id} className="flex items-center gap-3">
          <div className="w-6 h-6 rounded-full bg-brand-gradient text-[10px] font-bold text-white flex items-center justify-center shrink-0">
            {(r.name || r.email || '?').slice(0, 1).toUpperCase()}
          </div>
          <div className="flex-1 min-w-0">
            <p className="text-xs font-medium text-ink-1 truncate">{r.name || r.email || r.user_id.slice(0, 8)}</p>
            <div className="h-1.5 rounded-full bg-ink-5/30 overflow-hidden mt-1">
              <div className="h-full bg-brand-500 rounded-full transition-all" style={{ width: `${(r.total_s / max) * 100}%` }} />
            </div>
          </div>
          <span className="text-xs font-semibold text-ink-1 tabular-nums">{formatHMS(r.total_s)}</span>
        </div>
      ))}
    </div>
  )
}
