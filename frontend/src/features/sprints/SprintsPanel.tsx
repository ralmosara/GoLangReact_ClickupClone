import { useMemo, useState } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import { Plus, Flag, CheckCircle2 } from 'lucide-react'

import { api } from '../../lib/api'
import { queryClient } from '../../lib/queryClient'
import { Button, Input, Modal } from '../../components/ui'
import { cn, formatRelative } from '../../lib/utils'
import type { CloseSprintResult, Sprint } from '../../types'

export function SprintsPanel({ listId }: { listId: string }) {
  const [showCreate, setShowCreate] = useState(false)

  const { data: sprints = [] } = useQuery({
    queryKey: ['sprints', listId],
    queryFn: () => api.get(`lists/${listId}/sprints`).json<Sprint[]>(),
  })

  const active = useMemo(
    () => sprints.filter((s) => s.status !== 'closed').sort((a, b) => a.starts_at.localeCompare(b.starts_at)),
    [sprints],
  )
  const closed = useMemo(
    () => sprints.filter((s) => s.status === 'closed').sort((a, b) => b.ends_at.localeCompare(a.ends_at)),
    [sprints],
  )

  return (
    <div className="bg-surface border border-ink-5/30 rounded-2xl shadow-card p-5">
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <Flag className="w-4 h-4 text-brand-500" />
          <h3 className="text-sm font-semibold text-ink-1">Sprints</h3>
          <span className="text-[11px] text-ink-4">({sprints.length})</span>
        </div>
        <Button size="sm" variant="secondary" onClick={() => setShowCreate(true)}>
          <Plus className="w-3 h-3 mr-1" />
          New sprint
        </Button>
      </div>

      {sprints.length === 0 && (
        <p className="text-xs text-ink-4">No sprints yet. Create one to start tracking velocity + burndown.</p>
      )}

      {active.length > 0 && (
        <div className="space-y-1.5">
          {active.map((s) => <SprintRow key={s.id} sprint={s} listId={listId} />)}
        </div>
      )}
      {closed.length > 0 && (
        <div className="mt-3 pt-3 border-t border-ink-5/20">
          <p className="text-[10px] font-bold uppercase tracking-wider text-ink-4 mb-1.5">Closed</p>
          <div className="space-y-1.5">
            {closed.slice(0, 5).map((s) => <SprintRow key={s.id} sprint={s} listId={listId} />)}
          </div>
        </div>
      )}

      {showCreate && <CreateSprintModal listId={listId} onClose={() => setShowCreate(false)} />}
    </div>
  )
}

function SprintRow({ sprint, listId }: { sprint: Sprint; listId: string }) {
  const close = useMutation({
    mutationFn: () => api.post(`sprints/${sprint.id}/close`).json<CloseSprintResult>(),
    onSuccess: (r) => {
      queryClient.invalidateQueries({ queryKey: ['sprints', listId] })
      queryClient.invalidateQueries({ queryKey: ['tasks', listId] })
      if (r.rolled_over > 0) {
        alert(`${r.rolled_over} open task${r.rolled_over === 1 ? '' : 's'} rolled over${r.next_sprint_id ? ' to the next sprint' : ' (no follow-on sprint — detached)'}`)
      }
    },
  })

  return (
    <div className="flex items-center gap-3 px-3 py-2 bg-canvas/60 border border-ink-5/20 rounded-xl group">
      <span
        className={cn(
          'text-[10px] font-bold uppercase tracking-wider px-2 py-0.5 rounded-full shrink-0',
          sprint.status === 'active'  ? 'bg-brand-50 text-brand-700' :
          sprint.status === 'planned' ? 'bg-ink-5/40 text-ink-2' :
                                        'bg-green-50 text-green-700',
        )}
      >
        {sprint.status}
      </span>
      <div className="flex-1 min-w-0">
        <p className="text-xs font-medium text-ink-1 truncate">{sprint.name}</p>
        <p className="text-[10px] text-ink-4">
          {new Date(sprint.starts_at).toLocaleDateString()} → {new Date(sprint.ends_at).toLocaleDateString()}
          {sprint.goal_points > 0 && ` · ${sprint.goal_points} pts`}
        </p>
      </div>
      {sprint.status !== 'closed' && (
        <button
          onClick={() => {
            if (confirm(`Close "${sprint.name}"? Incomplete tasks roll over to the next open sprint.`)) close.mutate()
          }}
          className="text-[11px] text-ink-4 hover:text-brand-600 opacity-0 group-hover:opacity-100 transition-opacity"
        >
          <CheckCircle2 className="w-3.5 h-3.5 inline mr-0.5" />
          Close
        </button>
      )}
      <span className="text-[10px] text-ink-5 shrink-0">{formatRelative(sprint.created_at)}</span>
    </div>
  )
}

function CreateSprintModal({ listId, onClose }: { listId: string; onClose: () => void }) {
  // Default: today → 2 weeks out.
  const defaults = useMemo(() => {
    const start = new Date()
    const end = new Date()
    end.setDate(end.getDate() + 14)
    return { start: start.toISOString().slice(0, 10), end: end.toISOString().slice(0, 10) }
  }, [])
  const [name, setName] = useState('Sprint 1')
  const [start, setStart] = useState(defaults.start)
  const [end, setEnd] = useState(defaults.end)
  const [points, setPoints] = useState('')

  const create = useMutation({
    mutationFn: () =>
      api.post(`lists/${listId}/sprints`, {
        json: {
          name,
          starts_at: new Date(start).toISOString(),
          ends_at: new Date(end + 'T23:59:59').toISOString(),
          goal_points: points ? Number(points) : 0,
          status: 'active',
        },
      }).json<Sprint>(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['sprints', listId] })
      onClose()
    },
  })

  return (
    <Modal open title="New sprint" description="Define a time-boxed iteration." onClose={onClose}>
      <div className="space-y-3">
        <Input label="Name" value={name} onChange={(e) => setName(e.target.value)} autoFocus />
        <div className="grid grid-cols-2 gap-3">
          <div>
            <label className="block text-xs font-medium text-ink-2 mb-1">Start</label>
            <input
              type="date" value={start} onChange={(e) => setStart(e.target.value)}
              className="h-9 w-full bg-surface border border-ink-5/40 rounded-lg px-3 text-sm focus:outline-none focus:ring-2 focus:ring-brand-400/40"
            />
          </div>
          <div>
            <label className="block text-xs font-medium text-ink-2 mb-1">End</label>
            <input
              type="date" value={end} onChange={(e) => setEnd(e.target.value)}
              className="h-9 w-full bg-surface border border-ink-5/40 rounded-lg px-3 text-sm focus:outline-none focus:ring-2 focus:ring-brand-400/40"
            />
          </div>
        </div>
        <Input
          label="Goal points (optional)"
          type="number"
          value={points}
          onChange={(e) => setPoints(e.target.value)}
        />
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
