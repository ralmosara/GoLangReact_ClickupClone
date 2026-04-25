import { useState } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import { useParams } from 'react-router-dom'
import { api } from '../../lib/api'
import { queryClient } from '../../lib/queryClient'
import { Button, Input, PageSpinner } from '../../components/ui'
import { formatRelative } from '../../lib/utils'
import type { Member } from '../../types'

const ROLES = ['owner', 'admin', 'member', 'guest'] as const

export function MembersPage() {
  const { workspaceId } = useParams<{ workspaceId: string }>()
  const [email, setEmail] = useState('')
  const [role, setRole] = useState<(typeof ROLES)[number]>('member')

  const { data: members, isLoading } = useQuery({
    queryKey: ['workspace-members', workspaceId],
    queryFn: () => api.get(`workspaces/${workspaceId}/members`).json<Member[]>(),
    enabled: !!workspaceId,
  })

  const add = useMutation({
    mutationFn: () =>
      api.post(`workspaces/${workspaceId}/members`, { json: { email, role } }).json<Member>(),
    onSuccess: () => {
      setEmail('')
      queryClient.invalidateQueries({ queryKey: ['workspace-members', workspaceId] })
    },
  })

  const remove = useMutation({
    mutationFn: (userID: string) => api.delete(`workspaces/${workspaceId}/members/${userID}`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['workspace-members', workspaceId] }),
  })

  return (
    <div className="max-w-2xl mx-auto px-6 py-10 animate-slide-up">
      <h1 className="text-xl font-bold text-ink-1 mb-2">Workspace members</h1>
      <p className="text-sm text-ink-4 mb-6">Invite teammates by email. They must already have an account (SSO invites arrive in M8).</p>

      <div className="bg-surface border border-ink-5/30 rounded-2xl p-5 shadow-card mb-5">
        <p className="text-xs font-semibold text-ink-2 mb-3">Add a member</p>
        <div className="flex items-center gap-2">
          <Input
            type="email"
            placeholder="teammate@company.com"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && email.trim() && add.mutate()}
          />
          <select
            className="h-9 px-3 bg-surface border border-ink-5/40 rounded-lg text-sm text-ink-1 focus:outline-none focus:ring-2 focus:ring-brand-400/40"
            value={role}
            onChange={(e) => setRole(e.target.value as (typeof ROLES)[number])}
          >
            {ROLES.map((r) => <option key={r} value={r}>{r}</option>)}
          </select>
          <Button onClick={() => add.mutate()} disabled={!email.trim() || add.isPending} loading={add.isPending} size="sm">
            Add
          </Button>
        </div>
        {add.error && <p className="text-red-500 text-xs mt-2">{String(add.error)}</p>}
      </div>

      <div className="bg-surface border border-ink-5/30 rounded-2xl shadow-card">
        {isLoading && <div className="p-5"><PageSpinner /></div>}
        {!isLoading && (members ?? []).length === 0 && (
          <p className="p-6 text-sm text-ink-4">No members yet.</p>
        )}
        <ul>
          {(members ?? []).map((m) => (
            <li key={m.user_id} className="flex items-center gap-3 px-5 py-3 border-b border-ink-5/20 last:border-b-0">
              <div className="w-8 h-8 rounded-full bg-brand-gradient text-xs font-bold text-white flex items-center justify-center">
                {(m.name || m.email || '?').slice(0, 1).toUpperCase()}
              </div>
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium text-ink-1 truncate">{m.name || m.email}</p>
                <p className="text-xs text-ink-4 truncate">{m.email}</p>
              </div>
              <span className="text-[11px] font-semibold bg-ink-5/30 text-ink-2 px-2 py-0.5 rounded-full capitalize">
                {m.role}
              </span>
              <span className="text-[11px] text-ink-4 hidden sm:block">{formatRelative(m.created_at)}</span>
              {m.role !== 'owner' && (
                <button
                  onClick={() => {
                    if (confirm(`Remove ${m.email || m.name} from the workspace?`)) remove.mutate(m.user_id)
                  }}
                  className="text-ink-4 hover:text-red-500 p-1.5 rounded hover:bg-red-50 transition-colors"
                  title="Remove"
                >
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              )}
            </li>
          ))}
        </ul>
      </div>
    </div>
  )
}
