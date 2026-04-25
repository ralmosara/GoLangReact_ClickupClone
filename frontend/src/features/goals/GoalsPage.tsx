import { useMemo, useState } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import { useParams } from 'react-router-dom'
import { Plus, ChevronRight, ChevronDown, Target, Trash2, Check } from 'lucide-react'

import { api } from '../../lib/api'
import { queryClient } from '../../lib/queryClient'
import { Button, Input, Modal, PageSpinner } from '../../components/ui'
import { cn } from '../../lib/utils'
import type { Goal, GoalTarget, List, Space, TargetKind } from '../../types'

const TARGET_KINDS: Array<{ value: TargetKind; label: string; hint: string }> = [
  { value: 'number',         label: 'Number',    hint: 'e.g. 100 signups' },
  { value: 'currency',       label: 'Currency',  hint: 'e.g. $10,000 MRR' },
  { value: 'boolean',        label: 'Yes / No',  hint: 'ship by launch day' },
  { value: 'task_completed', label: 'Task list', hint: 'percent of tasks done' },
]

export function GoalsPage() {
  const { workspaceId } = useParams<{ workspaceId: string }>()
  const [showCreate, setShowCreate] = useState(false)

  const { data: goals = [], isLoading } = useQuery({
    queryKey: ['goals', workspaceId],
    queryFn: () => api.get(`workspaces/${workspaceId}/goals`).json<Goal[]>(),
    enabled: !!workspaceId,
  })

  const tree = useMemo(() => buildGoalTree(goals), [goals])

  return (
    <div className="max-w-4xl mx-auto px-6 py-10 animate-slide-up">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-xl font-bold text-ink-1 flex items-center gap-2">
            <Target className="w-5 h-5 text-brand-500" />
            Goals
          </h1>
          <p className="text-sm text-ink-4 mt-0.5">OKR-style goals with measurable targets.</p>
        </div>
        <Button onClick={() => setShowCreate(true)} size="sm">
          <Plus className="w-3.5 h-3.5 mr-1.5" />
          New goal
        </Button>
      </div>

      {isLoading && <PageSpinner />}

      {!isLoading && goals.length === 0 && (
        <div className="text-center py-14 bg-surface/50 border border-ink-5/20 rounded-2xl text-sm text-ink-4">
          No goals yet. Create your first one.
        </div>
      )}

      {tree.length > 0 && (
        <div className="space-y-2">
          {tree.map((node) => (
            <GoalCard key={node.goal.id} node={node} workspaceId={workspaceId!} depth={0} />
          ))}
        </div>
      )}

      {showCreate && workspaceId && (
        <CreateGoalModal workspaceId={workspaceId} goals={goals} onClose={() => setShowCreate(false)} />
      )}
    </div>
  )
}

interface GoalNode { goal: Goal; children: GoalNode[] }

function buildGoalTree(goals: Goal[]): GoalNode[] {
  const byParent = new Map<string, Goal[]>()
  for (const g of goals) {
    const key = g.parent_goal_id ?? '__root'
    if (!byParent.has(key)) byParent.set(key, [])
    byParent.get(key)!.push(g)
  }
  const build = (id: string): GoalNode[] =>
    (byParent.get(id) ?? []).map((goal) => ({ goal, children: build(goal.id) }))
  return build('__root')
}

function GoalCard({ node, workspaceId, depth }: { node: GoalNode; workspaceId: string; depth: number }) {
  const [open, setOpen] = useState(true)
  const { goal } = node

  const { data: targets = [] } = useQuery({
    queryKey: ['goal-targets', goal.id],
    queryFn: () => api.get(`goals/${goal.id}/targets`).json<GoalTarget[]>(),
    enabled: open,
  })

  const del = useMutation({
    mutationFn: () => api.delete(`goals/${goal.id}`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['goals', workspaceId] }),
  })

  return (
    <div className="bg-surface border border-ink-5/30 rounded-2xl shadow-card overflow-hidden" style={{ marginLeft: depth * 24 }}>
      <div className="flex items-center gap-3 px-4 py-3 group">
        <button onClick={() => setOpen((v) => !v)} className="text-ink-4 hover:text-ink-1">
          {open ? <ChevronDown className="w-4 h-4" /> : <ChevronRight className="w-4 h-4" />}
        </button>
        <div className="flex-1 min-w-0">
          <p className="text-sm font-semibold text-ink-1 truncate">{goal.name}</p>
          {goal.description && <p className="text-[11px] text-ink-4 truncate">{goal.description}</p>}
        </div>
        <div className="flex items-center gap-2 min-w-[160px]">
          <div className="flex-1 h-1.5 rounded-full bg-ink-5/30 overflow-hidden">
            <div className="h-full bg-brand-500 rounded-full transition-all" style={{ width: `${Math.round(goal.progress * 100)}%` }} />
          </div>
          <span className="text-[11px] font-semibold text-ink-2 tabular-nums w-9 text-right">{Math.round(goal.progress * 100)}%</span>
        </div>
        <button
          onClick={() => {
            if (confirm(`Delete goal "${goal.name}"?`)) del.mutate()
          }}
          className="opacity-0 group-hover:opacity-100 p-1 rounded text-ink-4 hover:text-red-500 hover:bg-red-50 transition-all"
        >
          <Trash2 className="w-3.5 h-3.5" />
        </button>
      </div>
      {open && (
        <div className="border-t border-ink-5/20 bg-canvas/30 px-4 py-3">
          <TargetsBlock goalId={goal.id} workspaceId={workspaceId} targets={targets} />
          {node.children.length > 0 && (
            <div className="mt-3 space-y-2">
              {node.children.map((c) => (
                <GoalCard key={c.goal.id} node={c} workspaceId={workspaceId} depth={depth + 1} />
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  )
}

function TargetsBlock({
  goalId, workspaceId, targets,
}: {
  goalId: string
  workspaceId: string
  targets: GoalTarget[]
}) {
  const [showAdd, setShowAdd] = useState(false)

  const update = useMutation({
    mutationFn: ({ id, ...patch }: { id: string } & Partial<GoalTarget>) =>
      api.patch(`targets/${id}`, { json: patch }).json<GoalTarget>(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['goal-targets', goalId] })
      queryClient.invalidateQueries({ queryKey: ['goals', workspaceId] })
    },
  })

  const del = useMutation({
    mutationFn: (id: string) => api.delete(`targets/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['goal-targets', goalId] })
      queryClient.invalidateQueries({ queryKey: ['goals', workspaceId] })
    },
  })

  return (
    <div className="space-y-2">
      {targets.length === 0 && <p className="text-[11px] text-ink-4">No targets yet.</p>}
      {targets.map((t) => (
        <TargetRow
          key={t.id}
          target={t}
          onUpdate={(patch) => update.mutate({ id: t.id, ...patch })}
          onDelete={() => {
            if (confirm(`Remove target "${t.name}"?`)) del.mutate(t.id)
          }}
        />
      ))}
      <button
        onClick={() => setShowAdd(true)}
        className="text-[11px] text-brand-600 hover:text-brand-700 font-medium"
      >
        + Add target
      </button>

      {showAdd && (
        <AddTargetModal goalId={goalId} workspaceId={workspaceId} onClose={() => setShowAdd(false)} />
      )}
    </div>
  )
}

function TargetRow({
  target, onUpdate, onDelete,
}: {
  target: GoalTarget
  onUpdate: (patch: Partial<GoalTarget>) => void
  onDelete: () => void
}) {
  const label = useMemo(() => {
    switch (target.kind) {
      case 'number':         return `${target.current_number} / ${target.target_number ?? '—'}`
      case 'currency':       return `${target.currency ?? 'USD'} ${target.current_number.toLocaleString()} / ${target.target_number?.toLocaleString() ?? '—'}`
      case 'boolean':        return target.current_boolean ? 'Done' : 'Pending'
      case 'task_completed': return `${target.completed_tasks ?? 0} / ${target.total_tasks ?? 0} tasks`
      default:               return ''
    }
  }, [target])

  return (
    <div className="flex items-center gap-3 group">
      {target.kind === 'boolean' ? (
        <button
          onClick={() => onUpdate({ current_boolean: !target.current_boolean })}
          className={cn(
            'w-5 h-5 rounded border flex items-center justify-center transition-colors shrink-0',
            target.current_boolean ? 'bg-brand-500 border-brand-500' : 'border-ink-5/60 hover:border-brand-400 bg-surface',
          )}
        >
          {target.current_boolean && <Check className="w-3 h-3 text-white" />}
        </button>
      ) : (
        <Target className="w-3.5 h-3.5 text-ink-4 shrink-0" />
      )}
      <span className="flex-1 text-xs text-ink-1 min-w-0 truncate">{target.name}</span>
      <span className="text-[11px] text-ink-4 tabular-nums">{label}</span>
      <div className="flex items-center gap-2 w-36">
        <div className="flex-1 h-1 rounded-full bg-ink-5/30 overflow-hidden">
          <div className="h-full bg-brand-400 rounded-full transition-all" style={{ width: `${Math.round(target.progress * 100)}%` }} />
        </div>
        <span className="text-[10px] text-ink-3 tabular-nums w-8 text-right">{Math.round(target.progress * 100)}%</span>
      </div>
      {(target.kind === 'number' || target.kind === 'currency') && (
        <input
          type="number"
          step="any"
          className="h-6 w-20 text-[11px] px-1.5 bg-surface border border-ink-5/30 rounded focus:outline-none focus:border-brand-400 opacity-0 group-hover:opacity-100 transition-opacity"
          defaultValue={target.current_number}
          onBlur={(e) => {
            const n = Number(e.target.value)
            if (!isNaN(n) && n !== target.current_number) onUpdate({ current_number: n })
          }}
        />
      )}
      <button onClick={onDelete} className="opacity-0 group-hover:opacity-100 text-ink-4 hover:text-red-500">
        <Trash2 className="w-3 h-3" />
      </button>
    </div>
  )
}

function AddTargetModal({
  goalId, workspaceId, onClose,
}: {
  goalId: string
  workspaceId: string
  onClose: () => void
}) {
  const [name, setName] = useState('')
  const [kind, setKind] = useState<TargetKind>('number')
  const [targetNumber, setTargetNumber] = useState('')
  const [currency, setCurrency] = useState('USD')
  const [taskListId, setTaskListId] = useState('')

  const { data: spaces = [] } = useQuery({
    queryKey: ['spaces', workspaceId],
    queryFn: () => api.get(`workspaces/${workspaceId}/spaces`).json<Space[]>(),
  })
  const listsBySpace = useQueries_lists(spaces)

  const create = useMutation({
    mutationFn: () => {
      const body: Record<string, unknown> = { name, kind }
      if (kind === 'number' || kind === 'currency') body.target_number = Number(targetNumber)
      if (kind === 'currency') body.currency = currency
      if (kind === 'task_completed') body.task_list_id = taskListId
      return api.post(`goals/${goalId}/targets`, { json: body }).json<GoalTarget>()
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['goal-targets', goalId] })
      queryClient.invalidateQueries({ queryKey: ['goals', workspaceId] })
      onClose()
    },
  })

  const canSubmit =
    name.trim() &&
    ((kind === 'boolean') ||
      (kind === 'number' && targetNumber) ||
      (kind === 'currency' && targetNumber && currency) ||
      (kind === 'task_completed' && taskListId))

  return (
    <Modal open title="New target" description="Pick a target kind and wire it to a measurement." onClose={onClose}>
      <div className="space-y-3">
        <Input label="Name" placeholder="Ship docs v1" value={name} onChange={(e) => setName(e.target.value)} autoFocus />
        <div>
          <label className="block text-xs font-medium text-ink-2 mb-1">Kind</label>
          <div className="grid grid-cols-2 gap-2">
            {TARGET_KINDS.map((k) => (
              <button
                key={k.value}
                onClick={() => setKind(k.value)}
                className={cn(
                  'text-left px-3 py-2 rounded-lg border transition-all',
                  kind === k.value ? 'border-brand-400 bg-brand-50' : 'border-ink-5/30 bg-surface hover:border-ink-5/60',
                )}
              >
                <p className="text-xs font-semibold text-ink-1">{k.label}</p>
                <p className="text-[10px] text-ink-4 mt-0.5">{k.hint}</p>
              </button>
            ))}
          </div>
        </div>

        {(kind === 'number' || kind === 'currency') && (
          <div className="flex items-end gap-2">
            <Input
              label={kind === 'currency' ? 'Target amount' : 'Target number'}
              type="number"
              value={targetNumber}
              onChange={(e) => setTargetNumber(e.target.value)}
            />
            {kind === 'currency' && (
              <div className="w-24">
                <label className="block text-xs font-medium text-ink-2 mb-1">Currency</label>
                <input
                  className="h-9 w-full bg-surface border border-ink-5/40 rounded-lg px-3 text-sm text-ink-1 focus:outline-none focus:ring-2 focus:ring-brand-400/40 focus:border-brand-400"
                  value={currency}
                  onChange={(e) => setCurrency(e.target.value.toUpperCase())}
                  maxLength={5}
                />
              </div>
            )}
          </div>
        )}

        {kind === 'task_completed' && (
          <div>
            <label className="block text-xs font-medium text-ink-2 mb-1">Task list</label>
            <select
              className="h-9 w-full bg-surface border border-ink-5/40 rounded-lg px-3 text-sm text-ink-1 focus:outline-none focus:ring-2 focus:ring-brand-400/40"
              value={taskListId}
              onChange={(e) => setTaskListId(e.target.value)}
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

        <div className="flex justify-end gap-2 pt-1">
          <Button variant="secondary" onClick={onClose}>Cancel</Button>
          <Button onClick={() => create.mutate()} disabled={!canSubmit || create.isPending} loading={create.isPending}>
            Add target
          </Button>
        </div>
      </div>
    </Modal>
  )
}

// useQueries_lists loads all lists grouped by space without hitting the useQuery
// rule-of-hooks limit in a map.
function useQueries_lists(spaces: Space[]): Record<string, List[]> {
  const spaceIds = spaces.map((s) => s.id).join(',')
  const { data } = useQuery({
    queryKey: ['goal-target-list-picker', spaceIds],
    queryFn: async () => {
      const entries: Record<string, List[]> = {}
      await Promise.all(
        spaces.map(async (s) => {
          entries[s.id] = await api.get(`spaces/${s.id}/lists`).json<List[]>()
        }),
      )
      return entries
    },
    enabled: spaces.length > 0,
  })
  return data ?? {}
}

function CreateGoalModal({
  workspaceId, goals, onClose,
}: {
  workspaceId: string
  goals: Goal[]
  onClose: () => void
}) {
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [parentGoalId, setParentGoalId] = useState('')
  const [dueAt, setDueAt] = useState('')

  const create = useMutation({
    mutationFn: () =>
      api.post(`workspaces/${workspaceId}/goals`, {
        json: {
          name,
          description,
          parent_goal_id: parentGoalId || undefined,
          due_at: dueAt ? new Date(dueAt).toISOString() : undefined,
        },
      }).json<Goal>(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['goals', workspaceId] })
      onClose()
    },
  })

  return (
    <Modal open title="New goal" description="A goal groups measurable targets." onClose={onClose}>
      <div className="space-y-3">
        <Input label="Name" placeholder="Q3 launch" value={name} onChange={(e) => setName(e.target.value)} autoFocus />
        <div>
          <label className="block text-xs font-medium text-ink-2 mb-1">Description</label>
          <textarea
            className="w-full bg-surface border border-ink-5/40 rounded-xl px-3 py-2.5 text-sm text-ink-1 h-20 resize-none focus:outline-none focus:ring-2 focus:ring-brand-400/40"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
          />
        </div>
        <div className="grid grid-cols-2 gap-3">
          <div>
            <label className="block text-xs font-medium text-ink-2 mb-1">Parent goal (optional)</label>
            <select
              className="h-9 w-full bg-surface border border-ink-5/40 rounded-lg px-3 text-sm text-ink-1 focus:outline-none focus:ring-2 focus:ring-brand-400/40"
              value={parentGoalId}
              onChange={(e) => setParentGoalId(e.target.value)}
            >
              <option value="">None</option>
              {goals.map((g) => <option key={g.id} value={g.id}>{g.name}</option>)}
            </select>
          </div>
          <div>
            <label className="block text-xs font-medium text-ink-2 mb-1">Due</label>
            <input
              type="date"
              className="h-9 w-full bg-surface border border-ink-5/40 rounded-lg px-3 text-sm focus:outline-none focus:ring-2 focus:ring-brand-400/40"
              value={dueAt}
              onChange={(e) => setDueAt(e.target.value)}
            />
          </div>
        </div>
        <div className="flex justify-end gap-2 pt-1">
          <Button variant="secondary" onClick={onClose}>Cancel</Button>
          <Button onClick={() => create.mutate()} disabled={!name.trim() || create.isPending} loading={create.isPending}>
            Create
          </Button>
        </div>
      </div>
    </Modal>
  )
}
