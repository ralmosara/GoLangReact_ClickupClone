import { useMemo, useState } from 'react'
import {
  DndContext,
  DragOverlay,
  KeyboardSensor,
  PointerSensor,
  closestCenter,
  useDroppable,
  useSensor,
  useSensors,
  type DragEndEvent,
  type DragStartEvent,
} from '@dnd-kit/core'
import { SortableContext, sortableKeyboardCoordinates, verticalListSortingStrategy } from '@dnd-kit/sortable'
import { Plus } from 'lucide-react'

import { api } from '../../../lib/api'
import { queryClient } from '../../../lib/queryClient'
import { cn, STATUS_DOT, STATUS_LABEL } from '../../../lib/utils'
import type { Status, Task, ViewConfig } from '../../../types'
import { TaskCard } from '../../board/components/TaskCard'
import { useReorderTask } from '../../board/hooks/useReorderTask'
import { applyView } from '../lib/applyView'

const LEGACY_STATUSES = ['open', 'in_progress', 'review', 'completed'] as const

interface Column {
  key: string
  label: string
  color: string
  tasks: Task[]
  isLegacy: boolean
}

interface Props {
  tasks: Task[]
  statuses: Status[]
  workspaceId: string
  listId: string
  config: ViewConfig
  taskAssignees: Record<string, string[]>
  taskTagIds: Record<string, string[]>
  onAddTask: (statusId?: string) => void
}

export function BoardView({ tasks, statuses, workspaceId, listId, config, taskAssignees, taskTagIds, onAddTask }: Props) {
  const [activeTaskId, setActiveTaskId] = useState<string | null>(null)
  const reorder = useReorderTask()

  const visible = applyView(tasks, config, taskTagIds, taskAssignees)

  const columns: Column[] = useMemo(() => {
    if (statuses.length > 0) {
      return statuses
        .slice()
        .sort((a, b) => a.order_index - b.order_index)
        .map((s) => ({
          key: s.id,
          label: s.name,
          color: s.color,
          isLegacy: false,
          tasks: visible.filter((t) => t.status_id === s.id).sort((a, b) => a.position - b.position),
        }))
    }
    return LEGACY_STATUSES.map((s) => ({
      key: s,
      label: STATUS_LABEL[s] ?? s,
      color: STATUS_DOT[s] ?? '#94a3b8',
      isLegacy: true,
      tasks: visible.filter((t) => t.status === s).sort((a, b) => a.position - b.position),
    }))
  }, [visible, statuses])

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 4 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  )

  const activeTask = useMemo(() => tasks.find((t) => t.id === activeTaskId), [activeTaskId, tasks])

  const handleDragStart = (e: DragStartEvent) => setActiveTaskId(String(e.active.id))
  const handleDragEnd = (e: DragEndEvent) => {
    setActiveTaskId(null)
    const taskId = String(e.active.id)
    const overId = e.over ? String(e.over.id) : null
    if (!overId) return
    const task = tasks.find((t) => t.id === taskId)
    if (!task) return

    let targetColumn: Column | undefined
    let overIndex = -1
    if (overId.startsWith('col:')) {
      targetColumn = columns.find((c) => c.key === overId.slice(4))
    } else {
      targetColumn = columns.find((c) => c.tasks.some((t) => t.id === overId))
      if (targetColumn) overIndex = targetColumn.tasks.findIndex((t) => t.id === overId)
    }
    if (!targetColumn) return

    if (targetColumn.isLegacy) {
      if (task.status !== targetColumn.key) {
        reorder.mutate({
          taskId,
          listId,
          statusId: null, // Clear custom status_id
          status: targetColumn.key,
          prev: undefined,
          next: undefined,
        })
      }
      return
    }

    const targetTasks = targetColumn.tasks.filter((t) => t.id !== taskId)
    const insertAt = overIndex < 0 ? targetTasks.length : overIndex
    const prev = targetTasks[insertAt - 1]?.position
    const next = targetTasks[insertAt]?.position

    const statusChange = targetColumn.key === task.status_id ? undefined : targetColumn.key
    reorder.mutate({ taskId, listId, statusId: statusChange, prev, next })
  }

  return (
    <div className="flex-1 overflow-x-auto px-7 py-5">
      <DndContext sensors={sensors} collisionDetection={closestCenter} onDragStart={handleDragStart} onDragEnd={handleDragEnd}>
        <div className="flex gap-4 min-h-full">
          {columns.map((col) => (
            <ColumnView
              key={col.key}
              column={col}
              workspaceId={workspaceId}
              onAdd={() => onAddTask(col.isLegacy ? undefined : col.key)}
            />
          ))}
        </div>
        <DragOverlay>
          {activeTask && (
            <div className="opacity-90">
              <TaskCard task={activeTask} workspaceId={workspaceId} />
            </div>
          )}
        </DragOverlay>
      </DndContext>
    </div>
  )
}

function ColumnView({ column, workspaceId, onAdd }: { column: Column; workspaceId: string; onAdd: () => void }) {
  const { setNodeRef, isOver } = useDroppable({ id: `col:${column.key}`, data: { type: 'column', column } })
  return (
    <div className="w-[280px] shrink-0 flex flex-col animate-fade-in">
      <div className="flex items-center gap-2 mb-3 px-1">
        <div className="w-2 h-2 rounded-full shrink-0" style={{ backgroundColor: column.color }} />
        <span className="text-xs font-semibold text-ink-2">{column.label}</span>
        <span className="text-[11px] font-medium text-ink-4 bg-ink-5/30 rounded-full px-2 py-0.5 min-w-[22px] text-center">
          {column.tasks.length}
        </span>
        <button onClick={onAdd} className="ml-auto w-5 h-5 flex items-center justify-center rounded text-ink-4 hover:text-ink-1 hover:bg-ink-1/5 transition-colors" title="Add task to this column">
          <Plus className="w-3.5 h-3.5" />
        </button>
      </div>
      <div
        ref={setNodeRef}
        className={cn(
          'rounded-2xl border p-2 flex flex-col gap-2 flex-1 min-h-[200px] transition-colors',
          isOver ? 'bg-brand-50 border-brand-300' : 'bg-canvas/60 border-ink-5/20',
        )}
      >
        <SortableContext items={column.tasks.map((t) => t.id)} strategy={verticalListSortingStrategy}>
          {column.tasks.map((t) => (
            <TaskCard key={t.id} task={t} workspaceId={workspaceId} />
          ))}
        </SortableContext>
        {column.tasks.length === 0 && (
          <div className="flex items-center justify-center flex-1 py-8">
            <p className="text-xs text-ink-5 text-center">Drop tasks here</p>
          </div>
        )}
      </div>
    </div>
  )
}
