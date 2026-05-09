import { Link } from 'react-router-dom'
import { AlertCircle, Calendar, CheckSquare, Clock } from 'lucide-react'

import type { Task } from '../../../types'
import { cn, formatDate } from '../../../lib/utils'

// Bucket open tasks by relative due date so the eye lands on the
// overdue/today rows first. Computed client-side off the wire shape
// — the server returns an undifferentiated list ordered by due_at.
type BucketKey = 'overdue' | 'today' | 'this_week' | 'later'
const BUCKET_META: Record<BucketKey, { label: string; tone: string; Icon: typeof CheckSquare }> = {
  overdue:   { label: 'Overdue',   tone: 'text-red-600 dark:text-red-300',     Icon: AlertCircle },
  today:     { label: 'Today',     tone: 'text-amber-600 dark:text-amber-300', Icon: Clock },
  this_week: { label: 'This week', tone: 'text-brand-600 dark:text-brand-300', Icon: Calendar },
  later:     { label: 'Later',     tone: 'text-ink-3 dark:text-white/50',      Icon: CheckSquare },
}

function bucketFor(due?: string): BucketKey {
  if (!due) return 'later'
  const now = new Date()
  const d = new Date(due)
  const diffMs = d.getTime() - now.getTime()
  const dayMs = 86400000
  if (diffMs < 0) return 'overdue'
  if (d.toDateString() === now.toDateString()) return 'today'
  if (diffMs < 7 * dayMs) return 'this_week'
  return 'later'
}

export function MyTasksWidget({ data, workspaceId }: { data: Task[]; workspaceId?: string }) {
  if (!data || data.length === 0) {
    return <p className="text-xs text-ink-4 h-32 flex items-center justify-center dark:text-white/40">Nothing on your plate. 🎉</p>
  }

  const buckets: Record<BucketKey, Task[]> = { overdue: [], today: [], this_week: [], later: [] }
  for (const t of data) buckets[bucketFor(t.due_at)].push(t)

  return (
    <div className="space-y-3 max-h-72 overflow-y-auto pr-1">
      {(Object.keys(buckets) as BucketKey[]).map((k) => {
        const items = buckets[k]
        if (items.length === 0) return null
        const { label, tone, Icon } = BUCKET_META[k]
        return (
          <div key={k}>
            <p className={cn('text-[10px] font-bold uppercase tracking-wider flex items-center gap-1 mb-1', tone)}>
              <Icon className="w-3 h-3" /> {label} · {items.length}
            </p>
            <ul className="space-y-1">
              {items.map((t) => (
                <li key={t.id}>
                  <Link
                    to={workspaceId ? `/workspaces/${workspaceId}/tasks/${t.id}` : '#'}
                    className="flex items-center justify-between gap-2 px-2 py-1.5 rounded-lg hover:bg-ink-1/[0.04] dark:hover:bg-white/5"
                  >
                    <span className="text-sm text-ink-1 truncate dark:text-white">{t.name}</span>
                    {t.due_at && (
                      <span className="text-[10px] text-ink-4 shrink-0 tabular-nums dark:text-white/40">
                        {formatDate(t.due_at)}
                      </span>
                    )}
                  </Link>
                </li>
              ))}
            </ul>
          </div>
        )
      })}
    </div>
  )
}
