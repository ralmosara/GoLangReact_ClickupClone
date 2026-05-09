// GlobalUserManagementPage — global, workspace-agnostic User Management.
//
// Reachable from the WorkspacesPage's "Manage Users" nav button. Lets an
// admin (anyone with user.manage on at least one workspace) create user
// accounts that belong to NO workspace yet. Users created here can log
// in immediately; their /workspaces list will be empty until an admin
// invites them via the workspace-scoped Members or User Management page.
//
// Auth gate: the page itself optimistically renders the create form;
// 403 from /admin/users (the underlying list query) flips into a
// "no permission" placeholder.
import { useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { ArrowLeft, LogOut } from 'lucide-react'
import {
  useAllUsers,
  useGlobalCreateUser,
  useGlobalDeleteUser,
  useGlobalResetPassword,
  useGlobalUpdateUser,
} from '../../hooks/useUsers'
import { useAuthStore } from '../../store/auth'
import { useLogout } from '../../hooks/useAuth'
import { Button, Input, PageSpinner } from '../../components/ui'
import { formatRelative } from '../../lib/utils'
import type { User } from '../../types'

export function GlobalUserManagementPage() {
  const navigate = useNavigate()
  const logout = useLogout()
  const me = useAuthStore((s) => s.user)

  const list = useAllUsers()

  // Forbidden detection — ky throws on non-2xx, with the response on the
  // error. We surface a friendly placeholder rather than the generic
  // mutation error string.
  const isForbidden = !!list.error && /403/.test(String(list.error))

  return (
    <div className="min-h-screen bg-canvas-gradient flex flex-col">
      <header className="px-8 py-5 flex items-center justify-between border-b border-ink-5/20 bg-surface/60 backdrop-blur-sm">
        <div className="flex items-center gap-3">
          <button
            onClick={() => navigate('/workspaces')}
            className="inline-flex items-center justify-center h-8 w-8 rounded-lg text-ink-3 hover:text-ink-1 hover:bg-ink-5/30 transition-colors"
            title="Back to workspaces"
            aria-label="Back to workspaces"
          >
            <ArrowLeft className="w-4 h-4" />
          </button>
          <span className="text-ink-1 font-semibold text-sm tracking-wide">User Management</span>
        </div>
        <button
          onClick={logout}
          title={me ? `Sign out (${me.email})` : 'Sign out'}
          aria-label="Sign out"
          className="inline-flex items-center justify-center h-8 w-8 rounded-lg text-ink-3 hover:text-red-500 hover:bg-red-50 transition-colors"
        >
          <LogOut className="w-4 h-4" />
        </button>
      </header>

      <main className="flex-1 px-8 py-12 max-w-4xl mx-auto w-full">
        <div className="mb-8 animate-slide-up">
          <h1 className="text-2xl font-bold text-ink-1 mb-1">Users</h1>
          <p className="text-sm text-ink-3">
            Create and manage user accounts across the system. New accounts can be added to a workspace later via{' '}
            <Link to="/workspaces" className="text-brand-600 hover:text-brand-700 font-medium">Members</Link>.
          </p>
        </div>

        {isForbidden && (
          <div className="bg-surface border border-ink-5/30 rounded-2xl p-8 shadow-card text-center">
            <p className="text-ink-2 font-semibold">Forbidden</p>
            <p className="text-sm text-ink-4 mt-1">
              You don't have <code className="text-ink-2 bg-ink-5/30 px-1 rounded">user.manage</code> on any workspace.
              Ask an owner to grant it.
            </p>
          </div>
        )}

        {!isForbidden && (
          <>
            <CreateUserCard />
            <UserTable users={list.data?.users ?? []} loading={list.isLoading} myId={me?.id} />
            {list.data && (
              <p className="text-[11px] text-ink-4 text-right mt-3">
                Showing {list.data.users.length} of {list.data.total}
              </p>
            )}
          </>
        )}
      </main>
    </div>
  )
}

// CreateUserCard — workspace-agnostic create. Just email + name + password
// (no role dropdown — there's no workspace to set the role on).
function CreateUserCard() {
  const [email, setEmail] = useState('')
  const [name, setName] = useState('')
  const [password, setPassword] = useState('')

  const create = useGlobalCreateUser()

  const submit = () => {
    if (!email.trim() || password.length < 8) return
    create.mutate(
      { email: email.trim().toLowerCase(), name: name.trim(), password },
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
    <div className="bg-surface border border-ink-5/30 rounded-2xl p-5 shadow-card mb-5">
      <p className="text-xs font-semibold text-ink-2 mb-3">Create user</p>
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-2">
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
          placeholder="Password (≥ 8 chars)"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
        />
      </div>
      <div className="flex items-center justify-between gap-3 mt-3">
        <p className="text-[11px] text-ink-4 leading-snug">
          Account is created without a workspace. The user can log in immediately;
          invite them to a workspace from Members when ready.
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

function UserTable({ users, loading, myId }: { users: User[]; loading: boolean; myId?: string }) {
  return (
    <div className="bg-surface border border-ink-5/30 rounded-2xl shadow-card">
      {loading && <div className="p-5"><PageSpinner /></div>}
      {!loading && users.length === 0 && (
        <p className="p-6 text-sm text-ink-4">No users yet.</p>
      )}
      {users.length > 0 && (
        <table className="w-full">
          <thead className="text-[11px] uppercase tracking-wide text-ink-4">
            <tr className="border-b border-ink-5/20">
              <th className="text-left py-2.5 px-5 font-semibold">User</th>
              <th className="text-left py-2.5 px-2 font-semibold hidden sm:table-cell">Created</th>
              <th className="text-right py-2.5 px-5 font-semibold">Actions</th>
            </tr>
          </thead>
          <tbody>
            {users.map((u) => (
              <UserRow key={u.id} user={u} isMe={u.id === myId} />
            ))}
          </tbody>
        </table>
      )}
    </div>
  )
}

function UserRow({ user, isMe }: { user: User; isMe: boolean }) {
  const [editing, setEditing] = useState(false)
  const [draftName, setDraftName] = useState(user.name)

  // Re-sync the draft when the underlying user prop changes (e.g.
  // after another row's invalidation re-fetches the list).
  useEffect(() => { setDraftName(user.name) }, [user.name])

  const update = useGlobalUpdateUser()
  const resetPassword = useGlobalResetPassword()
  const del = useGlobalDeleteUser()

  const saveName = () => {
    const next = draftName.trim()
    if (!next || next === user.name) {
      setEditing(false)
      setDraftName(user.name)
      return
    }
    update.mutate(
      { userId: user.id, name: next },
      { onSuccess: () => setEditing(false) },
    )
  }

  const onResetPassword = () => {
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
    if (isMe) {
      window.alert("You can't delete your own account.")
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
              <p className="text-sm font-medium text-ink-1 truncate">
                {user.name || <span className="text-ink-4 italic">no name</span>}
                {isMe && <span className="ml-2 text-[10px] font-semibold text-brand-600 bg-brand-50 px-1.5 py-0.5 rounded-full">you</span>}
              </p>
            )}
            <p className="text-xs text-ink-4 truncate">{user.email}</p>
          </div>
        </div>
      </td>
      <td className="py-3 px-2 text-[11px] text-ink-4 hidden sm:table-cell">{formatRelative(user.created_at)}</td>
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
          {!isMe && (
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
