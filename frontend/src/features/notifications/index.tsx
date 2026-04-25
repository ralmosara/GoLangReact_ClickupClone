import { useMutation, useQuery } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { api } from '../../lib/api'
import { queryClient } from '../../lib/queryClient'
import { Button, PageSpinner } from '../../components/ui'
import { cn, formatRelative } from '../../lib/utils'
import type { Notification } from '../../types'

export function NotificationsPage() {
  const navigate = useNavigate()

  const { data: notifications, isLoading } = useQuery({
    queryKey: ['notifications'],
    queryFn: () => api.get('me/notifications?limit=100').json<Notification[]>(),
  })

  const markRead = useMutation({
    mutationFn: (id: string) => api.patch(`notifications/${id}/read`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['notifications'] })
      queryClient.invalidateQueries({ queryKey: ['notifications-unread'] })
    },
  })

  const markAll = useMutation({
    mutationFn: () => api.post('notifications/read-all'),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['notifications'] })
      queryClient.invalidateQueries({ queryKey: ['notifications-unread'] })
    },
  })

  const unread = (notifications ?? []).filter((n) => !n.read_at)
  const total = (notifications ?? []).length

  return (
    <div className="max-w-2xl mx-auto px-6 py-10 animate-slide-up">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-xl font-bold text-ink-1">Notifications</h1>
          <p className="text-sm text-ink-4 mt-0.5">
            {unread.length > 0
              ? `${unread.length} unread notification${unread.length !== 1 ? 's' : ''}`
              : 'All caught up'}
          </p>
        </div>
        {unread.length > 0 && (
          <Button variant="secondary" size="sm" onClick={() => markAll.mutate()}>
            Mark all read
          </Button>
        )}
      </div>

      {isLoading && <PageSpinner />}

      {!isLoading && total === 0 && (
        <div className="text-center py-20 bg-surface/50 border border-ink-5/20 rounded-2xl">
          <div className="w-14 h-14 rounded-2xl bg-brand-50 border border-brand-100 flex items-center justify-center mx-auto mb-4">
            <svg className="w-7 h-7 text-brand-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9" />
            </svg>
          </div>
          <p className="font-semibold text-ink-2">You're all caught up!</p>
          <p className="text-sm text-ink-4 mt-1">New activity will appear here.</p>
        </div>
      )}

      {!isLoading && total > 0 && (
        <div className="space-y-2 animate-fade-in">
          {(notifications ?? []).map((n) => {
            const p = (n.payload as Record<string, unknown>) ?? {}
            const taskId = typeof p.task_id === 'string' ? (p.task_id as string) : null
            const title = (p.task_name as string) ?? labelFor(n.kind)
            const excerpt = n.kind === 'comment.mention' ? (p.excerpt as string) : null
            return (
              <div
                key={n.id}
                className={cn(
                  'bg-surface border rounded-xl p-4 flex items-start gap-4 transition-all duration-150 shadow-card',
                  n.read_at ? 'border-ink-5/20 opacity-70' : 'border-brand-200/50',
                )}
              >
                <div className="mt-1 shrink-0">
                  <div className={cn('w-2 h-2 rounded-full', n.read_at ? 'bg-ink-5/40' : 'bg-brand-500')} />
                </div>

                <button
                  onClick={() => {
                    if (!n.read_at) markRead.mutate(n.id)
                    // Best-effort deep link. workspace id is recoverable from the URL when we land.
                    if (taskId) {
                      const m = location.pathname.match(/\/workspaces\/([^/]+)/)
                      if (m) navigate(`/workspaces/${m[1]}/tasks/${taskId}`)
                    }
                  }}
                  className="flex-1 text-left min-w-0"
                >
                  <p className="text-[11px] font-bold uppercase tracking-wider text-brand-600">
                    {labelFor(n.kind)}
                  </p>
                  <p className="text-sm font-medium text-ink-1 truncate mt-0.5">{title}</p>
                  {excerpt && <p className="text-xs text-ink-4 mt-0.5 line-clamp-2">{excerpt}</p>}
                  <p className="text-[11px] text-ink-4 mt-1">{formatRelative(n.created_at)}</p>
                </button>

                {!n.read_at && (
                  <button
                    onClick={() => markRead.mutate(n.id)}
                    className="text-xs text-brand-600 hover:text-brand-700 font-medium shrink-0 transition-colors"
                  >
                    Mark read
                  </button>
                )}
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}

function labelFor(kind: string): string {
  switch (kind) {
    case 'comment.mention': return 'Mention'
    case 'task.assigned':   return 'Assigned'
    default:                return kind.replace(/_/g, ' ').replace(/\./g, ' · ')
  }
}
