import { useNavigate } from 'react-router-dom'
import { useMutation } from '@tanstack/react-query'
import { ChevronRight } from 'lucide-react'
import { api } from '../../../lib/api'
import { queryClient } from '../../../lib/queryClient'
import { cn, formatRelative, PRIORITY_COLORS, PRIORITY_LABELS } from '../../../lib/utils'
import type { Status, Task, ViewConfig } from '../../../types'
import { applyView, groupTasks } from '../lib/applyView'

interface Props {
  tasks: Task[]
  statuses: Status[]
  workspaceId: string
  listId: string
  config: ViewConfig
  taskAssignees: Record<string, string[]>
  taskTagIds: Record<string, string[]>
}

export function ListView({ tasks, statuses, workspaceId, listId, config, taskAssignees, taskTagIds }: Props) {
  const navigate = useNavigate()
  const visible = applyView(tasks, config, taskTagIds, taskAssignees)
  const groups = groupTasks(visible, config.group_by, { taskAssignees })

  const statusById = new Map(statuses.map((s) => [s.id, s]))

  const updateStatus = useMutation({
    mutationFn: ({ id, statusID }: { id: string; statusID: string }) =>
      api.patch(`tasks/${id}`, { json: { status_id: statusID } }).json<Task>(),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['tasks', listId] }),
  })

  return (
    <div className="px-7 py-4 space-y-5">
      {groups.map((g) => (
        <div key={g.key}>
          {config.group_by && (
            <div className="flex items-center gap-2 mb-2">
              <ChevronRight className="w-3.5 h-3.5 text-ink-4" />
              <h3 className="text-xs font-semibold text-ink-2">
                {groupLabel(g.key, g.label, config.group_by, statusById)}
              </h3>
              <span className="text-[11px] text-ink-4">{g.tasks.length}</span>
            </div>
          )}
          <div className="bg-surface border border-ink-5/30 rounded-xl shadow-card overflow-hidden">
            <div className="grid grid-cols-[28px_minmax(0,1fr)_120px_120px_100px_90px] gap-3 px-4 py-2 text-[10px] font-bold uppercase tracking-wider text-ink-4 border-b border-ink-5/20 bg-canvas/40">
              <div></div>
              <div>Name</div>
              <div>Status</div>
              <div>Assignees</div>
              <div>Priority</div>
              <div>Due</div>
            </div>
            {g.tasks.map((t) => {
              const st = t.status_id ? statusById.get(t.status_id) : undefined
              const assignees = taskAssignees[t.id] ?? []
              const done = st?.category === 'done' || t.status === 'completed'
              return (
                <button
                  key={t.id}
                  onClick={() => navigate(`/workspaces/${workspaceId}/tasks/${t.id}`)}
                  className="grid grid-cols-[28px_minmax(0,1fr)_120px_120px_100px_90px] gap-3 px-4 py-2 text-sm border-b border-ink-5/10 last:border-b-0 hover:bg-ink-1/2 text-left items-center w-full group"
                >
                  <button
                    onClick={(e) => {
                      e.stopPropagation()
                      // Toggle to "done"-category status if available
                      const doneStatus = statuses.find((s) => s.category === 'done')
                      if (doneStatus) updateStatus.mutate({ id: t.id, statusID: doneStatus.id })
                    }}
                    className={cn(
                      'w-4 h-4 rounded border flex items-center justify-center shrink-0 transition-colors',
                      done ? 'bg-brand-500 border-brand-500' : 'border-ink-5/60 hover:border-brand-400',
                    )}
                  >
                    {done && (
                      <svg className="w-2.5 h-2.5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={3} d="M5 13l4 4L19 7" />
                      </svg>
                    )}
                  </button>
                  <div className="min-w-0">
                    <p className={cn('text-sm truncate', done ? 'text-ink-4 line-through' : 'text-ink-1 group-hover:text-brand-600')}>{t.name}</p>
                    {t.description && (
                      <p className="text-[11px] text-ink-4 truncate">{stripHtml(t.description)}</p>
                    )}
                  </div>
                  <div>
                    {st ? (
                      <span
                        className="inline-flex items-center gap-1 text-[11px] font-medium px-2 py-0.5 rounded-full"
                        style={{ backgroundColor: st.color + '22', color: st.color }}
                      >
                        <span className="w-1.5 h-1.5 rounded-full" style={{ backgroundColor: st.color }} />
                        {st.name}
                      </span>
                    ) : (
                      <span className="text-[11px] text-ink-4 capitalize">{t.status.replace('_', ' ')}</span>
                    )}
                  </div>
                  <div className="flex -space-x-1.5">
                    {assignees.slice(0, 3).map((uid) => (
                      <div key={uid} className="w-6 h-6 rounded-full bg-brand-gradient text-[10px] font-bold text-white flex items-center justify-center border-2 border-surface">
                        {uid.slice(0, 1).toUpperCase()}
                      </div>
                    ))}
                    {assignees.length > 3 && (
                      <div className="w-6 h-6 rounded-full bg-ink-5/60 text-[10px] font-bold text-ink-2 flex items-center justify-center border-2 border-surface">
                        +{assignees.length - 3}
                      </div>
                    )}
                    {assignees.length === 0 && <span className="text-[11px] text-ink-4">—</span>}
                  </div>
                  <span className={cn('text-[11px] font-medium px-2 py-0.5 rounded-full inline-flex self-start', PRIORITY_COLORS[t.priority] ?? PRIORITY_COLORS[0])}>
                    {PRIORITY_LABELS[t.priority] ?? 'No priority'}
                  </span>
                  <span className="text-[11px] text-ink-4">
                    {t.due_at ? new Date(t.due_at).toLocaleDateString(undefined, { month: 'short', day: 'numeric' }) : formatRelative(t.updated_at)}
                  </span>
                </button>
              )
            })}
            {g.tasks.length === 0 && (
              <div className="px-4 py-6 text-center text-xs text-ink-4">No tasks match current filters.</div>
            )}
          </div>
        </div>
      ))}
    </div>
  )
}

function stripHtml(html: string): string {
  return html.replace(/<[^>]*>/g, ' ').replace(/\s+/g, ' ').trim()
}

function groupLabel(key: string, label: string, groupBy: string, statusById: Map<string, Status>): string {
  if (label) return label
  if (groupBy === 'status_id') {
    if (key === '__none') return 'No status'
    return statusById.get(key)?.name ?? 'Unknown'
  }
  if (groupBy === 'priority') {
    return PRIORITY_LABELS[Number(key)] ?? `Priority ${key}`
  }
  if (groupBy === 'assignee') {
    if (key === '__none') return 'Unassigned'
    return key.slice(0, 8)
  }
  return key
}
