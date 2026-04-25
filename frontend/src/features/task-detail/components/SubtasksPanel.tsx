import { useState } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { api } from '../../../lib/api'
import { queryClient } from '../../../lib/queryClient'
import { cn } from '../../../lib/utils'
import type { Task } from '../../../types'

export function SubtasksPanel({ parentTask, workspaceId }: { parentTask: Task; workspaceId: string }) {
  const navigate = useNavigate()
  const [name, setName] = useState('')

  const { data: subtasks } = useQuery({
    queryKey: ['subtasks', parentTask.id],
    queryFn: () => api.get(`tasks/${parentTask.id}/subtasks`).json<Task[]>(),
  })

  const create = useMutation({
    mutationFn: () =>
      api
        .post(`tasks/${parentTask.id}/subtasks`, {
          json: { list_id: parentTask.list_id, name, description: '' },
        })
        .json<Task>(),
    onSuccess: () => {
      setName('')
      queryClient.invalidateQueries({ queryKey: ['subtasks', parentTask.id] })
    },
  })

  const toggle = useMutation({
    mutationFn: ({ id, status }: { id: string; status: string }) =>
      api.patch(`tasks/${id}`, { json: { status } }).json<Task>(),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['subtasks', parentTask.id] }),
  })

  return (
    <div>
      <ul className="space-y-1 mb-3">
        {(subtasks ?? []).map((s) => {
          const done = s.status === 'completed'
          return (
            <li
              key={s.id}
              className="flex items-center gap-2 px-2 py-1 rounded hover:bg-ink-1/5 group"
            >
              <button
                onClick={() => toggle.mutate({ id: s.id, status: done ? 'open' : 'completed' })}
                className={cn(
                  'w-4 h-4 rounded border shrink-0 flex items-center justify-center transition-colors',
                  done ? 'bg-brand-500 border-brand-500' : 'border-ink-5/60 hover:border-brand-400',
                )}
              >
                {done && (
                  <svg className="w-2.5 h-2.5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={3} d="M5 13l4 4L19 7" />
                  </svg>
                )}
              </button>
              <button
                onClick={() => navigate(`/workspaces/${workspaceId}/tasks/${s.id}`)}
                className={cn(
                  'flex-1 text-left text-sm truncate transition-colors',
                  done ? 'text-ink-4 line-through' : 'text-ink-1 hover:text-brand-600',
                )}
              >
                {s.name}
              </button>
            </li>
          )
        })}
        {subtasks?.length === 0 && (
          <p className="text-xs text-ink-4 py-1">No subtasks yet.</p>
        )}
      </ul>
      <div className="flex items-center gap-2">
        <input
          className="flex-1 h-8 bg-canvas/50 border border-ink-5/30 rounded-lg px-3 text-xs text-ink-1 focus:outline-none focus:ring-2 focus:ring-brand-400/40 focus:border-brand-400 placeholder:text-ink-5"
          placeholder="Add a subtask…"
          value={name}
          onChange={(e) => setName(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && name.trim()) create.mutate()
          }}
        />
      </div>
    </div>
  )
}
