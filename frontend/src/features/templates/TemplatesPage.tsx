import { useMemo, useState } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import { useParams } from 'react-router-dom'
import { Copy, Plus, Trash2 } from 'lucide-react'

import { api } from '../../lib/api'
import { queryClient } from '../../lib/queryClient'
import { Button, Input, Modal, PageSpinner } from '../../components/ui'
import { cn, formatRelative } from '../../lib/utils'
import type { List, Space, Template, TemplateKind } from '../../types'

const KINDS: Array<{ value: TemplateKind; label: string; hint: string }> = [
  { value: 'list', label: 'List',  hint: 'statuses + seed tasks' },
  { value: 'doc',  label: 'Doc',   hint: 'title + content + nested pages' },
  { value: 'task', label: 'Task',  hint: 'name + description + subtasks' },
]

export function TemplatesPage() {
  const { workspaceId } = useParams<{ workspaceId: string }>()
  const [kind, setKind] = useState<TemplateKind | ''>('')
  const [showCreate, setShowCreate] = useState(false)
  const [applying, setApplying] = useState<Template | null>(null)

  const { data: templates = [], isLoading } = useQuery({
    queryKey: ['templates', workspaceId, kind],
    queryFn: () =>
      api.get(`workspaces/${workspaceId}/templates${kind ? `?kind=${kind}` : ''}`).json<Template[]>(),
    enabled: !!workspaceId,
  })

  const del = useMutation({
    mutationFn: (id: string) => api.delete(`templates/${id}`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['templates', workspaceId] }),
  })

  return (
    <div className="max-w-4xl mx-auto px-6 py-10 animate-slide-up">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-xl font-bold text-ink-1 flex items-center gap-2">
            <Copy className="w-5 h-5 text-brand-500" />
            Templates
          </h1>
          <p className="text-sm text-ink-4 mt-0.5">Reusable list, doc, and task structures.</p>
        </div>
        <Button onClick={() => setShowCreate(true)} size="sm">
          <Plus className="w-3.5 h-3.5 mr-1.5" />
          New template
        </Button>
      </div>

      <div className="flex items-center gap-2 mb-4">
        <button
          onClick={() => setKind('')}
          className={cn(
            'h-7 px-3 rounded-lg text-xs font-medium border transition-colors',
            kind === '' ? 'bg-brand-500 text-white border-brand-500' : 'bg-surface border-ink-5/30 text-ink-2 hover:border-brand-300',
          )}
        >
          All
        </button>
        {KINDS.map((k) => (
          <button
            key={k.value}
            onClick={() => setKind(k.value)}
            className={cn(
              'h-7 px-3 rounded-lg text-xs font-medium border transition-colors',
              kind === k.value ? 'bg-brand-500 text-white border-brand-500' : 'bg-surface border-ink-5/30 text-ink-2 hover:border-brand-300',
            )}
          >
            {k.label}
          </button>
        ))}
      </div>

      {isLoading && <PageSpinner />}

      {!isLoading && templates.length === 0 && (
        <div className="text-center py-14 bg-surface/50 border border-ink-5/20 rounded-2xl text-sm text-ink-4">
          No templates yet. Create one to reuse structure across projects.
        </div>
      )}

      {templates.length > 0 && (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
          {templates.map((t) => (
            <div key={t.id} className="bg-surface border border-ink-5/30 rounded-2xl shadow-card p-4 group">
              <div className="flex items-center justify-between mb-1.5">
                <span className="text-[10px] font-bold uppercase tracking-wider text-brand-600 bg-brand-50 border border-brand-100 rounded px-1.5 py-0.5">
                  {t.kind}
                </span>
                <span className="text-[10px] text-ink-4">{formatRelative(t.created_at)}</span>
              </div>
              <p className="text-sm font-semibold text-ink-1 truncate mb-1">{t.name}</p>
              {t.description && <p className="text-xs text-ink-4 mb-2 line-clamp-2">{t.description}</p>}
              <div className="flex items-center justify-end gap-2 mt-3">
                <button
                  onClick={() => { if (confirm(`Delete template "${t.name}"?`)) del.mutate(t.id) }}
                  className="opacity-0 group-hover:opacity-100 text-ink-4 hover:text-red-500 p-1 rounded transition-all"
                >
                  <Trash2 className="w-3.5 h-3.5" />
                </button>
                <Button size="sm" onClick={() => setApplying(t)}>Apply</Button>
              </div>
            </div>
          ))}
        </div>
      )}

      {showCreate && workspaceId && (
        <CreateTemplateModal workspaceId={workspaceId} onClose={() => setShowCreate(false)} />
      )}
      {applying && workspaceId && (
        <ApplyTemplateModal
          template={applying}
          workspaceId={workspaceId}
          onClose={() => setApplying(null)}
        />
      )}
    </div>
  )
}

function CreateTemplateModal({
  workspaceId, onClose,
}: {
  workspaceId: string
  onClose: () => void
}) {
  const [kind, setKind] = useState<TemplateKind>('list')
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [snapshot, setSnapshot] = useState(() => JSON.stringify(defaultSnapshot('list'), null, 2))
  const [error, setError] = useState<string | null>(null)

  const create = useMutation({
    mutationFn: () => {
      let parsed: unknown
      try { parsed = JSON.parse(snapshot) }
      catch { throw new Error('Snapshot must be valid JSON') }
      return api.post(`workspaces/${workspaceId}/templates`, {
        json: { kind, name, description, snapshot: parsed },
      }).json<Template>()
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['templates', workspaceId] })
      onClose()
    },
    onError: (e) => setError(e instanceof Error ? e.message : String(e)),
  })

  return (
    <Modal open title="New template" description="A reusable structure you can apply later." onClose={onClose} width="max-w-lg">
      <div className="space-y-3">
        <div>
          <label className="block text-xs font-medium text-ink-2 mb-1">Kind</label>
          <div className="grid grid-cols-3 gap-2">
            {KINDS.map((k) => (
              <button
                key={k.value}
                onClick={() => {
                  setKind(k.value)
                  setSnapshot(JSON.stringify(defaultSnapshot(k.value), null, 2))
                }}
                className={cn(
                  'text-left px-3 py-2 rounded-lg border transition-all',
                  kind === k.value ? 'border-brand-400 bg-brand-50' : 'border-ink-5/30 bg-surface hover:border-ink-5/60',
                )}
              >
                <p className="text-xs font-semibold text-ink-1">{k.label}</p>
                <p className="text-[10px] text-ink-4 mt-0.5">{k.hint}</p>
              </button>
            ))}
          </div>
        </div>
        <Input label="Name" value={name} onChange={(e) => setName(e.target.value)} placeholder="Feature launch" autoFocus />
        <div>
          <label className="block text-xs font-medium text-ink-2 mb-1">Description</label>
          <textarea
            className="w-full bg-surface border border-ink-5/40 rounded-xl px-3 py-2.5 text-sm h-16 resize-none focus:outline-none focus:ring-2 focus:ring-brand-400/40"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
          />
        </div>
        <div>
          <label className="block text-xs font-medium text-ink-2 mb-1">Snapshot (JSON)</label>
          <textarea
            className="w-full bg-canvas/60 border border-ink-5/30 rounded-xl px-3 py-2 text-[11px] font-mono h-48 resize-none focus:outline-none focus:ring-2 focus:ring-brand-400/40"
            value={snapshot}
            onChange={(e) => setSnapshot(e.target.value)}
          />
        </div>
        {error && <p className="text-xs text-red-600">{error}</p>}
        <div className="flex justify-end gap-2 pt-1">
          <Button variant="secondary" onClick={onClose}>Cancel</Button>
          <Button onClick={() => create.mutate()} disabled={!name.trim() || create.isPending} loading={create.isPending}>Create</Button>
        </div>
      </div>
    </Modal>
  )
}

function ApplyTemplateModal({
  template, workspaceId, onClose,
}: {
  template: Template
  workspaceId: string
  onClose: () => void
}) {
  const [spaceID, setSpaceID] = useState('')
  const [listID, setListID] = useState('')
  const [error, setError] = useState<string | null>(null)

  const { data: spaces = [] } = useQuery({
    queryKey: ['spaces', workspaceId],
    queryFn: () => api.get(`workspaces/${workspaceId}/spaces`).json<Space[]>(),
  })
  const listsBySpace = useTemplateLists(spaces)
  const allLists = useMemo(() => spaces.flatMap((s) => listsBySpace[s.id] ?? []), [spaces, listsBySpace])

  const apply = useMutation({
    mutationFn: () => {
      const target: Record<string, string> = {}
      if (template.kind === 'list') target.space_id = spaceID
      if (template.kind === 'task') target.list_id = listID
      if (template.kind === 'doc')  target.workspace_id = workspaceId
      return api.post(`templates/${template.id}/apply`, { json: target }).json<Record<string, unknown>>()
    },
    onSuccess: () => onClose(),
    onError: (e) => setError(e instanceof Error ? e.message : String(e)),
  })

  const canApply =
    (template.kind === 'list' && spaceID) ||
    (template.kind === 'task' && listID) ||
    (template.kind === 'doc')

  return (
    <Modal open title={`Apply "${template.name}"`} description="Inflates the template into real entities." onClose={onClose}>
      <div className="space-y-3">
        {template.kind === 'list' && (
          <div>
            <label className="block text-xs font-medium text-ink-2 mb-1">Target space</label>
            <select
              className="h-9 w-full bg-surface border border-ink-5/40 rounded-lg px-3 text-sm focus:outline-none focus:ring-2 focus:ring-brand-400/40"
              value={spaceID}
              onChange={(e) => setSpaceID(e.target.value)}
            >
              <option value="">Pick a space…</option>
              {spaces.map((s) => <option key={s.id} value={s.id}>{s.name}</option>)}
            </select>
          </div>
        )}
        {template.kind === 'task' && (
          <div>
            <label className="block text-xs font-medium text-ink-2 mb-1">Target list</label>
            <select
              className="h-9 w-full bg-surface border border-ink-5/40 rounded-lg px-3 text-sm focus:outline-none focus:ring-2 focus:ring-brand-400/40"
              value={listID}
              onChange={(e) => setListID(e.target.value)}
            >
              <option value="">Pick a list…</option>
              {allLists.map((l) => <option key={l.id} value={l.id}>{l.name}</option>)}
            </select>
          </div>
        )}
        {template.kind === 'doc' && (
          <p className="text-xs text-ink-4">Creates the doc at the workspace root.</p>
        )}
        {error && <p className="text-xs text-red-600">{error}</p>}
        <div className="flex justify-end gap-2 pt-1">
          <Button variant="secondary" onClick={onClose}>Cancel</Button>
          <Button onClick={() => apply.mutate()} disabled={!canApply || apply.isPending} loading={apply.isPending}>Apply</Button>
        </div>
      </div>
    </Modal>
  )
}

function defaultSnapshot(kind: TemplateKind): unknown {
  switch (kind) {
    case 'list':
      return {
        name: 'My list',
        statuses: [
          { name: 'Todo',       color: '#6375E8', category: 'active' },
          { name: 'In progress', color: '#F97316', category: 'active' },
          { name: 'Done',       color: '#22C55E', category: 'done'   },
        ],
        tasks: [{ name: 'First task', description: '' }],
      }
    case 'doc':
      return { title: 'Untitled doc', content: {}, pages: [] }
    case 'task':
      return { name: 'New task', description: '', subtasks: [] }
  }
}

function useTemplateLists(spaces: Space[]): Record<string, List[]> {
  const spaceIds = spaces.map((s) => s.id).join(',')
  const { data } = useQuery({
    queryKey: ['templates-page-lists', spaceIds],
    queryFn: async () => {
      const entries: Record<string, List[]> = {}
      await Promise.all(spaces.map(async (s) => {
        entries[s.id] = await api.get(`spaces/${s.id}/lists`).json<List[]>()
      }))
      return entries
    },
    enabled: spaces.length > 0,
  })
  return data ?? {}
}
