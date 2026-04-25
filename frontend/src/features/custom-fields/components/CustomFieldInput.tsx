import { useState, useEffect } from 'react'
import * as Popover from '@radix-ui/react-popover'
import { useQuery } from '@tanstack/react-query'
import { api } from '../../../lib/api'
import { cn } from '../../../lib/utils'
import type { CustomField, FieldOption, Member } from '../../../types'

interface Props {
  field: CustomField
  value: unknown
  onChange: (next: unknown) => void
  workspaceId?: string
}

/**
 * Polymorphic renderer — picks the matching concrete input for each field kind.
 * Every input debounces onChange on blur (except checkbox/dropdown/labels/people
 * which commit immediately).
 */
export function CustomFieldInput({ field, value, onChange, workspaceId }: Props) {
  switch (field.kind) {
    case 'text':     return <TextInput value={value} onChange={onChange} config={field.config} />
    case 'number':   return <NumberInput value={value} onChange={onChange} config={field.config} />
    case 'dropdown': return <DropdownInput value={value} onChange={onChange} options={field.config.options ?? []} />
    case 'labels':   return <LabelsInput value={value} onChange={onChange} options={field.config.options ?? []} />
    case 'date':     return <DateInput value={value} onChange={onChange} includeTime={!!field.config.include_time} />
    case 'checkbox': return <CheckboxInput value={value} onChange={onChange} />
    case 'url':      return <UrlInput value={value} onChange={onChange} />
    case 'email':    return <EmailInput value={value} onChange={onChange} />
    case 'phone':    return <PhoneInput value={value} onChange={onChange} />
    case 'money':    return <MoneyInput value={value} onChange={onChange} config={field.config} />
    case 'progress': return <ProgressInput value={value} onChange={onChange} config={field.config} />
    case 'people':   return <PeopleInput value={value} onChange={onChange} multiple={!!field.config.multiple} workspaceId={workspaceId} />
  }
}

/* ----- text / number / simple ---------------------------------------------- */

function TextInput({ value, onChange, config }: { value: unknown; onChange: (v: unknown) => void; config: CustomField['config'] }) {
  const [local, setLocal] = useState(asString(value))
  useEffect(() => setLocal(asString(value)), [value])
  return (
    <input
      className="h-8 w-full bg-canvas/50 border border-ink-5/30 rounded-lg px-3 text-xs text-ink-1 focus:outline-none focus:ring-2 focus:ring-brand-400/40 focus:border-brand-400"
      placeholder={config.placeholder ?? 'Empty'}
      maxLength={config.max_length}
      value={local}
      onChange={(e) => setLocal(e.target.value)}
      onBlur={() => local !== asString(value) && onChange(local || null)}
    />
  )
}

function NumberInput({ value, onChange, config }: { value: unknown; onChange: (v: unknown) => void; config: CustomField['config'] }) {
  const [local, setLocal] = useState(asString(value))
  useEffect(() => setLocal(asString(value)), [value])
  return (
    <input
      type="number"
      className="h-8 w-full bg-canvas/50 border border-ink-5/30 rounded-lg px-3 text-xs text-ink-1 focus:outline-none focus:ring-2 focus:ring-brand-400/40 focus:border-brand-400"
      min={config.min}
      max={config.max}
      step={config.step ?? 1}
      value={local}
      onChange={(e) => setLocal(e.target.value)}
      onBlur={() => {
        const n = local === '' ? null : Number(local)
        onChange(n)
      }}
    />
  )
}

function DateInput({ value, onChange, includeTime }: { value: unknown; onChange: (v: unknown) => void; includeTime: boolean }) {
  const iso = typeof value === 'string' ? value : ''
  const local = iso ? (includeTime ? iso.slice(0, 16) : iso.slice(0, 10)) : ''
  return (
    <input
      type={includeTime ? 'datetime-local' : 'date'}
      className="h-8 w-full bg-canvas/50 border border-ink-5/30 rounded-lg px-3 text-xs text-ink-1 focus:outline-none focus:ring-2 focus:ring-brand-400/40"
      value={local}
      onChange={(e) => onChange(e.target.value ? new Date(e.target.value).toISOString() : null)}
    />
  )
}

function CheckboxInput({ value, onChange }: { value: unknown; onChange: (v: unknown) => void }) {
  const on = !!value
  return (
    <button
      onClick={() => onChange(!on)}
      className={cn(
        'w-5 h-5 rounded border flex items-center justify-center transition-colors',
        on ? 'bg-brand-500 border-brand-500' : 'border-ink-5/60 hover:border-brand-400 bg-canvas/50',
      )}
    >
      {on && (
        <svg className="w-3 h-3 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={3} d="M5 13l4 4L19 7" />
        </svg>
      )}
    </button>
  )
}

function UrlInput({ value, onChange }: { value: unknown; onChange: (v: unknown) => void }) {
  const [local, setLocal] = useState(asString(value))
  useEffect(() => setLocal(asString(value)), [value])
  return (
    <div className="flex items-center gap-1.5 w-full">
      <input
        type="url"
        className="h-8 flex-1 bg-canvas/50 border border-ink-5/30 rounded-lg px-3 text-xs text-ink-1 focus:outline-none focus:ring-2 focus:ring-brand-400/40"
        placeholder="https://…"
        value={local}
        onChange={(e) => setLocal(e.target.value)}
        onBlur={() => local !== asString(value) && onChange(local || null)}
      />
      {local && (
        <a href={local} target="_blank" rel="noreferrer" className="text-brand-600 hover:text-brand-700 text-xs">↗</a>
      )}
    </div>
  )
}

function EmailInput({ value, onChange }: { value: unknown; onChange: (v: unknown) => void }) {
  const [local, setLocal] = useState(asString(value))
  useEffect(() => setLocal(asString(value)), [value])
  return (
    <input
      type="email"
      className="h-8 w-full bg-canvas/50 border border-ink-5/30 rounded-lg px-3 text-xs text-ink-1 focus:outline-none focus:ring-2 focus:ring-brand-400/40"
      placeholder="name@example.com"
      value={local}
      onChange={(e) => setLocal(e.target.value)}
      onBlur={() => local !== asString(value) && onChange(local || null)}
    />
  )
}

function PhoneInput({ value, onChange }: { value: unknown; onChange: (v: unknown) => void }) {
  const [local, setLocal] = useState(asString(value))
  useEffect(() => setLocal(asString(value)), [value])
  return (
    <input
      type="tel"
      className="h-8 w-full bg-canvas/50 border border-ink-5/30 rounded-lg px-3 text-xs text-ink-1 focus:outline-none focus:ring-2 focus:ring-brand-400/40"
      placeholder="+1 555 000 0000"
      value={local}
      onChange={(e) => setLocal(e.target.value)}
      onBlur={() => local !== asString(value) && onChange(local || null)}
    />
  )
}

function MoneyInput({ value, onChange, config }: { value: unknown; onChange: (v: unknown) => void; config: CustomField['config'] }) {
  const [local, setLocal] = useState(value != null ? String(value) : '')
  useEffect(() => setLocal(value != null ? String(value) : ''), [value])
  return (
    <div className="flex items-center gap-1.5">
      <span className="text-[11px] font-semibold text-ink-4">{config.currency ?? 'USD'}</span>
      <input
        type="number"
        step={Math.pow(10, -(config.precision ?? 2))}
        className="h-8 flex-1 bg-canvas/50 border border-ink-5/30 rounded-lg px-3 text-xs text-ink-1 focus:outline-none focus:ring-2 focus:ring-brand-400/40"
        value={local}
        onChange={(e) => setLocal(e.target.value)}
        onBlur={() => onChange(local === '' ? null : Number(local))}
      />
    </div>
  )
}

function ProgressInput({ value, onChange, config }: { value: unknown; onChange: (v: unknown) => void; config: CustomField['config'] }) {
  const min = config.min ?? 0
  const max = config.max ?? 100
  const n = typeof value === 'number' ? value : min
  return (
    <div className="flex items-center gap-2 w-full">
      <div className="flex-1 h-1.5 rounded-full bg-ink-5/30 overflow-hidden">
        <div
          className="h-full bg-brand-500 rounded-full transition-all"
          style={{ width: `${((n - min) / (max - min)) * 100}%` }}
        />
      </div>
      <input
        type="number"
        min={min}
        max={max}
        className="h-7 w-16 bg-canvas/50 border border-ink-5/30 rounded text-xs text-ink-1 text-right px-1.5 focus:outline-none focus:ring-2 focus:ring-brand-400/40"
        value={n}
        onChange={(e) => onChange(Math.max(min, Math.min(max, Number(e.target.value))))}
      />
      <span className="text-[10px] text-ink-4">/ {max}</span>
    </div>
  )
}

/* ----- dropdown / labels --------------------------------------------------- */

function DropdownInput({ value, onChange, options }: { value: unknown; onChange: (v: unknown) => void; options: FieldOption[] }) {
  const selected = options.find((o) => o.id === value)
  return (
    <Popover.Root>
      <Popover.Trigger asChild>
        <button className="h-8 w-full bg-canvas/50 border border-ink-5/30 rounded-lg px-2 text-xs text-ink-1 flex items-center justify-between hover:bg-canvas/80">
          {selected ? (
            <span className="inline-flex items-center gap-1.5">
              <span className="w-2 h-2 rounded-full" style={{ backgroundColor: selected.color ?? '#94a3b8' }} />
              {selected.label}
            </span>
          ) : (
            <span className="text-ink-4">Select…</span>
          )}
        </button>
      </Popover.Trigger>
      <Popover.Portal>
        <Popover.Content sideOffset={4} className="bg-surface border border-ink-5/30 rounded-xl shadow-modal p-1 min-w-[200px] z-50">
          {options.map((o) => (
            <button
              key={o.id}
              onClick={() => onChange(value === o.id ? null : o.id)}
              className="flex items-center gap-2 w-full px-2 py-1.5 rounded text-xs text-ink-1 hover:bg-ink-1/5"
            >
              <span className="w-2 h-2 rounded-full" style={{ backgroundColor: o.color ?? '#94a3b8' }} />
              {o.label}
              {value === o.id && <span className="ml-auto text-brand-500">✓</span>}
            </button>
          ))}
          {options.length === 0 && <p className="px-2 py-1 text-[11px] text-ink-4">No options defined</p>}
        </Popover.Content>
      </Popover.Portal>
    </Popover.Root>
  )
}

function LabelsInput({ value, onChange, options }: { value: unknown; onChange: (v: unknown) => void; options: FieldOption[] }) {
  const selected: string[] = Array.isArray(value) ? (value as string[]) : []
  const chosen = options.filter((o) => selected.includes(o.id))
  const toggle = (id: string) => {
    const next = selected.includes(id) ? selected.filter((x) => x !== id) : [...selected, id]
    onChange(next)
  }
  return (
    <Popover.Root>
      <Popover.Trigger asChild>
        <button className="min-h-8 w-full bg-canvas/50 border border-ink-5/30 rounded-lg px-2 py-1 text-xs flex flex-wrap items-center gap-1">
          {chosen.length === 0 && <span className="text-ink-4">Select…</span>}
          {chosen.map((o) => (
            <span
              key={o.id}
              className="text-[10px] font-medium px-1.5 py-0.5 rounded-full"
              style={{ backgroundColor: (o.color ?? '#94a3b8') + '22', color: o.color ?? '#64748b' }}
            >
              {o.label}
            </span>
          ))}
        </button>
      </Popover.Trigger>
      <Popover.Portal>
        <Popover.Content sideOffset={4} className="bg-surface border border-ink-5/30 rounded-xl shadow-modal p-1 min-w-[220px] z-50">
          {options.map((o) => (
            <button
              key={o.id}
              onClick={() => toggle(o.id)}
              className="flex items-center gap-2 w-full px-2 py-1.5 rounded text-xs text-ink-1 hover:bg-ink-1/5"
            >
              <span className="w-2 h-2 rounded-full" style={{ backgroundColor: o.color ?? '#94a3b8' }} />
              {o.label}
              {selected.includes(o.id) && <span className="ml-auto text-brand-500">✓</span>}
            </button>
          ))}
          {options.length === 0 && <p className="px-2 py-1 text-[11px] text-ink-4">No options defined</p>}
        </Popover.Content>
      </Popover.Portal>
    </Popover.Root>
  )
}

/* ----- people -------------------------------------------------------------- */

function PeopleInput({
  value,
  onChange,
  multiple,
  workspaceId,
}: {
  value: unknown
  onChange: (v: unknown) => void
  multiple: boolean
  workspaceId?: string
}) {
  const { data: members } = useQuery({
    queryKey: ['workspace-members', workspaceId],
    queryFn: () => api.get(`workspaces/${workspaceId}/members`).json<Member[]>(),
    enabled: !!workspaceId,
  })

  const selected: string[] = multiple
    ? (Array.isArray(value) ? (value as string[]) : [])
    : (typeof value === 'string' ? [value] : [])

  const toggle = (id: string) => {
    if (!multiple) {
      onChange(selected[0] === id ? null : id)
      return
    }
    const next = selected.includes(id) ? selected.filter((x) => x !== id) : [...selected, id]
    onChange(next)
  }

  const chosen = (members ?? []).filter((m) => selected.includes(m.user_id))

  return (
    <Popover.Root>
      <Popover.Trigger asChild>
        <button className="h-8 w-full bg-canvas/50 border border-ink-5/30 rounded-lg px-2 text-xs text-ink-1 flex items-center gap-1 hover:bg-canvas/80">
          {chosen.length === 0 ? (
            <span className="text-ink-4">Select…</span>
          ) : (
            <div className="flex -space-x-1.5">
              {chosen.slice(0, 3).map((m) => (
                <div key={m.user_id} className="w-5 h-5 rounded-full bg-brand-gradient text-[9px] font-bold text-white flex items-center justify-center border-2 border-surface">
                  {(m.name || m.email || '?').slice(0, 1).toUpperCase()}
                </div>
              ))}
            </div>
          )}
        </button>
      </Popover.Trigger>
      <Popover.Portal>
        <Popover.Content sideOffset={4} className="bg-surface border border-ink-5/30 rounded-xl shadow-modal p-1 min-w-[220px] max-h-64 overflow-y-auto z-50">
          {(members ?? []).map((m) => (
            <button
              key={m.user_id}
              onClick={() => toggle(m.user_id)}
              className="flex items-center gap-2 w-full px-2 py-1.5 rounded text-xs hover:bg-ink-1/5"
            >
              <div className="w-5 h-5 rounded-full bg-brand-gradient text-[9px] font-bold text-white flex items-center justify-center">
                {(m.name || m.email || '?').slice(0, 1).toUpperCase()}
              </div>
              <span className="flex-1 text-ink-1">{m.name || m.email}</span>
              {selected.includes(m.user_id) && <span className="text-brand-500">✓</span>}
            </button>
          ))}
        </Popover.Content>
      </Popover.Portal>
    </Popover.Root>
  )
}

function asString(v: unknown): string {
  if (v == null) return ''
  return String(v)
}
