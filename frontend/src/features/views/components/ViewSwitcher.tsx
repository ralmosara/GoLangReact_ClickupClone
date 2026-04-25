import { Kanban, List as ListIcon, CalendarDays, GanttChart, Table as TableIcon } from 'lucide-react'
import type { ViewKind } from '../../../types'
import { cn } from '../../../lib/utils'

export const VIEW_META: Record<ViewKind, { label: string; icon: React.ComponentType<{ className?: string }> }> = {
  board:    { label: 'Board',    icon: Kanban },
  list:     { label: 'List',     icon: ListIcon },
  calendar: { label: 'Calendar', icon: CalendarDays },
  gantt:    { label: 'Gantt',    icon: GanttChart },
  table:    { label: 'Table',    icon: TableIcon },
  timeline: { label: 'Timeline', icon: GanttChart },
}

const ORDER: ViewKind[] = ['board', 'list', 'calendar', 'gantt', 'table']

export function ViewSwitcher({
  current,
  onChange,
}: {
  current: ViewKind
  onChange: (kind: ViewKind) => void
}) {
  return (
    <div className="flex items-center gap-0.5 bg-ink-5/20 rounded-lg p-0.5">
      {ORDER.map((k) => {
        const meta = VIEW_META[k]
        const Icon = meta.icon
        const active = current === k
        return (
          <button
            key={k}
            onClick={() => onChange(k)}
            className={cn(
              'flex items-center gap-1.5 px-2.5 py-1 rounded text-xs font-medium transition-all',
              active
                ? 'bg-surface text-ink-1 shadow-card'
                : 'text-ink-3 hover:text-ink-1 hover:bg-surface/60',
            )}
            title={meta.label}
          >
            <Icon className="w-3.5 h-3.5" />
            <span className="hidden sm:inline">{meta.label}</span>
          </button>
        )
      })}
    </div>
  )
}
