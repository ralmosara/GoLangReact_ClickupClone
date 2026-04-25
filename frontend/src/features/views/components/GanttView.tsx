import { useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import { cn } from '../../../lib/utils'
import type { Status, Task, ViewConfig } from '../../../types'
import { applyView } from '../lib/applyView'

interface Props {
  tasks: Task[]
  statuses: Status[]
  workspaceId: string
  config: ViewConfig
  taskAssignees: Record<string, string[]>
  taskTagIds: Record<string, string[]>
}

const DAY_MS = 24 * 60 * 60 * 1000
const DAY_WIDTH = 28 // px

/**
 * Minimal custom SVG Gantt (MVP). Only tasks with both start_at and due_at (or
 * just due_at → treated as single-day) appear. Polished interactions (drag
 * edges, dependency arrows, zoom) land in M4 alongside task dependencies.
 */
export function GanttView({ tasks, statuses, workspaceId, config, taskAssignees, taskTagIds }: Props) {
  const navigate = useNavigate()
  const statusById = useMemo(() => new Map(statuses.map((s) => [s.id, s])), [statuses])
  const visible = applyView(tasks, config, taskTagIds, taskAssignees)

  const dated = visible
    .filter((t) => !!t.due_at)
    .map((t) => {
      const end = startOfDay(new Date(t.due_at!))
      const start = t.start_at ? startOfDay(new Date(t.start_at)) : end
      return { task: t, start, end }
    })
    .sort((a, b) => a.start.getTime() - b.start.getTime())

  if (dated.length === 0) {
    return (
      <div className="px-7 py-10 text-center text-sm text-ink-4">
        Add a due date to any task to see it on the Gantt chart.
      </div>
    )
  }

  // Compute timeline bounds with a small pad
  const minStart = new Date(Math.min(...dated.map((d) => d.start.getTime())) - 2 * DAY_MS)
  const maxEnd = new Date(Math.max(...dated.map((d) => d.end.getTime())) + 2 * DAY_MS)
  const days = Math.max(1, Math.round((maxEnd.getTime() - minStart.getTime()) / DAY_MS))

  const dayCols = Array.from({ length: days }, (_, i) => new Date(minStart.getTime() + i * DAY_MS))

  return (
    <div className="px-7 py-4">
      <div className="bg-surface border border-ink-5/30 rounded-2xl shadow-card overflow-hidden">
        <div className="flex">
          {/* Left: task names */}
          <div className="w-60 shrink-0 border-r border-ink-5/20">
            <div className="h-10 border-b border-ink-5/20 bg-canvas/40 flex items-center px-3 text-[10px] font-bold uppercase tracking-wider text-ink-4">
              Task
            </div>
            {dated.map(({ task }) => (
              <button
                key={task.id}
                onClick={() => navigate(`/workspaces/${workspaceId}/tasks/${task.id}`)}
                className="w-full h-9 border-b border-ink-5/10 text-left px-3 text-xs text-ink-1 hover:bg-ink-1/5 truncate"
              >
                {task.name}
              </button>
            ))}
          </div>

          {/* Right: timeline */}
          <div className="flex-1 overflow-x-auto">
            <div style={{ width: days * DAY_WIDTH, minWidth: '100%' }}>
              {/* header */}
              <div className="h-10 border-b border-ink-5/20 bg-canvas/40 flex">
                {dayCols.map((d, i) => {
                  const isFirstOfMonth = d.getDate() === 1
                  const isWeekend = d.getDay() === 0 || d.getDay() === 6
                  return (
                    <div
                      key={i}
                      className={cn(
                        'shrink-0 border-r border-ink-5/10 flex flex-col items-center justify-center text-[10px]',
                        isWeekend ? 'bg-ink-5/10 text-ink-4' : 'text-ink-2',
                        isFirstOfMonth && 'font-bold',
                      )}
                      style={{ width: DAY_WIDTH }}
                    >
                      <span>{d.getDate()}</span>
                      {isFirstOfMonth && (
                        <span className="text-ink-4">{d.toLocaleDateString(undefined, { month: 'short' })}</span>
                      )}
                    </div>
                  )
                })}
              </div>
              {/* rows */}
              {dated.map(({ task, start, end }) => {
                const offset = Math.round((start.getTime() - minStart.getTime()) / DAY_MS)
                const span = Math.max(1, Math.round((end.getTime() - start.getTime()) / DAY_MS) + 1)
                const st = task.status_id ? statusById.get(task.status_id) : undefined
                const color = st?.color ?? '#5B5CF8'
                return (
                  <div key={task.id} className="relative h-9 border-b border-ink-5/10">
                    {/* column grid */}
                    <div className="absolute inset-0 flex">
                      {dayCols.map((_, i) => (
                        <div key={i} className="shrink-0 border-r border-ink-5/10" style={{ width: DAY_WIDTH }} />
                      ))}
                    </div>
                    <button
                      onClick={() => navigate(`/workspaces/${workspaceId}/tasks/${task.id}`)}
                      className="absolute top-1.5 h-6 rounded shadow-card hover:shadow-card-hover transition-shadow px-2 flex items-center overflow-hidden text-[11px] font-medium text-white whitespace-nowrap"
                      style={{
                        left: offset * DAY_WIDTH + 2,
                        width: span * DAY_WIDTH - 4,
                        backgroundColor: color,
                      }}
                      title={task.name}
                    >
                      {task.name}
                    </button>
                  </div>
                )
              })}
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

function startOfDay(d: Date): Date {
  const out = new Date(d)
  out.setHours(0, 0, 0, 0)
  return out
}
