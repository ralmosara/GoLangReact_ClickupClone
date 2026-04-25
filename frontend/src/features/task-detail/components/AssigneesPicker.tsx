import { useState } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import * as Popover from '@radix-ui/react-popover'

import { api } from '../../../lib/api'
import { queryClient } from '../../../lib/queryClient'
import type { Member } from '../../../types'

export function AssigneesPicker({ taskId, workspaceId }: { taskId: string; workspaceId: string }) {
  const [open, setOpen] = useState(false)

  const { data: assignees } = useQuery({
    queryKey: ['task-assignees', taskId],
    queryFn: () => api.get(`tasks/${taskId}/assignees`).json<string[]>(),
  })

  const { data: members } = useQuery({
    queryKey: ['workspace-members', workspaceId],
    queryFn: () => api.get(`workspaces/${workspaceId}/members`).json<Member[]>(),
  })

  const add = useMutation({
    mutationFn: (userID: string) =>
      api.post(`tasks/${taskId}/assignees`, { json: { user_id: userID } }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['task-assignees', taskId] }),
  })

  const remove = useMutation({
    mutationFn: (userID: string) => api.delete(`tasks/${taskId}/assignees/${userID}`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['task-assignees', taskId] }),
  })

  const assignedSet = new Set(assignees ?? [])
  const assignedMembers = (members ?? []).filter((m) => assignedSet.has(m.user_id))

  return (
    <Popover.Root open={open} onOpenChange={setOpen}>
      <Popover.Trigger asChild>
        <button className="flex items-center gap-1 hover:bg-ink-1/5 rounded-lg px-2 py-1 transition-colors">
          {assignedMembers.length === 0 ? (
            <span className="text-xs text-ink-4">Unassigned</span>
          ) : (
            <div className="flex -space-x-1.5">
              {assignedMembers.slice(0, 4).map((m) => (
                <div
                  key={m.user_id}
                  className="w-6 h-6 rounded-full bg-brand-gradient border-2 border-surface text-[10px] font-bold text-white flex items-center justify-center"
                  title={m.name || m.email}
                >
                  {(m.name || m.email || '?').slice(0, 1).toUpperCase()}
                </div>
              ))}
              {assignedMembers.length > 4 && (
                <div className="w-6 h-6 rounded-full bg-ink-5/60 border-2 border-surface text-[10px] font-bold text-ink-2 flex items-center justify-center">
                  +{assignedMembers.length - 4}
                </div>
              )}
            </div>
          )}
        </button>
      </Popover.Trigger>
      <Popover.Portal>
        <Popover.Content
          sideOffset={6}
          className="bg-surface border border-ink-5/30 rounded-xl shadow-modal w-64 p-2 max-h-80 overflow-y-auto z-50"
        >
          <p className="text-[10px] font-bold uppercase tracking-wider text-ink-4 px-2 py-1.5">Assignees</p>
          {(members ?? []).length === 0 && (
            <p className="text-xs text-ink-4 px-2 py-2">No members yet. Invite teammates from the workspace settings.</p>
          )}
          {(members ?? []).map((m) => {
            const isAssigned = assignedSet.has(m.user_id)
            return (
              <button
                key={m.user_id}
                onClick={() => {
                  if (isAssigned) remove.mutate(m.user_id)
                  else add.mutate(m.user_id)
                }}
                className="flex items-center gap-2 w-full px-2 py-1.5 rounded text-left text-sm hover:bg-ink-1/5 transition-colors"
              >
                <div className="w-6 h-6 rounded-full bg-brand-gradient text-[10px] font-bold text-white flex items-center justify-center shrink-0">
                  {(m.name || m.email || '?').slice(0, 1).toUpperCase()}
                </div>
                <div className="flex-1 min-w-0">
                  <p className="text-xs font-medium text-ink-1 truncate">{m.name || m.email}</p>
                  {m.email && m.name && <p className="text-[10px] text-ink-4 truncate">{m.email}</p>}
                </div>
                {isAssigned && (
                  <svg className="w-4 h-4 text-brand-500 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2.5} d="M5 13l4 4L19 7" />
                  </svg>
                )}
              </button>
            )
          })}
        </Popover.Content>
      </Popover.Portal>
    </Popover.Root>
  )
}
