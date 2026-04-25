import { useState } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import { api } from '../../../lib/api'
import { queryClient } from '../../../lib/queryClient'
import { Button, Input, Modal } from '../../../components/ui'
import type { Status } from '../../../types'

const DEFAULT_COLORS = ['#6375E8', '#F97316', '#EAB308', '#22C55E', '#EF4444', '#8B5CF6', '#06B6D4']

const CATEGORIES: Array<Status['category']> = ['active', 'done', 'closed']

export function StatusManagerDialog({ listId, onClose }: { listId: string; onClose: () => void }) {
  const { data: statuses } = useQuery({
    queryKey: ['statuses', listId],
    queryFn: () => api.get(`lists/${listId}/statuses`).json<Status[]>(),
  })

  const create = useMutation({
    mutationFn: (body: Partial<Status>) =>
      api.post(`lists/${listId}/statuses`, { json: body }).json<Status>(),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['statuses', listId] }),
  })

  const update = useMutation({
    mutationFn: ({ id, ...patch }: { id: string } & Partial<Status>) =>
      api.patch(`statuses/${id}`, { json: patch }).json<Status>(),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['statuses', listId] }),
  })

  const del = useMutation({
    mutationFn: (id: string) => api.delete(`statuses/${id}`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['statuses', listId] }),
  })

  const [newName, setNewName] = useState('')
  const [newColor, setNewColor] = useState(DEFAULT_COLORS[0])
  const [newCategory, setNewCategory] = useState<Status['category']>('active')

  const handleAdd = () => {
    if (!newName.trim()) return
    create.mutate(
      {
        name: newName.trim(),
        color: newColor,
        category: newCategory,
        order_index: (statuses?.length ?? 0) * 10,
      },
      { onSuccess: () => setNewName('') },
    )
  }

  return (
    <Modal open title="List statuses" description="Define the columns shown on the board." onClose={onClose} width="max-w-lg">
      <div className="space-y-3">
        {(statuses ?? []).map((s) => (
          <StatusRow
            key={s.id}
            status={s}
            onUpdate={(patch) => update.mutate({ id: s.id, ...patch })}
            onDelete={() => {
              if (confirm(`Delete status "${s.name}"? Tasks will keep pointing at it until reassigned.`)) {
                del.mutate(s.id)
              }
            }}
          />
        ))}

        <div className="pt-3 border-t border-ink-5/30 space-y-2">
          <p className="text-xs font-semibold text-ink-2">Add status</p>
          <div className="flex items-center gap-2">
            <Input
              placeholder="Status name"
              value={newName}
              onChange={(e) => setNewName(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && handleAdd()}
            />
            <div className="flex items-center gap-1.5">
              {DEFAULT_COLORS.map((c) => (
                <button
                  key={c}
                  type="button"
                  onClick={() => setNewColor(c)}
                  className="w-5 h-5 rounded-full transition-all"
                  style={{
                    backgroundColor: c,
                    boxShadow: newColor === c ? `0 0 0 2px ${c}, 0 0 0 4px #fff` : 'none',
                  }}
                  aria-label={`Pick color ${c}`}
                />
              ))}
            </div>
            <select
              className="h-9 px-2 rounded border border-ink-5/40 text-xs"
              value={newCategory}
              onChange={(e) => setNewCategory(e.target.value as Status['category'])}
            >
              {CATEGORIES.map((c) => <option key={c} value={c}>{c}</option>)}
            </select>
            <Button onClick={handleAdd} disabled={!newName.trim() || create.isPending} size="sm">
              Add
            </Button>
          </div>
        </div>

        <div className="flex justify-end pt-2">
          <Button variant="secondary" onClick={onClose}>Done</Button>
        </div>
      </div>
    </Modal>
  )
}

function StatusRow({
  status,
  onUpdate,
  onDelete,
}: {
  status: Status
  onUpdate: (patch: Partial<Status>) => void
  onDelete: () => void
}) {
  const [name, setName] = useState(status.name)
  return (
    <div className="flex items-center gap-2 bg-canvas/60 border border-ink-5/20 rounded-xl px-3 py-2">
      <input
        type="color"
        className="h-6 w-8 rounded cursor-pointer bg-transparent border-0"
        value={status.color}
        onChange={(e) => onUpdate({ color: e.target.value })}
      />
      <input
        className="flex-1 text-sm bg-transparent focus:outline-none text-ink-1"
        value={name}
        onChange={(e) => setName(e.target.value)}
        onBlur={() => name !== status.name && onUpdate({ name })}
        onKeyDown={(e) => {
          if (e.key === 'Enter') {
            ;(e.target as HTMLInputElement).blur()
          }
        }}
      />
      <select
        className="h-7 text-[11px] rounded border border-ink-5/40 px-1.5"
        value={status.category}
        onChange={(e) => onUpdate({ category: e.target.value as Status['category'] })}
      >
        {CATEGORIES.map((c) => <option key={c} value={c}>{c}</option>)}
      </select>
      <button
        onClick={onDelete}
        className="w-7 h-7 rounded text-ink-4 hover:text-red-500 hover:bg-red-50 flex items-center justify-center transition-colors"
        title="Delete status"
      >
        <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6M1 7h22M9 7V5a2 2 0 012-2h2a2 2 0 012 2v2" />
        </svg>
      </button>
    </div>
  )
}
