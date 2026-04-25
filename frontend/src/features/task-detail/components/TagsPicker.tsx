import { useState } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import * as Popover from '@radix-ui/react-popover'

import { api } from '../../../lib/api'
import { queryClient } from '../../../lib/queryClient'
import { Input } from '../../../components/ui'
import type { Tag } from '../../../types'

const DEFAULT_COLORS = ['#6375E8', '#F97316', '#EAB308', '#22C55E', '#EF4444', '#8B5CF6', '#06B6D4', '#64748B']

export function TagsPicker({ taskId, workspaceId }: { taskId: string; workspaceId: string }) {
  const [open, setOpen] = useState(false)
  const [newName, setNewName] = useState('')
  const [newColor, setNewColor] = useState(DEFAULT_COLORS[0])

  const { data: taskTags } = useQuery({
    queryKey: ['task-tags', taskId],
    queryFn: () => api.get(`tasks/${taskId}/tags`).json<Tag[]>(),
  })

  const { data: wsTags } = useQuery({
    queryKey: ['workspace-tags', workspaceId],
    queryFn: () => api.get(`workspaces/${workspaceId}/tags`).json<Tag[]>(),
  })

  const attach = useMutation({
    mutationFn: (tagID: string) =>
      api.post(`tasks/${taskId}/tags`, { json: { tag_id: tagID } }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['task-tags', taskId] }),
  })

  const detach = useMutation({
    mutationFn: (tagID: string) => api.delete(`tasks/${taskId}/tags/${tagID}`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['task-tags', taskId] }),
  })

  const createTag = useMutation({
    mutationFn: () =>
      api.post(`workspaces/${workspaceId}/tags`, { json: { name: newName, color: newColor } }).json<Tag>(),
    onSuccess: (t) => {
      queryClient.invalidateQueries({ queryKey: ['workspace-tags', workspaceId] })
      setNewName('')
      attach.mutate(t.id)
    },
  })

  const taskTagIds = new Set((taskTags ?? []).map((t) => t.id))

  return (
    <div className="flex items-center flex-wrap gap-1.5">
      {(taskTags ?? []).map((t) => (
        <span
          key={t.id}
          className="inline-flex items-center gap-1 text-[11px] font-medium px-2 py-0.5 rounded-full group"
          style={{
            backgroundColor: (t.color ?? '#94a3b8') + '22',
            color: t.color ?? '#64748b',
          }}
        >
          {t.name}
          <button
            onClick={() => detach.mutate(t.id)}
            className="opacity-0 group-hover:opacity-100 transition-opacity"
            aria-label="Remove tag"
          >
            ×
          </button>
        </span>
      ))}
      <Popover.Root open={open} onOpenChange={setOpen}>
        <Popover.Trigger asChild>
          <button className="text-[11px] text-ink-4 hover:text-ink-1 hover:bg-ink-1/5 px-2 py-0.5 rounded-full border border-dashed border-ink-5/50 transition-colors">
            + Tag
          </button>
        </Popover.Trigger>
        <Popover.Portal>
          <Popover.Content
            sideOffset={6}
            className="bg-surface border border-ink-5/30 rounded-xl shadow-modal w-72 p-2 max-h-80 overflow-y-auto z-50"
          >
            <p className="text-[10px] font-bold uppercase tracking-wider text-ink-4 px-2 py-1.5">Tags</p>
            <div className="space-y-0.5">
              {(wsTags ?? []).map((t) => {
                const on = taskTagIds.has(t.id)
                return (
                  <button
                    key={t.id}
                    onClick={() => (on ? detach.mutate(t.id) : attach.mutate(t.id))}
                    className="flex items-center gap-2 w-full px-2 py-1.5 rounded hover:bg-ink-1/5 text-left"
                  >
                    <span
                      className="w-2.5 h-2.5 rounded-full shrink-0"
                      style={{ backgroundColor: t.color ?? '#94a3b8' }}
                    />
                    <span className="flex-1 text-xs text-ink-1">{t.name}</span>
                    {on && (
                      <svg className="w-3.5 h-3.5 text-brand-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2.5} d="M5 13l4 4L19 7" />
                      </svg>
                    )}
                  </button>
                )
              })}
            </div>

            <div className="pt-2 mt-2 border-t border-ink-5/20">
              <p className="text-[10px] font-bold uppercase tracking-wider text-ink-4 px-2 py-1.5">New tag</p>
              <div className="px-2 flex items-center gap-2">
                <Input
                  placeholder="Name"
                  value={newName}
                  onChange={(e) => setNewName(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter' && newName.trim()) createTag.mutate()
                  }}
                />
              </div>
              <div className="px-2 flex items-center gap-1.5 mt-2">
                {DEFAULT_COLORS.map((c) => (
                  <button
                    key={c}
                    onClick={() => setNewColor(c)}
                    className="w-5 h-5 rounded-full transition-all"
                    style={{
                      backgroundColor: c,
                      boxShadow: newColor === c ? `0 0 0 2px ${c}` : 'none',
                    }}
                    aria-label={`Color ${c}`}
                  />
                ))}
              </div>
            </div>
          </Popover.Content>
        </Popover.Portal>
      </Popover.Root>
    </div>
  )
}
