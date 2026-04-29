import { useState } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import { useNavigate, useParams } from 'react-router-dom'
import { api } from '../../lib/api'
import { queryClient } from '../../lib/queryClient'
import { rooms } from '../../lib/ws'
import { useWsEvent, useWsRooms } from '../../hooks/useWebSocket'
import { cn, formatRelative, STATUS_COLORS, STATUS_DOT, STATUS_LABEL, PRIORITY_COLORS, PRIORITY_LABELS } from '../../lib/utils'
import { Button, PageSpinner } from '../../components/ui'
import type { Comment, Status, Task } from '../../types'

import { AssigneesPicker } from './components/AssigneesPicker'
import { TagsPicker } from './components/TagsPicker'
import { AttachmentsPanel } from './components/AttachmentsPanel'
import { SubtasksPanel } from './components/SubtasksPanel'
import { DescriptionEditor } from './components/DescriptionEditor'
import { CustomFieldsBlock } from '../custom-fields/components/CustomFieldsBlock'
import { CustomFieldsManager } from '../custom-fields/components/CustomFieldsManager'
import { TimeTracker } from '../time-tracking/components/TimeTracker'
import { TimeEntriesList } from '../time-tracking/components/TimeEntriesList'
import { DependenciesPanel } from '../dependencies/DependenciesPanel'
import { RecurrencePicker } from './components/RecurrencePicker'

const LEGACY_STATUSES = ['open', 'in_progress', 'review', 'completed', 'cancelled'] as const

export function TaskDetailPage() {
  const { taskId, workspaceId } = useParams<{ taskId: string; workspaceId: string }>()
  const navigate = useNavigate()
  const [commentBody, setCommentBody] = useState('')
  const [editingName, setEditingName] = useState(false)
  const [localName, setLocalName] = useState('')
  const [showFieldsManager, setShowFieldsManager] = useState(false)

  const { data: task, isLoading } = useQuery({
    queryKey: ['task', taskId],
    queryFn: () => api.get(`tasks/${taskId}`).json<Task>(),
    enabled: !!taskId,
  })

  const { data: statuses } = useQuery({
    queryKey: ['statuses', task?.list_id],
    queryFn: () => api.get(`lists/${task!.list_id}/statuses`).json<Status[]>(),
    enabled: !!task?.list_id,
  })

  const { data: comments } = useQuery({
    queryKey: ['comments', taskId],
    queryFn: () => api.get(`tasks/${taskId}/comments`).json<Comment[]>(),
    enabled: !!taskId,
  })

  // Join the task room so every sub-panel sees real-time updates.
  useWsRooms(taskId ? [rooms.task(taskId), task?.list_id ? rooms.list(task.list_id) : ''].filter(Boolean) : [])
  useWsEvent('task.updated', (e) => {
    if (e.entity_id === taskId) {
      queryClient.invalidateQueries({ queryKey: ['task', taskId] })
    }
  })
  useWsEvent('comment.created', (e) => {
    if (e.entity_id && comments?.some((c) => c.id === e.entity_id)) return
    queryClient.invalidateQueries({ queryKey: ['comments', taskId] })
  })
  useWsEvent('assignee.added', () => queryClient.invalidateQueries({ queryKey: ['task-assignees', taskId] }))
  useWsEvent('assignee.removed', () => queryClient.invalidateQueries({ queryKey: ['task-assignees', taskId] }))
  useWsEvent('attachment.created', () => queryClient.invalidateQueries({ queryKey: ['attachments', taskId] }))
  useWsEvent('attachment.deleted', () => queryClient.invalidateQueries({ queryKey: ['attachments', taskId] }))

  const updateTask = useMutation({
    mutationFn: (patch: Partial<Task>) => api.patch(`tasks/${taskId}`, { json: patch }).json<Task>(),
    onSuccess: (updated) => {
      queryClient.setQueryData(['task', taskId], updated)
      queryClient.invalidateQueries({ queryKey: ['tasks', updated.list_id] })
    },
  })

  const addComment = useMutation({
    mutationFn: () =>
      api.post('comments', { json: { task_id: taskId, body: commentBody } }).json<Comment>(),
    onSuccess: () => {
      setCommentBody('')
      queryClient.invalidateQueries({ queryKey: ['comments', taskId] })
    },
  })

  const deleteTask = useMutation({
    mutationFn: () => api.delete(`tasks/${taskId}`),
    onSuccess: () => navigate(`/workspaces/${workspaceId}`),
  })

  const archiveTask = useMutation({
    mutationFn: (archived: boolean) =>
      api.post(`tasks/${taskId}/${archived ? 'archive' : 'unarchive'}`).json<Task>(),
    onSuccess: (updated) => {
      queryClient.setQueryData(['task', taskId], updated)
      queryClient.invalidateQueries({ queryKey: ['tasks', updated.list_id] })
    },
  })

  if (isLoading) return <PageSpinner />

  if (!task) {
    return (
      <div className="flex items-center justify-center h-64 text-ink-4 text-sm">
        Task not found.
      </div>
    )
  }

  const hasCustomStatuses = (statuses?.length ?? 0) > 0
  const currentStatus = hasCustomStatuses
    ? statuses!.find((s) => s.id === task.status_id)
    : undefined

  return (
    <div className="max-w-3xl mx-auto px-6 py-8 animate-slide-up">
      <div className="flex items-center justify-between mb-6">
        <button
          onClick={() => navigate(-1)}
          className="flex items-center gap-1.5 text-xs text-ink-4 hover:text-ink-1 transition-colors group"
        >
          <svg className="w-3.5 h-3.5 group-hover:-translate-x-0.5 transition-transform" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
          </svg>
          Back
        </button>
        <TimeTracker taskId={task.id} />
      </div>

      <div className="bg-surface border border-ink-5/30 rounded-2xl shadow-card p-6 mb-4">
        {editingName ? (
          <input
            autoFocus
            className="w-full text-xl font-bold text-ink-1 border-b-2 border-brand-400 focus:outline-none mb-5 pb-1 bg-transparent"
            value={localName}
            onChange={(e) => setLocalName(e.target.value)}
            onBlur={() => {
              if (localName.trim() && localName !== task.name) updateTask.mutate({ name: localName })
              setEditingName(false)
            }}
            onKeyDown={(e) => {
              if (e.key === 'Enter') {
                if (localName.trim() && localName !== task.name) updateTask.mutate({ name: localName })
                setEditingName(false)
              }
              if (e.key === 'Escape') setEditingName(false)
            }}
          />
        ) : (
          <h1
            onClick={() => { setLocalName(task.name); setEditingName(true) }}
            className="text-xl font-bold text-ink-1 mb-5 cursor-text hover:text-brand-600 transition-colors"
            title="Click to edit"
          >
            {task.name}
          </h1>
        )}

        {/* Meta row */}
        <div className="flex flex-wrap gap-x-6 gap-y-4 mb-5">
          {/* Status */}
          <div>
            <p className="text-[10px] font-semibold text-ink-4 uppercase tracking-wider mb-1.5">Status</p>
            {hasCustomStatuses ? (
              <select
                className="text-xs font-semibold px-3 py-1.5 rounded-full border-0 focus:outline-none focus:ring-2 focus:ring-brand-400/30 cursor-pointer appearance-none"
                value={task.status_id ?? ''}
                onChange={(e) => updateTask.mutate({ status_id: e.target.value || undefined })}
                style={{
                  backgroundColor: (currentStatus?.color ?? '#94a3b8') + '22',
                  color: currentStatus?.color ?? '#64748b',
                }}
              >
                <option value="">No status</option>
                {statuses!.map((s) => (
                  <option key={s.id} value={s.id}>{s.name}</option>
                ))}
              </select>
            ) : (
              <div className="relative">
                <select
                  className={cn(
                    'text-xs font-semibold px-3 py-1.5 rounded-full border-0 focus:outline-none focus:ring-2 focus:ring-brand-400/30 cursor-pointer appearance-none pr-6',
                    STATUS_COLORS[task.status] ?? 'bg-ink-5/30 text-ink-2',
                  )}
                  value={task.status}
                  onChange={(e) => updateTask.mutate({ status: e.target.value })}
                >
                  {LEGACY_STATUSES.map((s) => (
                    <option key={s} value={s}>{STATUS_LABEL[s]}</option>
                  ))}
                </select>
                <div
                  className="absolute right-2 top-1/2 -translate-y-1/2 w-1.5 h-1.5 rounded-full pointer-events-none"
                  style={{ backgroundColor: STATUS_DOT[task.status] ?? '#94a3b8' }}
                />
              </div>
            )}
          </div>

          {/* Priority */}
          <div>
            <p className="text-[10px] font-semibold text-ink-4 uppercase tracking-wider mb-1.5">Priority</p>
            <select
              className={cn(
                'text-xs font-semibold px-3 py-1.5 rounded-full border-0 focus:outline-none focus:ring-2 focus:ring-brand-400/30 cursor-pointer',
                PRIORITY_COLORS[task.priority] ?? PRIORITY_COLORS[0],
              )}
              value={task.priority}
              onChange={(e) => updateTask.mutate({ priority: Number(e.target.value) })}
            >
              {Object.entries(PRIORITY_LABELS).map(([v, l]) => (
                <option key={v} value={v}>{l}</option>
              ))}
            </select>
          </div>

          {/* Due date */}
          <div>
            <p className="text-[10px] font-semibold text-ink-4 uppercase tracking-wider mb-1.5">Due</p>
            <input
              type="date"
              value={task.due_at ? task.due_at.slice(0, 10) : ''}
              onChange={(e) =>
                updateTask.mutate({
                  due_at: e.target.value ? new Date(e.target.value).toISOString() : undefined,
                })
              }
              className="text-xs font-medium text-ink-1 bg-canvas/50 border border-ink-5/30 rounded-full px-3 py-1.5 focus:outline-none focus:ring-2 focus:ring-brand-400/40 focus:border-brand-400"
            />
          </div>

          {/* Assignees */}
          <div>
            <p className="text-[10px] font-semibold text-ink-4 uppercase tracking-wider mb-1.5">Assignees</p>
            {workspaceId && <AssigneesPicker taskId={task.id} workspaceId={workspaceId} />}
          </div>

          {/* Recurrence */}
          <div>
            <p className="text-[10px] font-semibold text-ink-4 uppercase tracking-wider mb-1.5">Recurrence</p>
            <RecurrencePicker
              value={task.recurring_rule ?? null}
              onChange={(rule) => updateTask.mutate({ recurring_rule: rule ?? '' } as Partial<Task>)}
            />
          </div>

          <div className="ml-auto self-end text-[11px] text-ink-5">
            Updated {formatRelative(task.updated_at)}
          </div>
        </div>

        {/* Tags */}
        <div className="mb-5">
          <p className="text-[10px] font-semibold text-ink-4 uppercase tracking-wider mb-1.5">Tags</p>
          {workspaceId && <TagsPicker taskId={task.id} workspaceId={workspaceId} />}
        </div>

        {/* Description */}
        <div>
          <p className="text-[10px] font-semibold text-ink-4 uppercase tracking-wider mb-2">Description</p>
          <DescriptionEditor
            value={task.description}
            onSave={(next) => updateTask.mutate({ description: next })}
          />
        </div>
      </div>

      {/* Custom fields */}
      {workspaceId && task.list_id && (
        <div className="bg-surface border border-ink-5/30 rounded-2xl shadow-card p-6 mb-4">
          <CustomFieldsBlock
            taskId={task.id}
            listId={task.list_id}
            workspaceId={workspaceId}
            onManage={() => setShowFieldsManager(true)}
          />
        </div>
      )}

      {/* Dependencies */}
      {workspaceId && task.list_id && (
        <div className="bg-surface border border-ink-5/30 rounded-2xl shadow-card p-6 mb-4">
          <h3 className="text-sm font-semibold text-ink-1 mb-4">Dependencies</h3>
          <DependenciesPanel taskId={task.id} listId={task.list_id} workspaceId={workspaceId} />
        </div>
      )}

      {/* Time tracking */}
      <div className="bg-surface border border-ink-5/30 rounded-2xl shadow-card p-6 mb-4">
        <h3 className="text-sm font-semibold text-ink-1 mb-4">Time tracking</h3>
        <TimeEntriesList taskId={task.id} />
      </div>

      {/* Subtasks */}
      <div className="bg-surface border border-ink-5/30 rounded-2xl shadow-card p-6 mb-4">
        <h3 className="text-sm font-semibold text-ink-1 mb-4">Subtasks</h3>
        {workspaceId && <SubtasksPanel parentTask={task} workspaceId={workspaceId} />}
      </div>

      {/* Attachments */}
      <div className="bg-surface border border-ink-5/30 rounded-2xl shadow-card p-6 mb-4">
        <h3 className="text-sm font-semibold text-ink-1 mb-4">Attachments</h3>
        <AttachmentsPanel taskId={task.id} />
      </div>

      {/* Comments */}
      <div className="bg-surface border border-ink-5/30 rounded-2xl shadow-card p-6">
        <div className="flex items-center gap-2 mb-5">
          <h3 className="text-sm font-semibold text-ink-1">Comments</h3>
          {(comments?.length ?? 0) > 0 && (
            <span className="text-[10px] font-medium bg-ink-5/30 text-ink-3 rounded-full px-2 py-0.5">
              {comments!.length}
            </span>
          )}
        </div>

        <div className="space-y-3 mb-4">
          {(comments ?? []).map((c) => (
            <div key={c.id} className="bg-canvas/60 border border-ink-5/20 rounded-xl px-4 py-3">
              <p className="text-sm text-ink-1 leading-relaxed whitespace-pre-wrap">{c.body}</p>
              <p className="text-[11px] text-ink-4 mt-1.5">{formatRelative(c.created_at)}</p>
            </div>
          ))}
          {comments?.length === 0 && (
            <p className="text-sm text-ink-4 py-2">No comments yet. Be the first.</p>
          )}
        </div>

        <div className="flex gap-2">
          <input
            className="flex-1 h-9 bg-canvas/50 border border-ink-5/30 rounded-xl px-3.5 text-sm text-ink-1 focus:outline-none focus:ring-2 focus:ring-brand-400/40 focus:border-brand-400 placeholder:text-ink-5 transition-all"
            placeholder="Write a comment… (use @email to mention)"
            value={commentBody}
            onChange={(e) => setCommentBody(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && !e.shiftKey && commentBody.trim()) {
                e.preventDefault()
                addComment.mutate()
              }
            }}
          />
          <Button
            onClick={() => addComment.mutate()}
            disabled={!commentBody.trim() || addComment.isPending}
            size="sm"
          >
            Send
          </Button>
        </div>
      </div>

      <div className="mt-5 flex justify-end gap-2">
        {task.archived ? (
          <Button
            variant="secondary"
            size="sm"
            onClick={() => archiveTask.mutate(false)}
            disabled={archiveTask.isPending}
          >
            Unarchive
          </Button>
        ) : task.completed_at ? (
          <Button
            variant="secondary"
            size="sm"
            onClick={() => archiveTask.mutate(true)}
            disabled={archiveTask.isPending}
            title="Hide this completed task from the board. It still shows in your accomplishments report."
          >
            Archive
          </Button>
        ) : null}
        <Button
          variant="danger"
          size="sm"
          onClick={() => {
            if (confirm('Delete this task? This cannot be undone.')) deleteTask.mutate()
          }}
          disabled={deleteTask.isPending}
        >
          Delete task
        </Button>
      </div>

      {showFieldsManager && task.list_id && (
        <CustomFieldsManager listId={task.list_id} onClose={() => setShowFieldsManager(false)} />
      )}
    </div>
  )
}
