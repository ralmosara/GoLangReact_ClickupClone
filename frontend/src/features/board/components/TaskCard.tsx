import { useSortable } from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import { useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { api } from '../../../lib/api'
import { cn, formatRelative, PRIORITY_LABELS } from '../../../lib/utils'
import { PriorityBadge } from '../../../components/ui'
import type { Task, Tag } from '../../../types'

const PRIORITY_ACCENT: Record<number, string> = {
  0: '',
  1: 'border-l-blue-400',
  2: 'border-l-slate-400',
  3: 'border-l-orange-400',
  4: 'border-l-red-500',
}

export function TaskCard({ task, workspaceId }: { task: Task; workspaceId: string }) {
  const navigate = useNavigate()
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } =
    useSortable({ id: task.id, data: { type: 'task', task } })

  const { data: tags } = useQuery({
    queryKey: ['task-tags', task.id],
    queryFn: () => api.get(`tasks/${task.id}/tags`).json<Tag[]>(),
    staleTime: 60_000,
  })

  const { data: assignees } = useQuery({
    queryKey: ['task-assignees', task.id],
    queryFn: () => api.get(`tasks/${task.id}/assignees`).json<string[]>(),
    staleTime: 60_000,
  })

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
  }

  return (
    <div
      ref={setNodeRef}
      style={style}
      {...attributes}
      {...listeners}
      onClick={(e) => {
        // dnd-kit uses pointerdown listeners; ignore if a drag actually started.
        if (isDragging) return
        e.stopPropagation()
        navigate(`/workspaces/${workspaceId}/tasks/${task.id}`)
      }}
      className={cn(
        'group bg-surface border border-ink-5/30 rounded-xl p-3.5 cursor-grab active:cursor-grabbing',
        'shadow-card hover:shadow-card-hover hover:border-brand-300/50 transition-all duration-150',
        'border-l-4',
        PRIORITY_ACCENT[task.priority] ?? '',
        isDragging && 'opacity-40 shadow-modal scale-[1.02]',
      )}
    >
      <p className="text-sm font-medium text-ink-1 mb-1.5 group-hover:text-brand-600 line-clamp-2 leading-snug transition-colors">
        {task.name}
      </p>
      {task.description && (
        <p className="text-xs text-ink-4 mb-2 line-clamp-2 leading-relaxed">{task.description}</p>
      )}

      {(tags?.length ?? 0) > 0 && (
        <div className="flex flex-wrap gap-1 mb-2">
          {tags!.slice(0, 4).map((t) => (
            <span
              key={t.id}
              className="text-[10px] font-medium px-1.5 py-0.5 rounded"
              style={{
                backgroundColor: (t.color ?? '#94a3b8') + '22',
                color: t.color ?? '#64748b',
              }}
            >
              {t.name}
            </span>
          ))}
        </div>
      )}

      <div className="flex items-center flex-wrap gap-1.5 mt-2">
        {task.priority > 0 && <PriorityBadge priority={task.priority} />}
        {task.due_at && (
          <span className="text-[10px] text-amber-700 bg-amber-50 border border-amber-200 rounded-md px-1.5 py-0.5">
            {new Date(task.due_at).toLocaleDateString(undefined, { month: 'short', day: 'numeric' })}
          </span>
        )}
        <div className="ml-auto flex items-center -space-x-1.5">
          {(assignees ?? []).slice(0, 3).map((uid) => (
            <div
              key={uid}
              className="w-5 h-5 rounded-full bg-brand-gradient border-2 border-surface text-[9px] font-bold text-white flex items-center justify-center"
              title={uid}
            >
              {uid.slice(0, 1).toUpperCase()}
            </div>
          ))}
          {(assignees?.length ?? 0) > 3 && (
            <div className="w-5 h-5 rounded-full bg-ink-5/60 border-2 border-surface text-[9px] font-bold text-ink-2 flex items-center justify-center">
              +{assignees!.length - 3}
            </div>
          )}
        </div>
      </div>

      {task.updated_at && (
        <p className="text-[10px] text-ink-5 mt-2 pt-2 border-t border-ink-5/20">
          {formatRelative(task.updated_at)} · {PRIORITY_LABELS[task.priority] ?? 'No priority'}
        </p>
      )}
    </div>
  )
}
