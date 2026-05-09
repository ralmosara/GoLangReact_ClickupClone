import { Target } from 'lucide-react'

import { cn, formatDate } from '../../../lib/utils'

interface GoalProgressItem {
  goal_id: string
  name: string
  due_at?: string | null
  progress: number   // 0..1
  owner_id?: string | null
}

export function GoalProgressWidget({ data }: { data: GoalProgressItem[] }) {
  if (!data || data.length === 0) {
    return <p className="text-xs text-ink-4 h-32 flex items-center justify-center dark:text-white/40">No active goals.</p>
  }
  return (
    <ul className="space-y-3 max-h-72 overflow-y-auto pr-1">
      {data.map((g) => {
        const pct = Math.round(Math.max(0, Math.min(1, g.progress)) * 100)
        return (
          <li key={g.goal_id}>
            <div className="flex items-center justify-between mb-1">
              <p className="text-sm text-ink-1 truncate flex items-center gap-1.5 dark:text-white">
                <Target className="w-3 h-3 text-brand-500 shrink-0" />
                {g.name}
              </p>
              <span className="text-[11px] tabular-nums text-ink-3 shrink-0 dark:text-white/60">
                {pct}%
              </span>
            </div>
            <div className="h-1.5 rounded-full bg-ink-5/30 overflow-hidden dark:bg-white/10">
              <div
                className={cn(
                  'h-full rounded-full transition-all duration-300',
                  pct >= 100 ? 'bg-green-500' : pct >= 75 ? 'bg-brand-500' : pct >= 25 ? 'bg-amber-500' : 'bg-red-400',
                )}
                style={{ width: `${pct}%` }}
              />
            </div>
            {g.due_at && (
              <p className="text-[10px] text-ink-4 mt-1 dark:text-white/40">
                Due {formatDate(g.due_at)}
              </p>
            )}
          </li>
        )
      })}
    </ul>
  )
}
