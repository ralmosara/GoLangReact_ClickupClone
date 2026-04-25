import { useMemo, useState } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { Link2, Ban, X, AlertTriangle } from 'lucide-react'
import * as Popover from '@radix-ui/react-popover'

import { api } from '../../lib/api'
import { queryClient } from '../../lib/queryClient'
import { cn } from '../../lib/utils'
import { Input } from '../../components/ui'
import type { DependencyGraph, Task } from '../../types'

export function DependenciesPanel({
  taskId,
  listId,
  workspaceId,
}: {
  taskId: string
  listId: string
  workspaceId: string
}) {
  const { data: graph } = useQuery({
    queryKey: ['dependencies', taskId],
    queryFn: () => api.get(`tasks/${taskId}/dependencies`).json<DependencyGraph>(),
  })

  const { data: tasks = [] } = useQuery({
    queryKey: ['tasks', listId],
    queryFn: () => api.get(`tasks?list_id=${listId}`).json<Task[]>(),
    enabled: !!listId,
  })
  const taskById = useMemo(() => new Map(tasks.map((t) => [t.id, t])), [tasks])

  const add = useMutation({
    mutationFn: ({ dependsOnId, kind }: { dependsOnId: string; kind: 'waiting_on' }) =>
      api.post(`tasks/${taskId}/dependencies`, { json: { depends_on_id: dependsOnId, kind } }).json(),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['dependencies', taskId] }),
  })

  const remove = useMutation({
    mutationFn: (dependsOnId: string) =>
      api.delete(`tasks/${taskId}/dependencies/${dependsOnId}`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['dependencies', taskId] }),
  })

  return (
    <div className="space-y-4">
      {/* Waiting on */}
      <Section
        title="Waiting on"
        icon={<Link2 className="w-3.5 h-3.5" />}
        emptyHint="This task is not waiting on anything."
        rows={graph?.waiting_on ?? []}
        targetKey="depends_on_id"
        taskById={taskById}
        workspaceId={workspaceId}
        onRemove={(depId) => remove.mutate(depId)}
      />

      {/* Blocking */}
      <Section
        title="Blocking"
        icon={<Ban className="w-3.5 h-3.5" />}
        emptyHint="Nothing is waiting on this task."
        rows={graph?.blocking ?? []}
        targetKey="task_id"
        taskById={taskById}
        workspaceId={workspaceId}
        onRemove={() => { /* cannot remove from this side — have to open the other task */ }}
        readOnly
      />

      <AddDependency
        tasks={tasks.filter((t) => t.id !== taskId)}
        existing={new Set([
          ...(graph?.waiting_on ?? []).map((d) => d.depends_on_id),
        ])}
        onAdd={(dependsOnId) => add.mutate({ dependsOnId, kind: 'waiting_on' })}
        error={add.error ? String(add.error) : null}
      />
    </div>
  )
}

function Section({
  title,
  icon,
  emptyHint,
  rows,
  targetKey,
  taskById,
  workspaceId,
  onRemove,
  readOnly,
}: {
  title: string
  icon: React.ReactNode
  emptyHint: string
  rows: { task_id: string; depends_on_id: string }[]
  targetKey: 'depends_on_id' | 'task_id'
  taskById: Map<string, Task>
  workspaceId: string
  onRemove: (id: string) => void
  readOnly?: boolean
}) {
  const navigate = useNavigate()
  return (
    <div>
      <div className="flex items-center gap-1.5 text-[10px] font-bold uppercase tracking-wider text-ink-4 mb-1.5">
        {icon}
        {title}
        <span className="text-ink-5">({rows.length})</span>
      </div>
      {rows.length === 0 ? (
        <p className="text-[11px] text-ink-5 italic">{emptyHint}</p>
      ) : (
        <ul className="space-y-1">
          {rows.map((d) => {
            const targetId = d[targetKey]
            const t = taskById.get(targetId)
            return (
              <li
                key={`${d.task_id}:${d.depends_on_id}`}
                className="flex items-center gap-2 px-2.5 py-1.5 bg-canvas/60 border border-ink-5/20 rounded-lg group"
              >
                <button
                  onClick={() => navigate(`/workspaces/${workspaceId}/tasks/${targetId}`)}
                  className="flex-1 text-left text-xs text-ink-1 hover:text-brand-600 truncate"
                >
                  {t?.name ?? targetId.slice(0, 8)}
                </button>
                {!readOnly && (
                  <button
                    onClick={() => onRemove(d.depends_on_id)}
                    className="opacity-0 group-hover:opacity-100 text-ink-4 hover:text-red-500 transition-opacity"
                  >
                    <X className="w-3.5 h-3.5" />
                  </button>
                )}
              </li>
            )
          })}
        </ul>
      )}
    </div>
  )
}

function AddDependency({
  tasks,
  existing,
  onAdd,
  error,
}: {
  tasks: Task[]
  existing: Set<string>
  onAdd: (id: string) => void
  error: string | null
}) {
  const [open, setOpen] = useState(false)
  const [query, setQuery] = useState('')

  const filtered = tasks
    .filter((t) => !existing.has(t.id))
    .filter((t) => t.name.toLowerCase().includes(query.toLowerCase()))
    .slice(0, 50)

  return (
    <Popover.Root open={open} onOpenChange={setOpen}>
      <Popover.Trigger asChild>
        <button
          className={cn(
            'inline-flex items-center gap-1 h-7 px-2.5 rounded-lg text-[11px] font-medium',
            'border border-dashed border-ink-5/50 text-ink-4 hover:text-ink-1 hover:border-brand-300 hover:bg-canvas/60',
          )}
        >
          + Add "waiting on" dependency
        </button>
      </Popover.Trigger>
      <Popover.Portal>
        <Popover.Content sideOffset={4} className="bg-surface border border-ink-5/30 rounded-xl shadow-modal w-72 p-2 max-h-80 overflow-y-auto z-50">
          <Input placeholder="Find task…" value={query} onChange={(e) => setQuery(e.target.value)} />
          <div className="mt-2 space-y-0.5">
            {filtered.map((t) => (
              <button
                key={t.id}
                onClick={() => { onAdd(t.id); setOpen(false) }}
                className="block w-full text-left px-2 py-1.5 rounded text-xs text-ink-1 hover:bg-ink-1/5 truncate"
              >
                {t.name}
              </button>
            ))}
            {filtered.length === 0 && <p className="text-[11px] text-ink-4 px-2 py-2">No matching tasks.</p>}
          </div>
          {error && (
            <div className="mt-2 flex items-start gap-1.5 text-[11px] text-red-600 bg-red-50 border border-red-200 rounded px-2 py-1.5">
              <AlertTriangle className="w-3 h-3 mt-0.5 shrink-0" />
              <span>{error.includes('cycle') ? 'Adding this dependency would create a cycle.' : error}</span>
            </div>
          )}
        </Popover.Content>
      </Popover.Portal>
    </Popover.Root>
  )
}
