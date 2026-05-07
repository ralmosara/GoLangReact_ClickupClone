import { useState, useEffect } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import { useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft, MoreVertical, Archive, Trash2, Clock, Calendar, Hash, User, List as ListIcon, Type } from 'lucide-react'
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
    <div className="max-w-6xl mx-auto px-4 sm:px-6 py-6 animate-slide-up">
      {/* Header / Breadcrumbs */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 mb-6">
        <button
          onClick={() => navigate(-1)}
          className="flex items-center gap-1.5 text-xs font-medium text-ink-4 hover:text-ink-1 transition-colors group w-fit"
        >
          <ArrowLeft className="w-3.5 h-3.5 group-hover:-translate-x-0.5 transition-transform" />
          Back to list
        </button>
        <div className="flex items-center gap-3">
          <TimeTracker taskId={task.id} />
          <div className="h-4 w-px bg-ink-5/20 hidden sm:block" />
          <div className="flex items-center gap-1.5">
             {task.archived ? (
                <Button variant="secondary" size="xs" onClick={() => archiveTask.mutate(false)} disabled={archiveTask.isPending}>
                  Unarchive
                </Button>
              ) : task.completed_at ? (
                <Button variant="secondary" size="xs" onClick={() => archiveTask.mutate(true)} disabled={archiveTask.isPending}>
                  Archive
                </Button>
              ) : null}
              <Button
                variant="danger"
                size="xs"
                onClick={() => {
                  if (confirm('Delete this task? This cannot be undone.')) deleteTask.mutate()
                }}
                disabled={deleteTask.isPending}
              >
                <Trash2 className="w-3.5 h-3.5 mr-1" />
                Delete
              </Button>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        {/* Main Content (Left) */}
        <div className="lg:col-span-8 space-y-6">
          <div className="bg-surface border border-ink-5/30 rounded-2xl shadow-card p-4 sm:p-6">
            {editingName ? (
              <input
                autoFocus
                className="w-full text-2xl font-bold text-ink-1 border-b-2 border-brand-400 focus:outline-none mb-6 pb-1 bg-transparent"
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
                className="text-2xl font-bold text-ink-1 mb-6 cursor-text hover:text-brand-600 transition-colors leading-tight"
                title="Click to edit"
              >
                {task.name}
              </h1>
            )}

            {/* Description */}
            <div className="space-y-3">
              <div className="flex items-center gap-2 text-[10px] font-bold text-ink-4 uppercase tracking-wider">
                <Type className="w-3 h-3" />
                Description
              </div>
              <DescriptionEditor
                value={task.description}
                onSave={(next) => updateTask.mutate({ description: next })}
              />
            </div>
          </div>

          {/* Subtasks */}
          <div className="bg-surface border border-ink-5/30 rounded-2xl shadow-card p-4 sm:p-6">
            <h3 className="text-sm font-bold text-ink-1 mb-4 flex items-center gap-2">
              <ListIcon className="w-4 h-4 text-brand-500" />
              Subtasks
            </h3>
            {workspaceId && <SubtasksPanel parentTask={task} workspaceId={workspaceId} />}
          </div>

          {/* Comments */}
          <div className="bg-surface border border-ink-5/30 rounded-2xl shadow-card p-4 sm:p-6">
            <div className="flex items-center gap-2 mb-6">
              <h3 className="text-sm font-bold text-ink-1">Comments</h3>
              {(comments?.length ?? 0) > 0 && (
                <span className="text-[10px] font-bold bg-ink-5/30 text-ink-3 rounded-full px-2 py-0.5">
                  {comments!.length}
                </span>
              )}
            </div>

            <div className="space-y-4 mb-6">
              {(comments ?? []).map((c) => (
                <div key={c.id} className="flex gap-3">
                  <div className="w-8 h-8 rounded-full bg-brand-100 flex items-center justify-center text-[10px] font-bold text-brand-600 shrink-0 uppercase">
                    {c.author_id?.slice(0, 1) || '?'}
                  </div>
                  <div className="flex-1">
                    <div className="bg-canvas/60 border border-ink-5/20 rounded-2xl px-4 py-3">
                      <p className="text-sm text-ink-1 leading-relaxed whitespace-pre-wrap">{c.body}</p>
                    </div>
                    <p className="text-[10px] text-ink-4 mt-1.5 ml-1">
                      {formatRelative(c.created_at)}
                    </p>
                  </div>
                </div>
              ))}
              {comments?.length === 0 && (
                <p className="text-sm text-ink-4 py-4 text-center border-2 border-dashed border-ink-5/20 rounded-2xl">
                  No comments yet. Start the conversation.
                </p>
              )}
            </div>

            <div className="flex gap-2">
              <input
                className="flex-1 h-10 bg-canvas/50 border border-ink-5/30 rounded-xl px-4 text-sm text-ink-1 focus:outline-none focus:ring-2 focus:ring-brand-400/40 focus:border-brand-400 placeholder:text-ink-5 transition-all"
                placeholder="Write a comment…"
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
              >
                Send
              </Button>
            </div>
          </div>
        </div>

        {/* Sidebar (Right) */}
        <div className="lg:col-span-4 space-y-6">
          <div className="bg-surface border border-ink-5/30 rounded-2xl shadow-card divide-y divide-ink-5/20">
            {/* Status */}
            <div className="p-4 sm:p-5">
              <p className="text-[10px] font-bold text-ink-4 uppercase tracking-wider mb-3 flex items-center gap-1.5">
                <Hash className="w-3 h-3" />
                Status
              </p>
              {hasCustomStatuses ? (
                <select
                  className="w-full text-xs font-bold px-3 py-2 rounded-lg border-0 focus:outline-none focus:ring-2 focus:ring-brand-400/30 cursor-pointer appearance-none"
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
                      'w-full text-xs font-bold px-3 py-2 rounded-lg border-0 focus:outline-none focus:ring-2 focus:ring-brand-400/30 cursor-pointer appearance-none pr-8',
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
                    className="absolute right-3 top-1/2 -translate-y-1/2 w-2 h-2 rounded-full pointer-events-none"
                    style={{ backgroundColor: STATUS_DOT[task.status] ?? '#94a3b8' }}
                  />
                </div>
              )}
            </div>

            {/* Priority */}
            <div className="p-4 sm:p-5">
              <p className="text-[10px] font-bold text-ink-4 uppercase tracking-wider mb-3">Priority</p>
              <select
                className={cn(
                  'w-full text-xs font-bold px-3 py-2 rounded-lg border-0 focus:outline-none focus:ring-2 focus:ring-brand-400/30 cursor-pointer',
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
            <div className="p-4 sm:p-5">
              <p className="text-[10px] font-bold text-ink-4 uppercase tracking-wider mb-3 flex items-center gap-1.5">
                <Calendar className="w-3 h-3" />
                Dates
              </p>
              <div className="space-y-3">
                <div>
                  <label className="block text-[9px] text-ink-4 mb-1">Due date</label>
                  <input
                    type="date"
                    value={task.due_at ? task.due_at.slice(0, 10) : ''}
                    onChange={(e) =>
                      updateTask.mutate({
                        due_at: e.target.value ? new Date(e.target.value).toISOString() : undefined,
                      })
                    }
                    className="w-full text-xs font-medium text-ink-1 bg-canvas/50 border border-ink-5/30 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-brand-400/40 focus:border-brand-400"
                  />
                </div>
                <div>
                    <RecurrencePicker
                    value={task.recurring_rule ?? null}
                    onChange={(rule) => updateTask.mutate({ recurring_rule: rule ?? '' } as Partial<Task>)}
                    />
                </div>
              </div>
            </div>

            {/* Assignees */}
            <div className="p-4 sm:p-5">
              <p className="text-[10px] font-bold text-ink-4 uppercase tracking-wider mb-3 flex items-center gap-1.5">
                <User className="w-3 h-3" />
                Assignees
              </p>
              {workspaceId && <AssigneesPicker taskId={task.id} workspaceId={workspaceId} />}
            </div>

            {/* Tags */}
            <div className="p-4 sm:p-5">
              <p className="text-[10px] font-bold text-ink-4 uppercase tracking-wider mb-3">Tags</p>
              {workspaceId && <TagsPicker taskId={task.id} workspaceId={workspaceId} />}
            </div>
          </div>

          {/* Extra Blocks */}
          <div className="space-y-4">
            {workspaceId && task.list_id && (
                <div className="bg-surface border border-ink-5/30 rounded-2xl shadow-card p-4 sm:p-5">
                <CustomFieldsBlock
                    taskId={task.id}
                    listId={task.list_id}
                    workspaceId={workspaceId}
                    onManage={() => setShowFieldsManager(true)}
                />
                </div>
            )}

            {workspaceId && task.list_id && (
                <div className="bg-surface border border-ink-5/30 rounded-2xl shadow-card p-4 sm:p-5">
                <h3 className="text-xs font-bold text-ink-1 mb-3">Dependencies</h3>
                <DependenciesPanel taskId={task.id} listId={task.list_id} workspaceId={workspaceId} />
                </div>
            )}

            <div className="bg-surface border border-ink-5/30 rounded-2xl shadow-card p-4 sm:p-5">
                <h3 className="text-xs font-bold text-ink-1 mb-3 flex items-center gap-2">
                    <Clock className="w-3.5 h-3.5 text-brand-500" />
                    Time Logs
                </h3>
                <TimeEntriesList taskId={task.id} />
            </div>

            <div className="bg-surface border border-ink-5/30 rounded-2xl shadow-card p-4 sm:p-5">
                <h3 className="text-xs font-bold text-ink-1 mb-3">Attachments</h3>
                <AttachmentsPanel taskId={task.id} />
            </div>
          </div>

          <div className="text-center text-[10px] text-ink-5 px-4">
            Created {formatRelative(task.created_at)} · Updated {formatRelative(task.updated_at)}
          </div>
        </div>
      </div>

      {showFieldsManager && task.list_id && (
        <CustomFieldsManager listId={task.list_id} onClose={() => setShowFieldsManager(false)} />
      )}
    </div>
  )
}
