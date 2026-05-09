import type { AuditEntry } from '../../../types'
import { formatRelative } from '../../../lib/utils'

const VERB_TONE: Record<string, string> = {
  created: 'bg-green-50 text-green-700 ring-green-200 dark:bg-green-500/15 dark:text-green-300 dark:ring-green-500/30',
  updated: 'bg-blue-50 text-blue-700 ring-blue-200 dark:bg-blue-500/15 dark:text-blue-300 dark:ring-blue-500/30',
  deleted: 'bg-red-50 text-red-700 ring-red-200 dark:bg-red-500/15 dark:text-red-300 dark:ring-red-500/30',
}
function toneFor(verb: string) {
  return VERB_TONE[verb] ?? 'bg-ink-5/20 text-ink-2 ring-ink-5/30 dark:bg-white/10 dark:text-white/70 dark:ring-white/10'
}

export function ActivityFeedWidget({ data }: { data: AuditEntry[] }) {
  if (!data || data.length === 0) {
    return <p className="text-xs text-ink-4 h-32 flex items-center justify-center dark:text-white/40">No recent activity.</p>
  }
  return (
    <ul className="space-y-2 max-h-72 overflow-y-auto pr-1">
      {data.map((e) => (
        <li key={e.id} className="flex items-start gap-2">
          <span className={`text-[9px] font-bold uppercase tracking-wider px-1.5 py-0.5 rounded ring-1 shrink-0 ${toneFor(e.verb)}`}>
            {e.verb}
          </span>
          <div className="flex-1 min-w-0">
            <p className="text-xs text-ink-1 truncate dark:text-white/90">
              <span className="font-mono text-ink-2 dark:text-white/70">{e.entity_type}</span>
              {e.entity_id && <span className="text-ink-4 ml-1.5 dark:text-white/40">{e.entity_id.slice(0, 8)}</span>}
            </p>
            <p className="text-[10px] text-ink-4 dark:text-white/40">{formatRelative(e.created_at)}</p>
          </div>
        </li>
      ))}
    </ul>
  )
}
