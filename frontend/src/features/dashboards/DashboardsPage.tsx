import { useMemo, useState } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import { useParams, useSearchParams } from 'react-router-dom'
import { Plus, LayoutDashboard, Trash2 } from 'lucide-react'

import { api } from '../../lib/api'
import { queryClient } from '../../lib/queryClient'
import { Button, Input, Modal, PageSpinner } from '../../components/ui'
import { cn } from '../../lib/utils'
import type { Dashboard, Widget, WidgetKind, Space, List, Sprint } from '../../types'

import { BurndownWidget } from './components/BurndownWidget'
import { VelocityWidget } from './components/VelocityWidget'
import { TaskCountWidget } from './components/TaskCountWidget'
import { TimePerUserWidget } from './components/TimePerUserWidget'
import { MyTasksWidget } from './components/MyTasksWidget'
import { ActivityFeedWidget } from './components/ActivityFeedWidget'
import { GoalProgressWidget } from './components/GoalProgressWidget'

const WIDGET_META: Record<WidgetKind, { label: string; desc: string }> = {
  burndown:      { label: 'Burndown',      desc: 'Remaining story points over a sprint' },
  velocity:      { label: 'Velocity',      desc: 'Completed points per sprint' },
  task_count:    { label: 'Task count',    desc: 'Tasks grouped by status in a list' },
  time_per_user: { label: 'Time per user', desc: 'Hours logged by teammate' },
  my_tasks:      { label: 'My tasks',      desc: "Open tasks assigned to you, bucketed by due date" },
  activity:      { label: 'Activity',      desc: 'Recent mutations across the workspace' },
  goal_progress: { label: 'Goal progress', desc: 'Active goals with completion %' },
}

export function DashboardsPage() {
  const { workspaceId } = useParams<{ workspaceId: string }>()
  const [params, setParams] = useSearchParams()
  const activeId = params.get('id') ?? ''
  const [showCreate, setShowCreate] = useState(false)

  const { data: dashboards = [], isLoading } = useQuery({
    queryKey: ['dashboards', workspaceId],
    queryFn: () => api.get(`workspaces/${workspaceId}/dashboards`).json<Dashboard[]>(),
    enabled: !!workspaceId,
  })

  const active = dashboards.find((d) => d.id === activeId) ?? dashboards[0]

  return (
    <div className="flex h-full bg-canvas">
      <aside className="w-56 shrink-0 border-r border-ink-5/20 bg-surface/50 flex flex-col">
        <div className="flex items-center justify-between px-4 py-3 border-b border-ink-5/20">
          <h2 className="text-sm font-semibold text-ink-1 flex items-center gap-1.5">
            <LayoutDashboard className="w-3.5 h-3.5 text-brand-500" />
            Dashboards
          </h2>
          <button
            onClick={() => setShowCreate(true)}
            className="h-7 w-7 flex items-center justify-center rounded-lg text-ink-4 hover:text-brand-600 hover:bg-brand-50"
            title="New dashboard"
          >
            <Plus className="w-3.5 h-3.5" />
          </button>
        </div>
        <div className="flex-1 overflow-y-auto p-1">
          {isLoading && <PageSpinner />}
          {dashboards.map((d) => (
            <button
              key={d.id}
              onClick={() => setParams({ id: d.id }, { replace: true })}
              className={cn(
                'block w-full text-left px-3 py-1.5 rounded text-xs truncate transition-colors',
                (active?.id === d.id) ? 'bg-brand-50 text-brand-700 font-semibold' : 'text-ink-2 hover:bg-ink-1/5',
              )}
            >
              {d.name}
            </button>
          ))}
          {!isLoading && dashboards.length === 0 && <p className="text-xs text-ink-4 px-3 py-4">No dashboards yet.</p>}
        </div>
      </aside>

      <main className="flex-1 overflow-y-auto p-6">
        {active && workspaceId ? (
          <DashboardView dashboard={active} workspaceId={workspaceId} />
        ) : (
          <div className="h-full flex items-center justify-center text-sm text-ink-4">
            Create a dashboard to get started.
          </div>
        )}
      </main>

      {showCreate && workspaceId && (
        <CreateDashboardModal
          workspaceId={workspaceId}
          onClose={() => setShowCreate(false)}
          onCreated={(d) => setParams({ id: d.id }, { replace: true })}
        />
      )}
    </div>
  )
}

function DashboardView({ dashboard, workspaceId }: { dashboard: Dashboard; workspaceId: string }) {
  const [showAddWidget, setShowAddWidget] = useState(false)
  const { data: widgets = [] } = useQuery({
    queryKey: ['widgets', dashboard.id],
    queryFn: () => api.get(`dashboards/${dashboard.id}/widgets`).json<Widget[]>(),
  })

  const del = useMutation({
    mutationFn: (id: string) => api.delete(`widgets/${id}`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['widgets', dashboard.id] }),
  })

  return (
    <div>
      <div className="flex items-center justify-between mb-5">
        <h1 className="text-xl font-bold text-ink-1">{dashboard.name}</h1>
        <Button size="sm" onClick={() => setShowAddWidget(true)}>
          <Plus className="w-3.5 h-3.5 mr-1" /> Add widget
        </Button>
      </div>

      {widgets.length === 0 ? (
        <div className="text-center py-14 bg-surface/50 border border-ink-5/20 rounded-2xl text-sm text-ink-4">
          No widgets yet. Click "Add widget" to get started.
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {widgets.map((w) => (
            <div key={w.id} className="bg-surface border border-ink-5/30 rounded-2xl shadow-card p-4 group">
              <div className="flex items-center justify-between mb-3">
                <div>
                  <p className="text-[10px] font-bold uppercase tracking-wider text-ink-4">{WIDGET_META[w.kind]?.label ?? w.kind}</p>
                  <p className="text-sm font-semibold text-ink-1">{w.title || WIDGET_META[w.kind]?.label}</p>
                </div>
                <button
                  onClick={() => { if (confirm('Remove widget?')) del.mutate(w.id) }}
                  className="opacity-0 group-hover:opacity-100 text-ink-4 hover:text-red-500 p-1 rounded transition-all"
                >
                  <Trash2 className="w-3.5 h-3.5" />
                </button>
              </div>
              <WidgetRenderer widget={w} />
            </div>
          ))}
        </div>
      )}

      {showAddWidget && (
        <AddWidgetModal
          dashboardId={dashboard.id}
          workspaceId={workspaceId}
          onClose={() => setShowAddWidget(false)}
        />
      )}
    </div>
  )
}

function WidgetRenderer({ widget }: { widget: Widget }) {
  const { data, isLoading, error } = useQuery({
    queryKey: ['widget-data', widget.id],
    queryFn: () => api.get(`widgets/${widget.id}/data`).json<unknown>(),
    staleTime: 30_000,
  })
  if (isLoading) return <div className="h-48 flex items-center justify-center"><PageSpinner /></div>
  if (error)     return <p className="text-xs text-red-500">Failed to load widget data.</p>
  switch (widget.kind) {
    case 'burndown':      return <BurndownWidget data={data as never} />
    case 'velocity':      return <VelocityWidget data={data as never} />
    case 'task_count':    return <TaskCountWidget data={data as never} />
    case 'time_per_user': return <TimePerUserWidget data={data as never} />
    case 'my_tasks':      return <MyTasksWidget data={data as never} workspaceId={(widget.config as { workspace_id?: string })?.workspace_id} />
    case 'activity':      return <ActivityFeedWidget data={data as never} />
    case 'goal_progress': return <GoalProgressWidget data={data as never} />
    default:              return <p className="text-xs text-ink-4">Unsupported widget kind.</p>
  }
}

/* --- modals -------------------------------------------------------------- */

function CreateDashboardModal({
  workspaceId, onClose, onCreated,
}: {
  workspaceId: string
  onClose: () => void
  onCreated: (d: Dashboard) => void
}) {
  const [name, setName] = useState('')
  const create = useMutation({
    mutationFn: () => api.post(`workspaces/${workspaceId}/dashboards`, { json: { name } }).json<Dashboard>(),
    onSuccess: (d) => {
      queryClient.invalidateQueries({ queryKey: ['dashboards', workspaceId] })
      onCreated(d)
      onClose()
    },
  })
  return (
    <Modal open title="New dashboard" onClose={onClose}>
      <div className="space-y-3">
        <Input label="Name" placeholder="Team metrics" value={name} onChange={(e) => setName(e.target.value)} autoFocus />
        <div className="flex justify-end gap-2 pt-1">
          <Button variant="secondary" onClick={onClose}>Cancel</Button>
          <Button onClick={() => create.mutate()} disabled={!name.trim() || create.isPending} loading={create.isPending}>Create</Button>
        </div>
      </div>
    </Modal>
  )
}

function AddWidgetModal({
  dashboardId, workspaceId, onClose,
}: {
  dashboardId: string
  workspaceId: string
  onClose: () => void
}) {
  const [kind, setKind] = useState<WidgetKind>('task_count')
  const [title, setTitle] = useState('')
  const [listId, setListId] = useState('')
  const [sprintId, setSprintId] = useState('')

  const { data: spaces = [] } = useQuery({
    queryKey: ['spaces', workspaceId],
    queryFn: () => api.get(`workspaces/${workspaceId}/spaces`).json<Space[]>(),
  })
  const listsBySpace = useWorkspaceLists(spaces)
  const { data: sprintsForList = [] } = useQuery({
    queryKey: ['sprints', listId],
    queryFn: () => api.get(`lists/${listId}/sprints`).json<Sprint[]>(),
    enabled: !!listId && kind === 'burndown',
  })

  const create = useMutation({
    mutationFn: () => {
      const config: Record<string, unknown> = {}
      if (kind === 'task_count')    config.list_id = listId
      if (kind === 'velocity')      config.list_id = listId
      if (kind === 'burndown')      config.sprint_id = sprintId
      if (kind === 'time_per_user') config.workspace_id = workspaceId
      // Workspace-scoped widgets — viewer comes from the auth token
      // server-side, no list/sprint picker needed.
      if (kind === 'my_tasks')      config.workspace_id = workspaceId
      if (kind === 'activity')      config.workspace_id = workspaceId
      if (kind === 'goal_progress') config.workspace_id = workspaceId
      return api.post(`dashboards/${dashboardId}/widgets`, {
        json: { kind, title: title || WIDGET_META[kind].label, config },
      }).json<Widget>()
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['widgets', dashboardId] })
      onClose()
    },
  })

  // Widgets that don't need a per-list/per-sprint pick can be added
  // immediately. Listed by kind to keep the matrix obvious.
  const workspaceOnlyKinds: WidgetKind[] = ['time_per_user', 'my_tasks', 'activity', 'goal_progress']
  const canSubmit =
    workspaceOnlyKinds.includes(kind) ||
    (kind === 'burndown' && sprintId) ||
    ((kind === 'task_count' || kind === 'velocity') && listId)

  return (
    <Modal open title="Add widget" description="Pick a kind and wire it to the data source." onClose={onClose}>
      <div className="space-y-3">
        <div>
          <label className="block text-xs font-medium text-ink-2 mb-1">Kind</label>
          <div className="grid grid-cols-2 gap-2">
            {(Object.keys(WIDGET_META) as WidgetKind[]).map((k) => (
              <button
                key={k}
                onClick={() => setKind(k)}
                className={cn(
                  'text-left px-3 py-2 rounded-lg border transition-all',
                  kind === k ? 'border-brand-400 bg-brand-50' : 'border-ink-5/30 bg-surface hover:border-ink-5/60',
                )}
              >
                <p className="text-xs font-semibold text-ink-1">{WIDGET_META[k].label}</p>
                <p className="text-[10px] text-ink-4 mt-0.5">{WIDGET_META[k].desc}</p>
              </button>
            ))}
          </div>
        </div>

        <Input label="Title (optional)" value={title} onChange={(e) => setTitle(e.target.value)} placeholder={WIDGET_META[kind].label} />

        {(kind === 'task_count' || kind === 'velocity' || kind === 'burndown') && (
          <div>
            <label className="block text-xs font-medium text-ink-2 mb-1">List</label>
            <select
              className="h-9 w-full bg-surface border border-ink-5/40 rounded-lg px-3 text-sm focus:outline-none focus:ring-2 focus:ring-brand-400/40"
              value={listId}
              onChange={(e) => { setListId(e.target.value); setSprintId('') }}
            >
              <option value="">Pick a list…</option>
              {spaces.map((sp) => (
                <optgroup key={sp.id} label={sp.name}>
                  {(listsBySpace[sp.id] ?? []).map((l) => (
                    <option key={l.id} value={l.id}>{l.name}</option>
                  ))}
                </optgroup>
              ))}
            </select>
          </div>
        )}

        {kind === 'burndown' && listId && (
          <div>
            <label className="block text-xs font-medium text-ink-2 mb-1">Sprint</label>
            <select
              className="h-9 w-full bg-surface border border-ink-5/40 rounded-lg px-3 text-sm focus:outline-none focus:ring-2 focus:ring-brand-400/40"
              value={sprintId}
              onChange={(e) => setSprintId(e.target.value)}
            >
              <option value="">Pick a sprint…</option>
              {sprintsForList.map((s) => <option key={s.id} value={s.id}>{s.name}</option>)}
            </select>
          </div>
        )}

        <div className="flex justify-end gap-2 pt-1">
          <Button variant="secondary" onClick={onClose}>Cancel</Button>
          <Button onClick={() => create.mutate()} disabled={!canSubmit || create.isPending} loading={create.isPending}>
            Add
          </Button>
        </div>
      </div>
    </Modal>
  )
}

function useWorkspaceLists(spaces: Space[]): Record<string, List[]> {
  const spaceIds = spaces.map((s) => s.id).join(',')
  const { data } = useQuery({
    queryKey: ['widget-list-picker', spaceIds],
    queryFn: async () => {
      const entries: Record<string, List[]> = {}
      await Promise.all(spaces.map(async (s) => {
        entries[s.id] = await api.get(`spaces/${s.id}/lists`).json<List[]>()
      }))
      return entries
    },
    enabled: spaces.length > 0,
  })
  return data ?? {}
}
