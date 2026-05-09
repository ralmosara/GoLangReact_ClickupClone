import { useMutation, useQuery } from '@tanstack/react-query'
import { Laptop, Shield, Trash2 } from 'lucide-react'

import { api } from '../../../lib/api'
import { queryClient } from '../../../lib/queryClient'
import { Button, Spinner } from '../../../components/ui'
import { formatRelative } from '../../../lib/utils'
import { t } from '../../../lib/i18n'

// Session is the wire shape returned by GET /me/sessions. The backend
// also flags `current` for the row matching the caller's own JWT — we
// use that to disable the per-row Revoke button and to highlight the
// row visually.
interface Session {
  id: string
  label: string
  user_agent: string
  ip?: string
  created_at: string
  last_seen_at: string
  expires_at: string
  current: boolean
}

export function ActiveSessionsPanel() {
  const { data: sessions = [], isLoading } = useQuery({
    queryKey: ['me-sessions'],
    queryFn: () => api.get('me/sessions').json<Session[]>(),
  })

  const revoke = useMutation({
    mutationFn: (id: string) => api.delete(`me/sessions/${id}`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['me-sessions'] }),
  })

  const revokeOthers = useMutation({
    mutationFn: () => api.post('me/sessions/revoke-others').json<{ revoked: number }>(),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['me-sessions'] }),
  })

  const otherCount = sessions.filter((s) => !s.current).length

  return (
    <section className="bg-surface border border-ink-5/30 rounded-2xl shadow-card p-6 dark:bg-white/[0.04] dark:border-white/10">
      <header className="flex items-start justify-between gap-4 mb-5">
        <div>
          <h2 className="text-sm font-semibold text-ink-1 flex items-center gap-2 dark:text-white">
            <Shield className="w-4 h-4 text-brand-500" />
            {t('sessions.title')}
          </h2>
          <p className="text-xs text-ink-4 mt-1 dark:text-white/50">
            Each row is a browser or device with a valid login. Revoking signs that device out immediately.
          </p>
        </div>
        <Button
          variant="secondary"
          size="sm"
          disabled={otherCount === 0 || revokeOthers.isPending}
          loading={revokeOthers.isPending}
          onClick={() => {
            if (confirm('Sign out all other devices? They will need to log in again.')) {
              revokeOthers.mutate()
            }
          }}
        >
          {t('sessions.signout_others')}
        </Button>
      </header>

      {isLoading ? (
        <div className="py-8 grid place-items-center"><Spinner /></div>
      ) : sessions.length === 0 ? (
        <p className="text-xs text-ink-4 text-center py-6 dark:text-white/40">
          No active sessions found. (Older logins from before sessions were tracked won't appear here.)
        </p>
      ) : (
        <ul className="divide-y divide-ink-5/20 dark:divide-white/5">
          {sessions.map((s) => (
            <li key={s.id} className="py-3 flex items-start gap-3">
              <span className="mt-0.5 w-8 h-8 rounded-lg bg-ink-5/20 grid place-items-center text-ink-3 shrink-0 dark:bg-white/10 dark:text-white/60">
                <Laptop className="w-4 h-4" />
              </span>
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium text-ink-1 flex items-center gap-2 dark:text-white">
                  {s.label || 'Unknown device'}
                  {s.current && (
                    <span className="text-[10px] font-semibold uppercase tracking-wider px-1.5 py-0.5 rounded bg-green-50 text-green-700 ring-1 ring-green-200 dark:bg-green-500/15 dark:text-green-300 dark:ring-green-500/30">
                      {t('sessions.this_device')}
                    </span>
                  )}
                </p>
                <p className="text-[11px] text-ink-4 mt-0.5 truncate dark:text-white/50">
                  {s.user_agent || '—'}{s.ip ? ` · ${s.ip}` : ''}
                </p>
                <p className="text-[11px] text-ink-3 mt-0.5 dark:text-white/40">
                  Last active {formatRelative(s.last_seen_at)} · Signed in {formatRelative(s.created_at)}
                </p>
              </div>
              <button
                onClick={() => {
                  if (s.current) return
                  if (confirm(`Revoke "${s.label || 'this session'}"?`)) revoke.mutate(s.id)
                }}
                disabled={s.current || revoke.isPending}
                className="self-start h-7 w-7 grid place-items-center rounded-md text-ink-4 hover:text-red-500 hover:bg-red-50 disabled:opacity-30 disabled:cursor-not-allowed transition-colors dark:hover:bg-red-500/15"
                title={s.current ? "Can't revoke your current session here — use sign out" : 'Revoke session'}
                aria-label="Revoke session"
              >
                <Trash2 className="w-3.5 h-3.5" />
              </button>
            </li>
          ))}
        </ul>
      )}
    </section>
  )
}
