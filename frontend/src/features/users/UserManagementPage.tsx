// UserManagementPage — workspace-scoped admin surface for the global
// users table. Only visible to users with the `user.manage` permission
// on the current workspace (owner + admin built-ins by default).
//
// Capabilities:
//   - Create user (email + name + password + workspace role) — server creates
//     the global user and adds them as a member of this workspace in one shot.
//   - List every user that's a member of this workspace, with role chips and
//     joined-at timestamps.
//   - Edit display name inline.
//   - Reset password (admin sets a new value; old one is discarded).
//   - Delete user account (hard delete — cascades through workspace_members,
//     mfa_secrets, oauth_identities, etc.).
//
// The MembersPage stays as the lighter "invite an existing user" surface;
// this page is for full account-lifecycle work.
import { useEffect, useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useParams } from 'react-router-dom'
import { api } from '../../lib/api'
import { Button, Input, PageSpinner } from '../../components/ui'
import {
  useCreateUser,
  useDeleteUser,
  useResetUserPassword,
  useUpdateUser,
  useWorkspaceUsers,
} from '../../hooks/useUsers'
import { formatRelative } from '../../lib/utils'
import type { Role } from '../../types'

interface EffectivePermissionsResponse {
  workspace_id: string
  user_id: string
  permissions: string[]
}

export function UserManagementPage() {
  const { workspaceId } = useParams<{ workspaceId: string }>()

  // Permission gate — same pattern as MembersPage. Hits /roles/effective so
  // custom-role grants are respected.
  const { data: perms, isLoading: permsLoading } = useQuery({
    queryKey: ['workspace-effective-permissions', workspaceId],
    queryFn: () =>
      api
        .get(`workspaces/${workspaceId}/roles/effective`)
        .json<EffectivePermissionsResponse>(),
    enabled: !!workspaceId,
  })
  const canManage = perms?.permissions?.includes('user.manage') ?? false

  const { data: roles } = useQuery({
    queryKey: ['workspace-roles', workspaceId],
    queryFn: () => api.get(`workspaces/${workspaceId}/roles`).json<Role[]>(),
    enabled: !!workspaceId && canManage,
  })

  const { data: users, isLoading: usersLoading } = useWorkspaceUsers(canManage ? workspaceId : undefined)

  if (permsLoading) {
    return <div className="p-10"><PageSpinner /></div>
  }
  if (!canManage) {
    return (
      <div className="max-w-2xl mx-auto px-6 py-10">
        <h1 className="text-xl font-bold text-ink-1 mb-2">User Management</h1>
        <p className="text-sm text-ink-4">
          You don't have the <code className="text-ink-2 bg-ink-5/30 px-1 rounded">user.manage</code> permission for this workspace.
          Ask the workspace owner or admin to grant it.
        </p>
      </div>
    )
  }

  return (
    <div className="max-w-4xl mx-auto px-6 py-10 animate-slide-up">
      <h1 className="text-xl font-bold text-ink-1 mb-2">User Management</h1>
      <p className="text-sm text-ink-4 mb-6">
        Create, edit, reset passwords, or delete user accounts. Use the Members page if you only
        want to add an already-registered user to this workspace.
      </p>

      <CreateUserCard workspaceId={workspaceId!} roles={roles ?? []} />

      <div className="bg-surface border border-ink-5/30 rounded-2xl shadow-card mt-6">
        {usersLoading && <div className="p-5"><PageSpinner /></div>}
        {!usersLoading && (users ?? []).length === 0 && (
          <p className="p-6 text-sm text-ink-4">No users yet.</p>
        )}
        <table className="w-full">
          <thead className="text-[11px] uppercase tracking-wide text-ink-4">
            <tr className="border-b border-ink-5/20">
              <th className="text-left py-2.5 px-5 font-semibold">User</th>
              <th className="text-left py-2.5 px-2 font-semibold">Role</th>
              <th className="text-left py-2.5 px-2 font-semibold hidden sm:table-cell">Joined</th>
              <th className="text-right py-2.5 px-5 font-semibold">Actions</th>
            </tr>
          </thead>
          <tbody>
            {(users ?? []).map((u) => (
              <UserRow key={u.id} workspaceId={workspaceId!} user={u} />
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}

// CreateUserCard — top-of-page form. Fields are kept inline (not a modal)
// so admins can rapid-fire-create users; the form clears on success.
function CreateUserCard({ workspaceId, roles }: { workspaceId: string; roles: Role[] }) {
  const [email, setEmail] = useState('')
  const [name, setName] = useState('')
  const [password, setPassword] = useState('')
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

  const create = useCreateUser(workspaceId)

  const submit = () => {
    if (!email.trim() || password.length < 8) return
    create.mutate(
      {
        email: email.trim().toLowerCase(),
        name: name.trim(),
        password,
        role,
      },
      {
        onSuccess: () => {
          setEmail('')
          setName('')
          setPassword('')
        },
      },
    )
  }

  const errMsg = create.error
    ? (create.error as Error & { message: string }).message
    : null

  return (
    <div className="bg-surface border border-ink-5/30 rounded-2xl p-5 shadow-card">
      <p className="text-xs font-semibold text-ink-2 mb-3">Create user</p>
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
        <Input
          type="email"
          placeholder="email@company.com"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
        />
        <Input
          type="text"
          placeholder="Full name"
          value={name}
          onChange={(e) => setName(e.target.value)}
        />
        <Input
          type="password"
          placeholder="Initial password (≥ 8 chars)"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
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
      <div className="flex items-center justify-between gap-3 mt-3">
        <p className="text-[11px] text-ink-4 leading-snug">
          The new user logs in immediately with the email + password you set, and is added to this workspace.
        </p>
        <Button
          onClick={submit}
          disabled={!email.trim() || password.length < 8 || create.isPending}
          loading={create.isPending}
          size="sm"
        >
          Create user
        </Button>
      </div>
      {errMsg && <p className="text-red-500 text-xs mt-2">{errMsg}</p>}
    </div>
  )
}

// UserRow — one row in the table. Inline-edit name, reset password
// (modal-ish dialog with browser prompt), delete with confirm.
function UserRow({
  workspaceId,
  user,
}: {
  workspaceId: string
  user: import('../../hooks/useUsers').WorkspaceUser
}) {
  const [editing, setEditing] = useState(false)
  const [draftName, setDraftName] = useState(user.name)

  const update = useUpdateUser(workspaceId)
  const resetPassword = useResetUserPassword(workspaceId)
  const del = useDeleteUser(workspaceId)

  const saveName = () => {
    const next = draftName.trim()
    if (!next || next === user.name) {
      setEditing(false)
      setDraftName(user.name)
      return
    }
    update.mutate(
      { userId: user.id, name: next },
      {
        onSuccess: () => setEditing(false),
      },
    )
  }

  const onResetPassword = () => {
    // Lightweight UX: prompt the admin for a new password. A modal would
    // be nicer; the prompt is sufficient and matches the existing
    // "remove member" confirm flow elsewhere on the page.
    const pw = window.prompt(`Set a new password for ${user.email}\n(min 8 chars)`)
    if (pw === null) return
    if (pw.length < 8) {
      window.alert('Password must be at least 8 characters.')
      return
    }
    resetPassword.mutate(
      { userId: user.id, password: pw },
      {
        onSuccess: () => window.alert(`Password reset. Share it with ${user.email}.`),
        onError: (e) => window.alert(`Reset failed: ${(e as Error).message}`),
      },
    )
  }

  const onDelete = () => {
    if (user.is_owner) {
      window.alert(`${user.email} owns this workspace and can't be deleted from here.`)
      return
    }
    if (!window.confirm(`Permanently delete ${user.email}? This cannot be undone.`)) return
    del.mutate(user.id)
  }

  return (
    <tr className="border-b border-ink-5/20 last:border-b-0 hover:bg-ink-5/10">
      <td className="py-3 px-5">
        <div className="flex items-center gap-3">
          <div className="w-8 h-8 rounded-full bg-brand-gradient text-xs font-bold text-white flex items-center justify-center shrink-0">
            {(user.name || user.email || '?').slice(0, 1).toUpperCase()}
          </div>
          <div className="min-w-0">
            {editing ? (
              <div className="flex items-center gap-2">
                <input
                  className="h-7 px-2 bg-surface border border-ink-5/40 rounded text-sm text-ink-1 focus:outline-none focus:ring-2 focus:ring-brand-400/40"
                  value={draftName}
                  onChange={(e) => setDraftName(e.target.value)}
                  onKeyDown={(e) => { if (e.key === 'Enter') saveName(); if (e.key === 'Escape') { setEditing(false); setDraftName(user.name) } }}
                  autoFocus
                />
                <button onClick={saveName} className="text-xs text-brand-600 hover:text-brand-700">Save</button>
                <button onClick={() => { setEditing(false); setDraftName(user.name) }} className="text-xs text-ink-4 hover:text-ink-2">Cancel</button>
              </div>
            ) : (
              <p className="text-sm font-medium text-ink-1 truncate">{user.name || <span className="text-ink-4 italic">no name</span>}</p>
            )}
            <p className="text-xs text-ink-4 truncate">{user.email}</p>
          </div>
        </div>
      </td>
      <td className="py-3 px-2">
        <span className="text-[11px] font-semibold bg-ink-5/30 text-ink-2 px-2 py-0.5 rounded-full capitalize">
          {user.role}
        </span>
      </td>
      <td className="py-3 px-2 text-[11px] text-ink-4 hidden sm:table-cell">{formatRelative(user.joined_at)}</td>
      <td className="py-3 px-5 text-right">
        <div className="inline-flex items-center gap-2">
          <button
            onClick={() => setEditing(true)}
            className="text-xs text-ink-3 hover:text-ink-1 px-2 py-1 rounded hover:bg-ink-5/30"
            disabled={editing}
          >
            Edit
          </button>
          <button
            onClick={onResetPassword}
            className="text-xs text-ink-3 hover:text-ink-1 px-2 py-1 rounded hover:bg-ink-5/30"
            disabled={resetPassword.isPending}
          >
            Reset password
          </button>
          {!user.is_owner && (
            <button
              onClick={onDelete}
              className="text-xs text-red-500 hover:text-red-600 px-2 py-1 rounded hover:bg-red-50"
              disabled={del.isPending}
            >
              Delete
            </button>
          )}
        </div>
      </td>
    </tr>
  )
}
