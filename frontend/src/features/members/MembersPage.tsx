import { useEffect, useMemo, useState } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import { Link, useParams } from 'react-router-dom'
import { api } from '../../lib/api'
import { queryClient } from '../../lib/queryClient'
import { Button, Input, PageSpinner } from '../../components/ui'
import { useInviteMember } from '../../hooks/useAuth'
import { formatRelative } from '../../lib/utils'
import type { Member, Role } from '../../types'

interface EffectivePermissionsResponse {
  workspace_id: string
  user_id: string
  permissions: string[]
}

interface InviteLookup {
  exists: boolean
  already_member: boolean
  name?: string
}

export function MembersPage() {
  const { workspaceId } = useParams<{ workspaceId: string }>()

  const { data: members, isLoading } = useQuery({
    queryKey: ['workspace-members', workspaceId],
    queryFn: () => api.get(`workspaces/${workspaceId}/members`).json<Member[]>(),
    enabled: !!workspaceId,
  })

  const { data: roles } = useQuery({
    queryKey: ['workspace-roles', workspaceId],
    queryFn: () => api.get(`workspaces/${workspaceId}/roles`).json<Role[]>(),
    enabled: !!workspaceId,
  })

  const { data: perms } = useQuery({
    queryKey: ['workspace-effective-permissions', workspaceId],
    queryFn: () =>
      api
        .get(`workspaces/${workspaceId}/roles/effective`)
        .json<EffectivePermissionsResponse>(),
    enabled: !!workspaceId,
  })
  const canInvite = perms?.permissions?.includes('member.invite') ?? false
  const canManageUsers = perms?.permissions?.includes('user.manage') ?? false
  const canRemove = perms?.permissions?.includes('member.remove') ?? false

  const remove = useMutation({
    mutationFn: (userID: string) => api.delete(`workspaces/${workspaceId}/members/${userID}`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['workspace-members', workspaceId] }),
  })

  return (
    <div className="max-w-2xl mx-auto px-6 py-10 animate-slide-up">
      <h1 className="text-xl font-bold text-ink-1 mb-2">Workspace members</h1>
      <p className="text-sm text-ink-4 mb-6">
        Invite an already-registered user to this workspace. To create a new account,
        use {canManageUsers
          ? <Link to={`/workspaces/${workspaceId}/users`} className="text-brand-600 hover:text-brand-700 font-medium">User Management</Link>
          : <span className="text-ink-3">User Management</span>
        }.
      </p>

      {canInvite && <InviteForm workspaceId={workspaceId!} roles={roles ?? []} canManageUsers={canManageUsers} />}

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
              {m.role !== 'owner' && canRemove && (
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

// InviteForm — strict invite-existing only. The form is just email + role.
// As soon as a complete-looking email is typed it debounces 350 ms and
// hits /workspaces/{id}/members/lookup; the result drives one of three
// states:
//
//   exists:false                → "No user with this email yet" + deep link
//                                  to User Management (when the caller has
//                                  user.manage); submit disabled.
//   exists:true, not member     → green hint "User exists — adding to
//                                  workspace"; submit "Add to workspace".
//   exists:true, already_member → amber hint "Already a member"; submit
//                                  disabled.
function InviteForm({
  workspaceId,
  roles,
  canManageUsers,
}: {
  workspaceId: string
  roles: Role[]
  canManageUsers: boolean
}) {
  const [email, setEmail] = useState('')
  const [role, setRole] = useState('member')

  const sortedRoles = useMemo(() => {
    const copy = [...roles]
    copy.sort((a, b) => {
      if (a.is_builtin !== b.is_builtin) return a.is_builtin ? -1 : 1
      if (a.is_builtin) return b.rank - a.rank
      return a.name.localeCompare(b.name)
    })
    return copy
  }, [roles])

  useEffect(() => {
    if (!sortedRoles.length) return
    if (!sortedRoles.some((r) => r.name === role)) {
      setRole(sortedRoles.find((r) => r.name === 'member')?.name ?? sortedRoles[0].name)
    }
  }, [sortedRoles, role])

  const trimmedEmail = email.trim().toLowerCase()
  const looksLikeEmail = /^[^@\s]+@[^@\s]+\.[^@\s]+$/.test(trimmedEmail)
  const [debouncedEmail, setDebouncedEmail] = useState('')
  useEffect(() => {
    if (!looksLikeEmail) {
      setDebouncedEmail('')
      return
    }
    const t = window.setTimeout(() => setDebouncedEmail(trimmedEmail), 350)
    return () => window.clearTimeout(t)
  }, [trimmedEmail, looksLikeEmail])

  const lookup = useQuery({
    queryKey: ['member-invite-lookup', workspaceId, debouncedEmail],
    queryFn: () =>
      api
        .get(`workspaces/${workspaceId}/members/lookup`, {
          searchParams: { email: debouncedEmail },
        })
        .json<InviteLookup>(),
    enabled: !!debouncedEmail,
    staleTime: 30_000,
  })

  const result = lookup.data
  const userExists = !!result?.exists
  const alreadyMember = !!result?.already_member

  const invite = useInviteMember(workspaceId)

  const submit = () => {
    if (!trimmedEmail || alreadyMember || !userExists) return
    invite.mutate(
      { email: trimmedEmail, role },
      { onSuccess: () => setEmail('') },
    )
  }

  const errBody = invite.error
    ? (invite.error as Error & { message: string }).message
    : null

  // Submit gating:
  //   - email must look like one
  //   - lookup must have settled
  //   - user must exist
  //   - user must not already be a member
  //   - mutation not in flight
  const submitDisabled =
    !looksLikeEmail ||
    !lookup.isFetched ||
    !userExists ||
    alreadyMember ||
    invite.isPending

  return (
    <div className="bg-surface border border-ink-5/30 rounded-2xl p-5 shadow-card mb-5">
      <p className="text-xs font-semibold text-ink-2 mb-3">Invite a member</p>
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
        <Input
          type="email"
          placeholder="teammate@company.com"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
        />
        <select
          className="h-9 px-3 bg-surface border border-ink-5/40 rounded-lg text-sm text-ink-1 focus:outline-none focus:ring-2 focus:ring-brand-400/40"
          value={role}
          onChange={(e) => setRole(e.target.value)}
        >
          {sortedRoles.map((r) => (
            <option key={r.id} value={r.name}>
              {r.name}{r.is_builtin ? '' : ' (custom)'}
            </option>
          ))}
          {sortedRoles.length === 0 && <option value="member">member</option>}
        </select>
      </div>

      {/* Status hints driven by the lookup result. */}
      {looksLikeEmail && lookup.isFetching && (
        <p className="text-[11px] text-ink-4 mt-2">Checking…</p>
      )}
      {alreadyMember && (
        <p className="text-[11px] text-amber-600 mt-2">
          This user is already a member of the workspace.
        </p>
      )}
      {userExists && !alreadyMember && (
        <p className="text-[11px] text-emerald-600 mt-2">
          {result?.name ? `${result.name} ` : ''}has an account — they'll be added to this workspace.
        </p>
      )}
      {looksLikeEmail && lookup.isFetched && !userExists && (
        <p className="text-[11px] text-ink-4 mt-2">
          No user with this email yet.{' '}
          {canManageUsers ? (
            <Link
              to={`/workspaces/${workspaceId}/users`}
              className="text-brand-600 hover:text-brand-700 font-medium"
            >
              Create them in User Management →
            </Link>
          ) : (
            <span>Ask a workspace owner to create the account in User Management.</span>
          )}
        </p>
      )}

      <div className="flex items-center justify-end gap-3 mt-3">
        <Button
          onClick={submit}
          disabled={submitDisabled}
          loading={invite.isPending}
          size="sm"
        >
          {alreadyMember ? 'Already a member' : userExists ? 'Add to workspace' : 'Invite'}
        </Button>
      </div>
      {errBody && <p className="text-red-500 text-xs mt-2">{errBody}</p>}
    </div>
  )
}
