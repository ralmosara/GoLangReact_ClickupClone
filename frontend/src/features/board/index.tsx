import { useMemo, useState } from 'react'
import { useMutation, useQueries, useQuery } from '@tanstack/react-query'
import { useParams, useSearchParams } from 'react-router-dom'
import { Save } from 'lucide-react'

import { api } from '../../lib/api'
import { queryClient } from '../../lib/queryClient'
import { rooms } from '../../lib/ws'
import { useWsEvent, useWsRooms } from '../../hooks/useWebSocket'
import { Button, Input, Modal, PageSpinner } from '../../components/ui'
import { PRIORITY_LABELS } from '../../lib/utils'
import type { List, Status, Tag, Task, View, ViewConfig, ViewKind } from '../../types'

import { StatusManagerDialog } from './components/StatusManagerDialog'
import { VIEW_META, ViewSwitcher } from '../views/components/ViewSwitcher'
import { FilterBar } from '../views/components/FilterBar'
import { BoardView } from '../views/components/BoardView'
import { ListView } from '../views/components/ListView'
import { CalendarView } from '../views/components/CalendarView'
import { GanttView } from '../views/components/GanttView'
import { TableView } from '../views/components/TableView'
import { useCreateView, useDeleteView, useUpdateView, useViewsForList } from '../views/hooks/useViews'

function CreateTaskModal({
  listId,
  defaultStatusId,
  onClose,
}: {
  listId: string
  defaultStatusId?: string
  onClose: () => void
}) {
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [priority, setPriority] = useState(0)

  const mut = useMutation({
    mutationFn: () =>
      api.post('tasks', {
        json: { list_id: listId, name, description, priority, status_id: defaultStatusId ?? undefined },
      }).json<Task>(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tasks', listId] })
      onClose()
    },
  })

  return (
    <Modal open title="New Task" description="Add a task to this list." onClose={onClose}>
      <div className="space-y-4">
        <Input label="Task name" placeholder="What needs to be done?" value={name} onChange={(e) => setName(e.target.value)} autoFocus />
        <div>
          <label className="block text-xs font-medium text-ink-2 mb-2">Description</label>
          <textarea
            className="w-full bg-surface border border-ink-5/40 rounded-xl px-3 py-2.5 text-sm text-ink-1 h-20 resize-none focus:outline-none focus:ring-2 focus:ring-brand-400/40 focus:border-brand-400 placeholder:text-ink-4 transition-all"
            placeholder="Optional details…"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
          />
        </div>
        <div>
          <label className="block text-xs font-medium text-ink-2 mb-2">Priority</label>
          <select
            className="w-full h-9 bg-surface border border-ink-5/40 rounded-lg px-3 text-sm text-ink-1 focus:outline-none focus:ring-2 focus:ring-brand-400/40 focus:border-brand-400 transition-all"
            value={priority}
            onChange={(e) => setPriority(Number(e.target.value))}
          >
            {Object.entries(PRIORITY_LABELS).map(([v, l]) => (
              <option key={v} value={v}>{l}</option>
            ))}
          </select>
        </div>
        {mut.error && <p className="text-red-500 text-xs">{String(mut.error)}</p>}
        <div className="flex justify-end gap-2 pt-1">
          <Button variant="secondary" onClick={onClose}>Cancel</Button>
          <Button onClick={() => mut.mutate()} disabled={!name.trim() || mut.isPending} loading={mut.isPending}>Create task</Button>
        </div>
      </div>
    </Modal>
  )
}

function SaveViewModal({
  listId,
  kind,
  config,
  onSaved,
  onClose,
}: {
  listId: string
  kind: ViewKind
  config: ViewConfig
  onSaved: (v: View) => void
  onClose: () => void
}) {
  const [name, setName] = useState('')
  const create = useCreateView(listId)
  const handle = () => {
    if (!name.trim()) return
    create.mutate(
      { name: name.trim(), kind, config },
      {
        onSuccess: (v) => { onSaved(v); onClose() },
      },
    )
  }
  return (
    <Modal open title="Save view" description="Save this filter + sort + group setup for later." onClose={onClose}>
      <div className="space-y-3">
        <Input label="View name" placeholder="My open tasks" value={name} onChange={(e) => setName(e.target.value)} autoFocus onKeyDown={(e) => e.key === 'Enter' && handle()} />
        <div className="flex justify-end gap-2">
          <Button variant="secondary" onClick={onClose}>Cancel</Button>
          <Button onClick={handle} disabled={!name.trim() || create.isPending} loading={create.isPending}>Save view</Button>
        </div>
      </div>
    </Modal>
  )
}

export function BoardPage() {
  const { listId, workspaceId } = useParams<{ listId: string; workspaceId: string }>()
  const [searchParams, setSearchParams] = useSearchParams()
  const activeViewId = searchParams.get('view') ?? ''
  const kindOverride = searchParams.get('kind') as ViewKind | null

  const [createForStatus, setCreateForStatus] = useState<string | undefined>()
  const [showCreate, setShowCreate] = useState(false)
  const [showStatusMgr, setShowStatusMgr] = useState(false)
  const [showSaveView, setShowSaveView] = useState(false)
  const [localConfig, setLocalConfig] = useState<ViewConfig>({})
  const [localKind, setLocalKind] = useState<ViewKind>('board')

  const { data: list } = useQuery({
    queryKey: ['list', listId],
    queryFn: () => api.get(`lists/${listId}`).json<List>(),
    enabled: !!listId,
  })
  const { data: statuses = [] } = useQuery({
    queryKey: ['statuses', listId],
    queryFn: () => api.get(`lists/${listId}/statuses`).json<Status[]>(),
    enabled: !!listId,
  })
  const { data: tasks = [], isLoading } = useQuery({
    queryKey: ['tasks', listId],
    queryFn: () => api.get(`tasks?list_id=${listId}`).json<Task[]>(),
    enabled: !!listId,
  })
  const { data: workspaceTags = [] } = useQuery({
    queryKey: ['workspace-tags', workspaceId],
    queryFn: () => api.get(`workspaces/${workspaceId}/tags`).json<Tag[]>(),
    enabled: !!workspaceId,
  })
  const { data: views = [] } = useViewsForList(listId)

  // Pre-fetch tag & assignee membership per task (needed by FilterBar, ListView, TaskCard).
  const tagQueries = useQueries({
    queries: tasks.map((t) => ({
      queryKey: ['task-tags', t.id],
      queryFn: () => api.get(`tasks/${t.id}/tags`).json<Tag[]>(),
      staleTime: 60_000,
    })),
  })
  const assigneeQueries = useQueries({
    queries: tasks.map((t) => ({
      queryKey: ['task-assignees', t.id],
      queryFn: () => api.get(`tasks/${t.id}/assignees`).json<string[]>(),
      staleTime: 60_000,
    })),
  })

  const taskTagIds = useMemo(() => {
    const out: Record<string, string[]> = {}
    tasks.forEach((t, i) => {
      out[t.id] = (tagQueries[i]?.data ?? []).map((tg) => tg.id)
    })
    return out
  }, [tasks, tagQueries])
  const taskAssignees = useMemo(() => {
    const out: Record<string, string[]> = {}
    tasks.forEach((t, i) => {
      out[t.id] = assigneeQueries[i]?.data ?? []
    })
    return out
  }, [tasks, assigneeQueries])

  useWsRooms(listId ? [rooms.list(listId)] : [])
  useWsEvent('task.created', () => queryClient.invalidateQueries({ queryKey: ['tasks', listId] }))
  useWsEvent('task.updated', () => queryClient.invalidateQueries({ queryKey: ['tasks', listId] }))
  useWsEvent('task.deleted', () => queryClient.invalidateQueries({ queryKey: ['tasks', listId] }))
  useWsEvent('task.moved',   () => queryClient.invalidateQueries({ queryKey: ['tasks', listId] }))
  useWsEvent('status.created', () => queryClient.invalidateQueries({ queryKey: ['statuses', listId] }))
  useWsEvent('status.updated', () => queryClient.invalidateQueries({ queryKey: ['statuses', listId] }))
  useWsEvent('status.deleted', () => queryClient.invalidateQueries({ queryKey: ['statuses', listId] }))

  // Determine the currently active view & its config.
  const activeView = views.find((v) => v.id === activeViewId)
  const currentKind: ViewKind = activeView?.kind ?? kindOverride ?? localKind
  const currentConfig: ViewConfig = activeView?.config ?? localConfig

  const updateView = useUpdateView(listId)
  const delView = useDeleteView(listId)

  const handleConfigChange = (next: ViewConfig) => {
    if (activeView) {
      updateView.mutate({ id: activeView.id, config: next })
    } else {
      setLocalConfig(next)
    }
  }

  const handleKindChange = (k: ViewKind) => {
    if (activeView) {
      updateView.mutate({ id: activeView.id, kind: k })
    } else {
      setLocalKind(k)
      const p = new URLSearchParams(searchParams)
      p.set('kind', k)
      setSearchParams(p, { replace: true })
    }
  }

  const ViewRenderer = () => {
    const props = {
      tasks,
      statuses,
      workspaceId: workspaceId!,
      listId: listId!,
      config: currentConfig,
      taskAssignees,
      taskTagIds,
    }
    switch (currentKind) {
      case 'board':
        return <BoardView {...props} onAddTask={(sid) => { setCreateForStatus(sid); setShowCreate(true) }} />
      case 'list':
        return <ListView {...props} />
      case 'calendar':
        return <CalendarView {...props} />
      case 'gantt':
      case 'timeline':
        return <GanttView {...props} />
      case 'table':
        return <TableView {...props} />
      default:
        return <BoardView {...props} onAddTask={(sid) => { setCreateForStatus(sid); setShowCreate(true) }} />
    }
  }

  return (
    <div className="h-full flex flex-col bg-canvas">
      <div className="flex items-center justify-between px-7 py-4 border-b border-ink-5/20 bg-surface/50 backdrop-blur-sm">
        <div className="min-w-0">
          <h1 className="text-base font-semibold text-ink-1 truncate">{list?.name ?? 'Board'}</h1>
          <p className="text-xs text-ink-4 mt-0.5">
            {tasks.length} {tasks.length === 1 ? 'task' : 'tasks'}
            {statuses.length > 0 && <> · {statuses.length} statuses</>}
            {activeView && <> · Saved view “{activeView.name}”</>}
          </p>
        </div>
        <div className="flex items-center gap-2">
          <ViewSwitcher current={currentKind} onChange={handleKindChange} />
          <SavedViewsMenu
            views={views}
            activeId={activeViewId}
            onSelect={(id) => {
              const p = new URLSearchParams(searchParams)
              if (id) p.set('view', id)
              else p.delete('view')
              p.delete('kind')
              setSearchParams(p, { replace: true })
            }}
            onDelete={(id) => {
              if (confirm('Delete this saved view?')) delView.mutate(id)
            }}
          />
          <Button variant="secondary" size="sm" onClick={() => setShowSaveView(true)} title="Save current filters as a view">
            <Save className="w-3.5 h-3.5 mr-1.5" />
            Save
          </Button>
          <Button variant="secondary" size="sm" onClick={() => setShowStatusMgr(true)}>Statuses</Button>
          <Button onClick={() => { setCreateForStatus(undefined); setShowCreate(true) }} size="sm">Add Task</Button>
        </div>
      </div>

      <FilterBar
        config={currentConfig}
        onChange={handleConfigChange}
        statuses={statuses}
        tags={workspaceTags}
      />

      {isLoading ? <PageSpinner /> : <div className="flex-1 overflow-hidden flex flex-col"><ViewRenderer /></div>}

      {showCreate && listId && (
        <CreateTaskModal listId={listId} defaultStatusId={createForStatus} onClose={() => setShowCreate(false)} />
      )}
      {showStatusMgr && listId && (
        <StatusManagerDialog listId={listId} onClose={() => setShowStatusMgr(false)} />
      )}
      {showSaveView && listId && (
        <SaveViewModal
          listId={listId}
          kind={currentKind}
          config={currentConfig}
          onSaved={(v) => {
            const p = new URLSearchParams(searchParams)
            p.set('view', v.id)
            p.delete('kind')
            setSearchParams(p, { replace: true })
            setLocalConfig({})
          }}
          onClose={() => setShowSaveView(false)}
        />
      )}
    </div>
  )
}

function SavedViewsMenu({
  views,
  activeId,
  onSelect,
  onDelete,
}: {
  views: View[]
  activeId: string
  onSelect: (id: string) => void
  onDelete: (id: string) => void
}) {
  if (views.length === 0) return null
  return (
    <select
      className="h-8 px-2 bg-surface border border-ink-5/40 rounded-lg text-xs text-ink-1 focus:outline-none focus:ring-2 focus:ring-brand-400/40 max-w-[160px]"
      value={activeId}
      onChange={(e) => {
        const v = e.target.value
        if (v.startsWith('delete:')) {
          onDelete(v.slice(7))
        } else {
          onSelect(v)
        }
      }}
    >
      <option value="">Default view</option>
      {views.map((v) => (
        <option key={v.id} value={v.id}>
          {v.name} ({VIEW_META[v.kind]?.label ?? v.kind})
        </option>
      ))}
      {activeId && <option value={`delete:${activeId}`}>— Delete current view</option>}
    </select>
  )
}
