import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { rooms } from '../../../lib/ws'
import { useWsEvent, useWsRooms } from '../../../hooks/useWebSocket'
import { useAuthStore } from '../../../store/auth'
import type { Notification } from '../../../types'

interface Toast {
  id: string
  n: Notification
}

/**
 * Floating toast that renders incoming `notification.created` WS events.
 * Mounted once at the workspace layout so it is visible everywhere inside a workspace.
 */
export function NotificationToast({ workspaceId }: { workspaceId: string }) {
  const navigate = useNavigate()
  const userId = useAuthStore((s) => s.user?.id)
  const [toasts, setToasts] = useState<Toast[]>([])

  useWsRooms(userId ? [rooms.user(userId)] : [])
  useWsEvent<Notification>('notification.created', (e) => {
    const n = e.payload
    if (!n) return
    setToasts((prev) => [...prev, { id: n.id, n }])
    // auto-dismiss after 6s
    setTimeout(() => {
      setToasts((prev) => prev.filter((t) => t.id !== n.id))
    }, 6000)
  })

  // cleanup on unmount
  useEffect(() => () => setToasts([]), [])

  if (toasts.length === 0) return null

  return (
    <div className="fixed right-4 bottom-4 z-40 flex flex-col gap-2 w-[320px] pointer-events-none">
      {toasts.map((t) => (
        <button
          key={t.id}
          onClick={() => {
            setToasts((prev) => prev.filter((x) => x.id !== t.id))
            navigate(`/workspaces/${workspaceId}/notifications`)
          }}
          className="pointer-events-auto bg-surface border border-ink-5/30 rounded-xl shadow-modal px-4 py-3 text-left animate-slide-up"
        >
          <p className="text-[11px] font-bold uppercase tracking-wider text-brand-600 mb-1">
            {labelFor(t.n.kind)}
          </p>
          <p className="text-sm text-ink-1 font-medium line-clamp-2">{renderTitle(t.n)}</p>
          {renderExcerpt(t.n) && (
            <p className="text-xs text-ink-4 mt-1 line-clamp-2">{renderExcerpt(t.n)}</p>
          )}
        </button>
      ))}
    </div>
  )
}

function labelFor(kind: string): string {
  switch (kind) {
    case 'comment.mention': return 'You were mentioned'
    case 'task.assigned':   return 'Assigned'
    default:                return kind.replace(/_/g, ' ')
  }
}

function renderTitle(n: Notification) {
  const p = n.payload as Record<string, unknown>
  if (typeof p?.task_name === 'string') return p.task_name
  if (typeof p?.excerpt === 'string') return p.excerpt as string
  return labelFor(n.kind)
}

function renderExcerpt(n: Notification): string | null {
  const p = n.payload as Record<string, unknown>
  if (typeof p?.excerpt === 'string' && n.kind === 'comment.mention') return p.excerpt as string
  return null
}
