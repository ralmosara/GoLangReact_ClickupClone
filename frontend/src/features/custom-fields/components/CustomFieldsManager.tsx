import { useState } from 'react'
import { Button, Input, Modal } from '../../../components/ui'
import { cn } from '../../../lib/utils'
import type { CustomField, FieldKind, FieldOption } from '../../../types'
import { useCreateField, useDeleteField, useFieldsForList, useUpdateField } from '../hooks/useCustomFields'

const KIND_META: Record<FieldKind, { label: string; desc: string }> = {
  text:     { label: 'Text',      desc: 'Short text' },
  number:   { label: 'Number',    desc: 'Integer or decimal' },
  dropdown: { label: 'Dropdown',  desc: 'Single choice' },
  labels:   { label: 'Labels',    desc: 'Multi-choice tags' },
  date:     { label: 'Date',      desc: 'Date / datetime' },
  checkbox: { label: 'Checkbox',  desc: 'Yes / no' },
  url:      { label: 'URL',       desc: 'Clickable link' },
  email:    { label: 'Email',     desc: 'Email address' },
  phone:    { label: 'Phone',     desc: 'Phone number' },
  money:    { label: 'Money',     desc: 'Currency amount' },
  progress: { label: 'Progress',  desc: 'Percent bar' },
  people:   { label: 'People',    desc: 'Workspace members' },
}
const KINDS: FieldKind[] = ['text', 'number', 'dropdown', 'labels', 'date', 'checkbox', 'url', 'email', 'phone', 'money', 'progress', 'people']

export function CustomFieldsManager({ listId, onClose }: { listId: string; onClose: () => void }) {
  const { data: fields } = useFieldsForList(listId)
  const create = useCreateField(listId)
  const update = useUpdateField(listId)
  const del = useDeleteField(listId)

  const [name, setName] = useState('')
  const [kind, setKind] = useState<FieldKind>('text')

  const handleAdd = () => {
    if (!name.trim()) return
    const defaults: Record<FieldKind, unknown> = {
      text: {}, number: {}, date: {}, checkbox: {}, url: {}, email: {}, phone: {}, money: { currency: 'USD' }, progress: { min: 0, max: 100 },
      dropdown: { options: [] }, labels: { options: [] }, people: { multiple: true },
    }
    create.mutate(
      { name: name.trim(), kind, config: defaults[kind] as never, order_index: (fields?.length ?? 0) * 10 },
      { onSuccess: () => setName('') },
    )
  }

  return (
    <Modal open title="Custom fields" description="Define columns for this list." onClose={onClose} width="max-w-2xl">
      <div className="space-y-3">
        {(fields ?? []).map((f) => (
          <FieldRow
            key={f.id}
            field={f}
            onUpdate={(patch) => update.mutate({ id: f.id, ...patch })}
            onDelete={() => {
              if (confirm(`Delete field "${f.name}"? Existing values will be lost.`)) del.mutate(f.id)
            }}
          />
        ))}

        <div className="pt-3 border-t border-ink-5/30 space-y-2">
          <p className="text-xs font-semibold text-ink-2">Add field</p>
          <div className="flex items-center gap-2">
            <Input
              placeholder="Field name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && handleAdd()}
            />
            <select
              className="h-9 px-2 rounded border border-ink-5/40 text-xs"
              value={kind}
              onChange={(e) => setKind(e.target.value as FieldKind)}
            >
              {KINDS.map((k) => (
                <option key={k} value={k}>{KIND_META[k].label}</option>
              ))}
            </select>
            <Button onClick={handleAdd} disabled={!name.trim() || create.isPending} size="sm" loading={create.isPending}>Add</Button>
          </div>
          <p className="text-[11px] text-ink-4">{KIND_META[kind].desc}</p>
        </div>

        <div className="flex justify-end pt-2">
          <Button variant="secondary" onClick={onClose}>Done</Button>
        </div>
      </div>
    </Modal>
  )
}

function FieldRow({
  field,
  onUpdate,
  onDelete,
}: {
  field: CustomField
  onUpdate: (patch: Partial<Pick<CustomField, 'name' | 'config' | 'required'>>) => void
  onDelete: () => void
}) {
  const [name, setName] = useState(field.name)

  return (
    <div className="bg-canvas/60 border border-ink-5/20 rounded-xl p-3 space-y-2">
      <div className="flex items-center gap-2">
        <span className="text-[10px] uppercase tracking-wider text-brand-600 font-bold bg-brand-50 border border-brand-100 rounded px-1.5 py-0.5 shrink-0">
          {KIND_META[field.kind].label}
        </span>
        <input
          className="flex-1 text-sm bg-transparent focus:outline-none text-ink-1 font-medium"
          value={name}
          onChange={(e) => setName(e.target.value)}
          onBlur={() => name !== field.name && onUpdate({ name })}
          onKeyDown={(e) => e.key === 'Enter' && (e.target as HTMLInputElement).blur()}
        />
        <label className="text-[11px] text-ink-4 flex items-center gap-1 cursor-pointer">
          <input
            type="checkbox"
            checked={field.required}
            onChange={(e) => onUpdate({ required: e.target.checked })}
          />
          Required
        </label>
        <button
          onClick={onDelete}
          className="w-7 h-7 rounded text-ink-4 hover:text-red-500 hover:bg-red-50 flex items-center justify-center"
          title="Delete"
        >
          <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>
      {(field.kind === 'dropdown' || field.kind === 'labels') && (
        <OptionsEditor
          options={field.config.options ?? []}
          onChange={(options) => onUpdate({ config: { ...field.config, options } })}
        />
      )}
    </div>
  )
}

const DEFAULT_COLORS = ['#6375E8', '#F97316', '#EAB308', '#22C55E', '#EF4444', '#8B5CF6', '#06B6D4', '#64748B']

function OptionsEditor({ options, onChange }: { options: FieldOption[]; onChange: (next: FieldOption[]) => void }) {
  const [label, setLabel] = useState('')
  const [color, setColor] = useState(DEFAULT_COLORS[0])
  return (
    <div className="space-y-1.5 pl-1">
      {options.map((o, i) => (
        <div key={o.id} className="flex items-center gap-2">
          <input
            type="color"
            className="h-6 w-8 rounded cursor-pointer bg-transparent border-0"
            value={o.color ?? '#94a3b8'}
            onChange={(e) => {
              const next = options.slice()
              next[i] = { ...o, color: e.target.value }
              onChange(next)
            }}
          />
          <input
            className="flex-1 text-xs bg-canvas/40 border border-ink-5/20 rounded px-2 py-1"
            value={o.label}
            onChange={(e) => {
              const next = options.slice()
              next[i] = { ...o, label: e.target.value }
              onChange(next)
            }}
          />
          <button
            onClick={() => onChange(options.filter((_, idx) => idx !== i))}
            className="text-ink-4 hover:text-red-500 text-xs"
          >
            ×
          </button>
        </div>
      ))}
      <div className="flex items-center gap-2 pt-1">
        <input
          type="color"
          className="h-6 w-8 rounded cursor-pointer bg-transparent border-0"
          value={color}
          onChange={(e) => setColor(e.target.value)}
        />
        <input
          className="flex-1 text-xs bg-surface border border-ink-5/30 rounded px-2 py-1 focus:outline-none focus:border-brand-400"
          placeholder="New option label"
          value={label}
          onChange={(e) => setLabel(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && label.trim()) {
              onChange([...options, { id: cryptoRandom(), label: label.trim(), color }])
              setLabel('')
            }
          }}
        />
        <button
          onClick={() => {
            if (!label.trim()) return
            onChange([...options, { id: cryptoRandom(), label: label.trim(), color }])
            setLabel('')
          }}
          className={cn('text-[11px] font-semibold px-2 py-1 rounded', label.trim() ? 'bg-brand-500 text-white hover:bg-brand-600' : 'bg-ink-5/30 text-ink-4')}
        >
          Add
        </button>
      </div>
    </div>
  )
}

function cryptoRandom(): string {
  if (typeof crypto !== 'undefined' && 'randomUUID' in crypto) return crypto.randomUUID()
  return Math.random().toString(36).slice(2)
}
