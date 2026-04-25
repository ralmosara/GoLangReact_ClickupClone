import { useMemo, useState } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import { useParams, useSearchParams } from 'react-router-dom'
import { Plus, ClipboardList, ExternalLink, Trash2, Copy, Check } from 'lucide-react'

import { api } from '../../lib/api'
import { queryClient } from '../../lib/queryClient'
import { Button, Input, Modal, PageSpinner } from '../../components/ui'
import { cn } from '../../lib/utils'
import type { FormDef, FormFieldDef, FormFieldKind, FormSubmission, List, Space } from '../../types'

const FIELD_KINDS: Array<{ value: FormFieldKind; label: string }> = [
  { value: 'text',     label: 'Short text' },
  { value: 'textarea', label: 'Long text' },
  { value: 'number',   label: 'Number' },
  { value: 'email',    label: 'Email' },
  { value: 'select',   label: 'Dropdown' },
  { value: 'checkbox', label: 'Checkbox' },
]

export function FormsPage() {
  const { workspaceId } = useParams<{ workspaceId: string }>()
  const [params, setParams] = useSearchParams()
  const activeId = params.get('id') ?? ''
  const [listId, setListId] = useState('')
  const [showCreate, setShowCreate] = useState(false)

  const { data: spaces = [] } = useQuery({
    queryKey: ['spaces', workspaceId],
    queryFn: () => api.get(`workspaces/${workspaceId}/spaces`).json<Space[]>(),
    enabled: !!workspaceId,
  })
  const listsBySpace = useWorkspaceLists(spaces)
  const allLists = useMemo(
    () => spaces.flatMap((s) => listsBySpace[s.id] ?? []),
    [spaces, listsBySpace],
  )

  const { data: forms = [] } = useQuery({
    queryKey: ['forms', listId],
    queryFn: () => api.get(`lists/${listId}/forms`).json<FormDef[]>(),
    enabled: !!listId,
  })

  const active = forms.find((f) => f.id === activeId) ?? forms[0]

  // Auto-pick the first list when the workspace has any.
  if (!listId && allLists.length > 0) {
    setListId(allLists[0].id)
  }

  return (
    <div className="flex h-full bg-canvas">
      <aside className="w-60 shrink-0 border-r border-ink-5/20 bg-surface/50 flex flex-col">
        <div className="flex items-center justify-between px-4 py-3 border-b border-ink-5/20">
          <h2 className="text-sm font-semibold text-ink-1 flex items-center gap-1.5">
            <ClipboardList className="w-3.5 h-3.5 text-brand-500" />
            Forms
          </h2>
          <button
            onClick={() => setShowCreate(true)}
            disabled={!listId}
            className="h-7 w-7 flex items-center justify-center rounded-lg text-ink-4 hover:text-brand-600 hover:bg-brand-50 disabled:opacity-40"
            title="New form"
          >
            <Plus className="w-3.5 h-3.5" />
          </button>
        </div>
        <div className="px-3 py-2 border-b border-ink-5/20">
          <label className="block text-[10px] font-bold uppercase tracking-wider text-ink-4 mb-1">List</label>
          <select
            className="h-8 w-full bg-surface border border-ink-5/40 rounded text-xs focus:outline-none focus:ring-2 focus:ring-brand-400/40"
            value={listId}
            onChange={(e) => { setListId(e.target.value); setParams({}, { replace: true }) }}
          >
            <option value="">Pick a list…</option>
            {spaces.map((sp) => (
              <optgroup key={sp.id} label={sp.name}>
                {(listsBySpace[sp.id] ?? []).map((l) => (
                  <option key={l.id} value={l.id}>{l.name}</option>
                ))}
              </optgroup>
            ))}
          </select>
        </div>
        <div className="flex-1 overflow-y-auto p-1">
          {listId && forms.length === 0 && (
            <p className="text-xs text-ink-4 px-3 py-4">No forms yet on this list.</p>
          )}
          {forms.map((f) => (
            <button
              key={f.id}
              onClick={() => setParams({ id: f.id }, { replace: true })}
              className={cn(
                'block w-full text-left px-3 py-1.5 rounded text-xs truncate transition-colors',
                active?.id === f.id ? 'bg-brand-50 text-brand-700 font-semibold' : 'text-ink-2 hover:bg-ink-1/5',
              )}
            >
              {f.name} <span className="text-[10px] text-ink-4">· {f.submit_count}</span>
            </button>
          ))}
        </div>
      </aside>

      <main className="flex-1 overflow-y-auto">
        {active ? (
          <FormEditor key={active.id} form={active} />
        ) : (
          <div className="h-full flex items-center justify-center text-sm text-ink-4">
            {listId ? 'Pick or create a form.' : 'Pick a list to see its forms.'}
          </div>
        )}
      </main>

      {showCreate && listId && (
        <CreateFormModal
          listId={listId}
          onCreated={(f) => setParams({ id: f.id }, { replace: true })}
          onClose={() => setShowCreate(false)}
        />
      )}
    </div>
  )
}

function FormEditor({ form }: { form: FormDef }) {
  const [local, setLocal] = useState<FormDef>(form)
  const [copied, setCopied] = useState(false)

  const update = useMutation({
    mutationFn: (patch: Partial<FormDef>) => api.patch(`forms/${form.id}`, { json: patch }).json<FormDef>(),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['forms', form.list_id] }),
  })

  const del = useMutation({
    mutationFn: () => api.delete(`forms/${form.id}`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['forms', form.list_id] }),
  })

  const { data: submissions = [] } = useQuery({
    queryKey: ['form-submissions', form.id],
    queryFn: () => api.get(`forms/${form.id}/submissions`).json<FormSubmission[]>(),
  })

  const publicUrl = `${location.origin}/forms/${form.id}`

  const addField = (kind: FormFieldKind) => {
    const f: FormFieldDef = {
      id: cryptoRandom(),
      label: FIELD_KINDS.find((k) => k.value === kind)?.label ?? 'Field',
      kind,
      required: false,
      options: kind === 'select' ? ['Option 1', 'Option 2'] : undefined,
    }
    const next = [...local.fields, f]
    setLocal({ ...local, fields: next })
    update.mutate({ fields: next as never })
  }

  const updateField = (idx: number, patch: Partial<FormFieldDef>) => {
    const next = local.fields.slice()
    next[idx] = { ...next[idx], ...patch }
    setLocal({ ...local, fields: next })
    update.mutate({ fields: next as never })
  }

  const removeField = (idx: number) => {
    const next = local.fields.filter((_, i) => i !== idx)
    setLocal({ ...local, fields: next })
    update.mutate({ fields: next as never })
  }

  return (
    <div className="max-w-3xl mx-auto px-6 py-8 animate-slide-up">
      <div className="flex items-start justify-between mb-4">
        <div className="flex-1 mr-4">
          <input
            className="w-full text-2xl font-bold text-ink-1 bg-transparent focus:outline-none"
            value={local.name}
            onChange={(e) => setLocal({ ...local, name: e.target.value })}
            onBlur={() => local.name !== form.name && update.mutate({ name: local.name })}
          />
          <textarea
            className="w-full text-sm text-ink-3 bg-transparent focus:outline-none resize-none mt-1"
            placeholder="Describe what this form captures…"
            rows={2}
            value={local.description}
            onChange={(e) => setLocal({ ...local, description: e.target.value })}
            onBlur={() => local.description !== form.description && update.mutate({ description: local.description })}
          />
        </div>
        <Button
          variant="secondary"
          size="sm"
          onClick={() => { if (confirm(`Delete "${form.name}"?`)) del.mutate() }}
        >
          <Trash2 className="w-3.5 h-3.5 mr-1" />
          Delete
        </Button>
      </div>

      <div className="bg-surface border border-ink-5/30 rounded-2xl shadow-card p-4 mb-4">
        <div className="flex items-center justify-between mb-2">
          <p className="text-[10px] font-bold uppercase tracking-wider text-ink-4">Public URL</p>
          <label className="flex items-center gap-1.5 text-[11px] text-ink-3 cursor-pointer">
            <input
              type="checkbox"
              checked={local.is_public}
              onChange={(e) => { setLocal({ ...local, is_public: e.target.checked }); update.mutate({ is_public: e.target.checked }) }}
            />
            Public
          </label>
        </div>
        <div className="flex items-center gap-2">
          <code className="flex-1 text-xs bg-canvas/60 border border-ink-5/20 rounded px-2 py-1.5 truncate font-mono">{publicUrl}</code>
          <Button
            variant="secondary"
            size="sm"
            onClick={() => {
              navigator.clipboard.writeText(publicUrl)
              setCopied(true)
              setTimeout(() => setCopied(false), 1200)
            }}
          >
            {copied ? <Check className="w-3.5 h-3.5" /> : <Copy className="w-3.5 h-3.5" />}
          </Button>
          <a href={publicUrl} target="_blank" rel="noreferrer" className="inline-flex items-center gap-1 h-8 px-2.5 rounded border border-ink-5/30 text-xs text-ink-2 hover:text-ink-1 hover:bg-canvas/60">
            <ExternalLink className="w-3.5 h-3.5" />
          </a>
        </div>
      </div>

      <div className="bg-surface border border-ink-5/30 rounded-2xl shadow-card p-4 mb-4">
        <p className="text-[10px] font-bold uppercase tracking-wider text-ink-4 mb-2">Fields</p>
        <div className="space-y-2">
          {local.fields.map((f, i) => (
            <FieldRow
              key={f.id}
              field={f}
              onChange={(patch) => updateField(i, patch)}
              onDelete={() => removeField(i)}
            />
          ))}
          {local.fields.length === 0 && <p className="text-[11px] text-ink-4">No fields yet.</p>}
        </div>
        <div className="mt-3 flex flex-wrap gap-1.5">
          {FIELD_KINDS.map((k) => (
            <button
              key={k.value}
              onClick={() => addField(k.value)}
              className="text-[11px] font-medium px-2.5 py-1 rounded border border-dashed border-ink-5/50 text-ink-4 hover:text-brand-600 hover:border-brand-300 hover:bg-canvas/60"
            >
              + {k.label}
            </button>
          ))}
        </div>
      </div>

      <div className="bg-surface border border-ink-5/30 rounded-2xl shadow-card p-4">
        <p className="text-[10px] font-bold uppercase tracking-wider text-ink-4 mb-2">
          Submissions ({submissions.length})
        </p>
        {submissions.length === 0 ? (
          <p className="text-xs text-ink-4">No submissions yet.</p>
        ) : (
          <ul className="space-y-2">
            {submissions.slice(0, 20).map((s) => (
              <li key={s.id} className="text-xs bg-canvas/60 border border-ink-5/20 rounded p-2">
                <p className="text-ink-4 mb-1 text-[10px]">{new Date(s.submitted_at).toLocaleString()}</p>
                <pre className="text-ink-1 whitespace-pre-wrap">{JSON.stringify(s.payload, null, 2)}</pre>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  )
}

function FieldRow({
  field, onChange, onDelete,
}: {
  field: FormFieldDef
  onChange: (patch: Partial<FormFieldDef>) => void
  onDelete: () => void
}) {
  return (
    <div className="bg-canvas/60 border border-ink-5/20 rounded-lg p-2.5 group">
      <div className="flex items-center gap-2 mb-1.5">
        <span className="text-[10px] uppercase tracking-wider text-brand-600 font-bold bg-brand-50 border border-brand-100 rounded px-1.5 py-0.5 shrink-0">
          {field.kind}
        </span>
        <input
          className="flex-1 text-sm bg-transparent focus:outline-none text-ink-1 font-medium"
          value={field.label}
          onChange={(e) => onChange({ label: e.target.value })}
        />
        <label className="text-[11px] text-ink-4 flex items-center gap-1 cursor-pointer">
          <input
            type="checkbox"
            checked={field.required}
            onChange={(e) => onChange({ required: e.target.checked })}
          />
          Required
        </label>
        <button onClick={onDelete} className="opacity-0 group-hover:opacity-100 text-ink-4 hover:text-red-500">
          <Trash2 className="w-3.5 h-3.5" />
        </button>
      </div>
      {field.kind === 'select' && (
        <textarea
          className="w-full text-[11px] bg-surface border border-ink-5/20 rounded px-2 py-1 resize-none focus:outline-none focus:border-brand-400"
          placeholder="One option per line"
          rows={2}
          value={(field.options ?? []).join('\n')}
          onChange={(e) => onChange({ options: e.target.value.split('\n').map((s) => s.trim()).filter(Boolean) })}
        />
      )}
    </div>
  )
}

function CreateFormModal({
  listId, onCreated, onClose,
}: {
  listId: string
  onCreated: (f: FormDef) => void
  onClose: () => void
}) {
  const [name, setName] = useState('')
  const create = useMutation({
    mutationFn: () => api.post(`lists/${listId}/forms`, { json: { name, is_public: true, fields: [] } }).json<FormDef>(),
    onSuccess: (f) => {
      queryClient.invalidateQueries({ queryKey: ['forms', listId] })
      onCreated(f)
      onClose()
    },
  })
  return (
    <Modal open title="New form" description="Public form that creates a task on submit." onClose={onClose}>
      <div className="space-y-3">
        <Input label="Name" value={name} onChange={(e) => setName(e.target.value)} placeholder="Bug report" autoFocus />
        <div className="flex justify-end gap-2 pt-1">
          <Button variant="secondary" onClick={onClose}>Cancel</Button>
          <Button onClick={() => create.mutate()} disabled={!name.trim() || create.isPending} loading={create.isPending}>Create</Button>
        </div>
      </div>
    </Modal>
  )
}

function cryptoRandom(): string {
  if (typeof crypto !== 'undefined' && 'randomUUID' in crypto) return crypto.randomUUID()
  return Math.random().toString(36).slice(2)
}

function useWorkspaceLists(spaces: Space[]): Record<string, List[]> {
  const spaceIds = spaces.map((s) => s.id).join(',')
  const { data } = useQuery({
    queryKey: ['forms-page-lists', spaceIds],
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
