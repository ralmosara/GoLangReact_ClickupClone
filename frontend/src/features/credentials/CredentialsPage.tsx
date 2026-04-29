import { useEffect, useMemo, useState } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import { Link, useParams } from 'react-router-dom'
import {
  ChevronLeft,
  Copy as CopyIcon,
  Eye,
  EyeOff,
  KeyRound,
  Pencil,
  Plus,
  Search,
  Trash2,
} from 'lucide-react'

import { api } from '../../lib/api'
import { queryClient } from '../../lib/queryClient'
import { Button, Input, Modal, PageSpinner } from '../../components/ui'
import { cn, formatRelative } from '../../lib/utils'
import type { Credential } from '../../types'

const REVEAL_TIMEOUT_MS = 30_000

export function CredentialsPage() {
  const { workspaceId } = useParams<{ workspaceId: string }>()
  const [editing, setEditing] = useState<Credential | 'new' | null>(null)
  const [confirmDelete, setConfirmDelete] = useState<Credential | null>(null)
  const [filter, setFilter] = useState('')

  const { data: credentials, isLoading } = useQuery({
    queryKey: ['credentials', workspaceId],
    queryFn: () =>
      api.get(`workspaces/${workspaceId}/credentials`).json<Credential[] | null>(),
    enabled: !!workspaceId,
  })
  const items = credentials ?? []

  const filtered = useMemo(() => {
    const q = filter.trim().toLowerCase()
    if (!q) return items
    return items.filter((c) =>
      [c.name, c.url, c.username].some((f) => f.toLowerCase().includes(q)),
    )
  }, [items, filter])

  return (
    <div className="min-h-screen bg-canvas">
      <div className="max-w-4xl mx-auto px-6 py-10 animate-slide-up">
        <Link
          to={workspaceId ? `/workspaces/${workspaceId}` : '/workspaces'}
          className="inline-flex items-center gap-1 text-xs text-ink-4 hover:text-ink-2 mb-4 transition-colors"
        >
          <ChevronLeft className="w-3.5 h-3.5" />
          Back to workspace
        </Link>

        <div className="flex items-start justify-between gap-4 mb-8">
          <div className="flex items-start gap-3">
            <div className="w-11 h-11 rounded-xl bg-brand-50 border border-brand-100 flex items-center justify-center shrink-0">
              <KeyRound className="w-5 h-5 text-brand-500" />
            </div>
            <div>
              <h1 className="text-xl font-bold text-ink-1">Credentials</h1>
              <p className="text-sm text-ink-4 mt-0.5">
                Encrypted password vault — entries are scoped to this workspace and only visible to you.
              </p>
            </div>
          </div>
          <Button onClick={() => setEditing('new')}>
            <Plus className="w-4 h-4" />
            New credential
          </Button>
        </div>

        {items.length > 0 && (
          <div className="mb-4">
            <Input
              prefixNode={<Search className="w-3.5 h-3.5" />}
              placeholder="Search by name, URL, or username"
              value={filter}
              onChange={(e) => setFilter(e.target.value)}
            />
          </div>
        )}

        {isLoading && <PageSpinner />}

        {!isLoading && items.length === 0 && (
          <div className="text-center py-20 bg-surface/50 border border-ink-5/20 rounded-2xl">
            <div className="w-14 h-14 rounded-2xl bg-brand-50 border border-brand-100 flex items-center justify-center mx-auto mb-4">
              <KeyRound className="w-7 h-7 text-brand-400" />
            </div>
            <p className="font-semibold text-ink-2">No credentials yet</p>
            <p className="text-sm text-ink-4 mt-1 mb-5">
              Save logins for the servers and websites you use.
            </p>
            <Button onClick={() => setEditing('new')}>
              <Plus className="w-4 h-4" />
              Add your first credential
            </Button>
          </div>
        )}

        {!isLoading && items.length > 0 && filtered.length === 0 && (
          <p className="text-sm text-ink-4 text-center py-12">
            No matches for &ldquo;{filter}&rdquo;.
          </p>
        )}

        {!isLoading && filtered.length > 0 && workspaceId && (
          <div className="space-y-2 animate-fade-in">
            {filtered.map((c) => (
              <CredentialRow
                key={c.id}
                workspaceId={workspaceId}
                credential={c}
                onEdit={() => setEditing(c)}
                onDelete={() => setConfirmDelete(c)}
              />
            ))}
          </div>
        )}
      </div>

      {editing && workspaceId && (
        <CredentialFormModal
          workspaceId={workspaceId}
          existing={editing === 'new' ? null : editing}
          onClose={() => setEditing(null)}
        />
      )}

      {confirmDelete && workspaceId && (
        <DeleteConfirmModal
          workspaceId={workspaceId}
          credential={confirmDelete}
          onClose={() => setConfirmDelete(null)}
        />
      )}
    </div>
  )
}

/* -------------------------------------------------------------------------- */
/* Row with reveal/copy                                                        */
/* -------------------------------------------------------------------------- */

function CredentialRow({
  workspaceId,
  credential,
  onEdit,
  onDelete,
}: {
  workspaceId: string
  credential: Credential
  onEdit: () => void
  onDelete: () => void
}) {
  const [revealed, setRevealed] = useState<string | null>(null)
  const [revealing, setRevealing] = useState(false)
  const [revealError, setRevealError] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)

  // Auto-hide the password after the timeout to limit shoulder-surfing exposure.
  useEffect(() => {
    if (!revealed) return
    const t = window.setTimeout(() => setRevealed(null), REVEAL_TIMEOUT_MS)
    return () => window.clearTimeout(t)
  }, [revealed])

  const handleReveal = async () => {
    if (revealed) {
      setRevealed(null)
      return
    }
    setRevealing(true)
    setRevealError(null)
    try {
      const full = await api
        .get(`workspaces/${workspaceId}/credentials/${credential.id}`)
        .json<Credential>()
      setRevealed(full.password ?? '')
    } catch (e) {
      setRevealError(e instanceof Error ? e.message : 'Failed to load password')
    } finally {
      setRevealing(false)
    }
  }

  const handleCopy = async () => {
    let pw = revealed
    if (!pw) {
      try {
        const full = await api
          .get(`workspaces/${workspaceId}/credentials/${credential.id}`)
          .json<Credential>()
        pw = full.password ?? ''
      } catch (e) {
        setRevealError(e instanceof Error ? e.message : 'Failed to load password')
        return
      }
    }
    try {
      await navigator.clipboard.writeText(pw)
      setCopied(true)
      window.setTimeout(() => setCopied(false), 1500)
    } catch {
      setRevealError('Clipboard unavailable')
    }
  }

  const host = hostFromUrl(credential.url)

  return (
    <div className="bg-surface border border-ink-5/20 rounded-xl p-4 shadow-card group">
      <div className="flex items-start gap-4">
        <div className="w-9 h-9 rounded-lg bg-brand-50 border border-brand-100 flex items-center justify-center shrink-0">
          <KeyRound className="w-4 h-4 text-brand-500" />
        </div>

        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 flex-wrap">
            <p className="text-sm font-semibold text-ink-1 truncate">{credential.name}</p>
            {host && (
              <a
                href={credential.url}
                target="_blank"
                rel="noreferrer noopener"
                className="text-[11px] text-brand-600 hover:underline truncate"
              >
                {host}
              </a>
            )}
          </div>
          {credential.username && (
            <p className="text-xs text-ink-3 mt-0.5 truncate">{credential.username}</p>
          )}

          <div className="mt-2 flex items-center gap-2 flex-wrap">
            <code
              className={cn(
                'inline-flex items-center h-7 px-2 rounded-md font-mono text-xs',
                'bg-canvas border border-ink-5/30',
                revealed ? 'text-ink-1' : 'text-ink-4 tracking-widest',
              )}
            >
              {revealed ?? '••••••••••'}
            </code>
            <button
              onClick={handleReveal}
              disabled={revealing}
              className="inline-flex items-center gap-1 h-7 px-2 rounded-md text-[11px] font-medium text-ink-2 hover:bg-ink-1/5 disabled:opacity-50 transition-colors"
              title={revealed ? 'Hide password' : 'Reveal password'}
            >
              {revealed ? <EyeOff className="w-3.5 h-3.5" /> : <Eye className="w-3.5 h-3.5" />}
              {revealed ? 'Hide' : 'Reveal'}
            </button>
            <button
              onClick={handleCopy}
              className="inline-flex items-center gap-1 h-7 px-2 rounded-md text-[11px] font-medium text-ink-2 hover:bg-ink-1/5 transition-colors"
              title="Copy password"
            >
              <CopyIcon className="w-3.5 h-3.5" />
              {copied ? 'Copied!' : 'Copy'}
            </button>
          </div>

          {revealError && (
            <p className="text-[11px] text-red-500 mt-1.5">{revealError}</p>
          )}
          {credential.notes && (
            <p className="text-[11px] text-ink-4 mt-2 whitespace-pre-wrap line-clamp-2">
              {credential.notes}
            </p>
          )}
          <p className="text-[10px] text-ink-4 mt-2">
            Updated {formatRelative(credential.updated_at)}
          </p>
        </div>

        <div className="flex flex-col gap-1 shrink-0 opacity-0 group-hover:opacity-100 transition-opacity">
          <button
            onClick={onEdit}
            className="p-1.5 text-ink-3 hover:text-ink-1 hover:bg-ink-1/5 rounded transition-colors"
            title="Edit"
            aria-label="Edit credential"
          >
            <Pencil className="w-3.5 h-3.5" />
          </button>
          <button
            onClick={onDelete}
            className="p-1.5 text-ink-3 hover:text-red-600 hover:bg-red-50 rounded transition-colors"
            title="Delete"
            aria-label="Delete credential"
          >
            <Trash2 className="w-3.5 h-3.5" />
          </button>
        </div>
      </div>
    </div>
  )
}

/* -------------------------------------------------------------------------- */
/* Create / edit modal                                                         */
/* -------------------------------------------------------------------------- */

function CredentialFormModal({
  workspaceId,
  existing,
  onClose,
}: {
  workspaceId: string
  existing: Credential | null
  onClose: () => void
}) {
  const isEdit = !!existing
  const [name, setName] = useState(existing?.name ?? '')
  const [url, setUrl] = useState(existing?.url ?? '')
  const [username, setUsername] = useState(existing?.username ?? '')
  const [password, setPassword] = useState('')
  const [notes, setNotes] = useState(existing?.notes ?? '')
  const [showPassword, setShowPassword] = useState(false)
  const [loadingExisting, setLoadingExisting] = useState(isEdit)

  // For edits, fetch the decrypted password so the user can see what's stored
  // (and choose whether to keep it or replace it).
  useEffect(() => {
    if (!existing) return
    let cancelled = false
    api
      .get(`workspaces/${workspaceId}/credentials/${existing.id}`)
      .json<Credential>()
      .then((full) => {
        if (!cancelled && full.password) setPassword(full.password)
      })
      .finally(() => {
        if (!cancelled) setLoadingExisting(false)
      })
    return () => {
      cancelled = true
    }
  }, [existing, workspaceId])

  const mut = useMutation({
    mutationFn: async () => {
      const body = { name, url, username, password, notes }
      if (isEdit && existing) {
        return api
          .patch(`workspaces/${workspaceId}/credentials/${existing.id}`, { json: body })
          .json<Credential>()
      }
      return api
        .post(`workspaces/${workspaceId}/credentials`, { json: body })
        .json<Credential>()
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['credentials', workspaceId] })
      onClose()
    },
  })

  const canSubmit =
    name.trim().length > 0 && password.length > 0 && !mut.isPending && !loadingExisting

  return (
    <Modal
      open
      onClose={onClose}
      title={isEdit ? 'Edit credential' : 'New credential'}
      description={
        isEdit
          ? 'Update the stored entry. Leaving the password unchanged keeps the existing value.'
          : 'Saved entries are encrypted with AES-256-GCM before they touch the database.'
      }
      width="max-w-lg"
    >
      <div className="space-y-4">
        <Input
          label="Name"
          placeholder="e.g. GitHub"
          value={name}
          onChange={(e) => setName(e.target.value)}
          autoFocus
        />
        <Input
          label="URL"
          placeholder="https://example.com"
          value={url}
          onChange={(e) => setUrl(e.target.value)}
        />
        <Input
          label="Username or email"
          placeholder="you@example.com"
          value={username}
          onChange={(e) => setUsername(e.target.value)}
          autoComplete="off"
        />

        <div>
          <label className="block text-xs font-semibold text-ink-2 mb-1.5 tracking-wide">
            Password
          </label>
          <div className="relative">
            <input
              type={showPassword ? 'text' : 'password'}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              autoComplete="new-password"
              placeholder={loadingExisting ? 'Loading…' : '••••••••'}
              className={cn(
                'w-full h-9 bg-white border rounded text-sm text-ink-1 transition-all duration-150',
                'placeholder:text-ink-4/70 pl-3 pr-10',
                'focus:outline-none focus:ring-2 focus:ring-brand-500/20 focus:border-brand-500',
                'border-ink-5/80 hover:border-ink-4/60',
              )}
            />
            <button
              type="button"
              onClick={() => setShowPassword((v) => !v)}
              className="absolute right-2 top-1/2 -translate-y-1/2 p-1 text-ink-4 hover:text-ink-2 rounded transition-colors"
              title={showPassword ? 'Hide' : 'Show'}
              aria-label={showPassword ? 'Hide password' : 'Show password'}
            >
              {showPassword ? <EyeOff className="w-3.5 h-3.5" /> : <Eye className="w-3.5 h-3.5" />}
            </button>
          </div>
        </div>

        <div>
          <label className="block text-xs font-semibold text-ink-2 mb-1.5 tracking-wide">
            Notes
          </label>
          <textarea
            value={notes}
            onChange={(e) => setNotes(e.target.value)}
            rows={3}
            placeholder="Recovery codes, hints, anything else…"
            className={cn(
              'w-full bg-white border rounded text-sm text-ink-1 transition-all duration-150 px-3 py-2',
              'placeholder:text-ink-4/70',
              'focus:outline-none focus:ring-2 focus:ring-brand-500/20 focus:border-brand-500',
              'border-ink-5/80 hover:border-ink-4/60',
            )}
          />
        </div>

        {mut.error && (
          <p className="text-red-500 text-xs">{String((mut.error as Error).message ?? mut.error)}</p>
        )}

        <div className="flex justify-end gap-2 pt-1">
          <Button variant="secondary" onClick={onClose}>
            Cancel
          </Button>
          <Button onClick={() => mut.mutate()} disabled={!canSubmit} loading={mut.isPending}>
            {isEdit ? 'Save changes' : 'Create credential'}
          </Button>
        </div>
      </div>
    </Modal>
  )
}

/* -------------------------------------------------------------------------- */
/* Delete confirm                                                              */
/* -------------------------------------------------------------------------- */

function DeleteConfirmModal({
  workspaceId,
  credential,
  onClose,
}: {
  workspaceId: string
  credential: Credential
  onClose: () => void
}) {
  const mut = useMutation({
    mutationFn: () =>
      api.delete(`workspaces/${workspaceId}/credentials/${credential.id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['credentials', workspaceId] })
      onClose()
    },
  })

  return (
    <Modal
      open
      onClose={onClose}
      title="Delete credential"
      description="This permanently removes the entry. There's no undo."
    >
      <div className="space-y-4">
        <p className="text-sm text-ink-2">
          Delete <span className="font-semibold text-ink-1">{credential.name}</span>?
        </p>
        {mut.error && (
          <p className="text-red-500 text-xs">{String((mut.error as Error).message ?? mut.error)}</p>
        )}
        <div className="flex justify-end gap-2 pt-1">
          <Button variant="secondary" onClick={onClose}>
            Cancel
          </Button>
          <Button variant="danger" onClick={() => mut.mutate()} loading={mut.isPending}>
            Delete
          </Button>
        </div>
      </div>
    </Modal>
  )
}

/* -------------------------------------------------------------------------- */

function hostFromUrl(raw: string): string | null {
  if (!raw) return null
  try {
    return new URL(raw).hostname
  } catch {
    return null
  }
}
