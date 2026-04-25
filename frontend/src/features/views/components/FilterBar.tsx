import { useState } from 'react'
import * as Popover from '@radix-ui/react-popover'
import { Filter, ArrowUpDown, Layers, X } from 'lucide-react'
import { cn, PRIORITY_LABELS } from '../../../lib/utils'
import type { FilterClause, SortField, Status, Tag, ViewConfig } from '../../../types'

interface Props {
  config: ViewConfig
  onChange: (next: ViewConfig) => void
  statuses: Status[]
  tags: Tag[]
}

const GROUP_OPTIONS = [
  { value: '',           label: 'No group' },
  { value: 'status_id',  label: 'Status' },
  { value: 'priority',   label: 'Priority' },
  { value: 'assignee',   label: 'Assignee' },
] as const

const SORT_OPTIONS = [
  { value: 'position',   label: 'Manual' },
  { value: 'name',       label: 'Name' },
  { value: 'priority',   label: 'Priority' },
  { value: 'due_at',     label: 'Due date' },
  { value: 'created_at', label: 'Created' },
  { value: 'updated_at', label: 'Updated' },
] as const

export function FilterBar({ config, onChange, statuses, tags }: Props) {
  const filters = config.filters ?? []
  const sort = config.sort ?? []
  const groupBy = config.group_by ?? ''

  const setFilters = (f: FilterClause[]) => onChange({ ...config, filters: f })
  const setSort = (s: SortField[]) => onChange({ ...config, sort: s })
  const setGroupBy = (g: string) => onChange({ ...config, group_by: g || undefined })

  return (
    <div className="flex items-center flex-wrap gap-2 px-7 py-2.5 bg-canvas/40 border-b border-ink-5/20 text-xs">
      {/* Group-by */}
      <Popover.Root>
        <Popover.Trigger asChild>
          <button className="inline-flex items-center gap-1.5 h-7 px-2.5 rounded-lg text-ink-2 hover:bg-surface hover:text-ink-1 border border-transparent hover:border-ink-5/30 transition-colors font-medium">
            <Layers className="w-3.5 h-3.5" />
            Group: <span className="text-ink-1">{GROUP_OPTIONS.find((o) => o.value === groupBy)?.label ?? 'No group'}</span>
          </button>
        </Popover.Trigger>
        <Popover.Portal>
          <Popover.Content sideOffset={4} className="bg-surface border border-ink-5/30 rounded-xl shadow-modal p-1 w-40 z-50">
            {GROUP_OPTIONS.map((o) => (
              <button
                key={o.value}
                onClick={() => setGroupBy(o.value)}
                className={cn(
                  'w-full text-left px-2 py-1.5 rounded text-xs hover:bg-ink-1/5',
                  groupBy === o.value ? 'text-brand-600 font-semibold' : 'text-ink-2',
                )}
              >
                {o.label}
              </button>
            ))}
          </Popover.Content>
        </Popover.Portal>
      </Popover.Root>

      {/* Sort */}
      <Popover.Root>
        <Popover.Trigger asChild>
          <button className="inline-flex items-center gap-1.5 h-7 px-2.5 rounded-lg text-ink-2 hover:bg-surface hover:text-ink-1 border border-transparent hover:border-ink-5/30 transition-colors font-medium">
            <ArrowUpDown className="w-3.5 h-3.5" />
            Sort
            {sort.length > 0 && (
              <span className="text-[10px] bg-brand-500 text-white rounded-full px-1.5">{sort.length}</span>
            )}
          </button>
        </Popover.Trigger>
        <Popover.Portal>
          <Popover.Content sideOffset={4} className="bg-surface border border-ink-5/30 rounded-xl shadow-modal p-2 w-64 z-50">
            <p className="text-[10px] font-bold uppercase tracking-wider text-ink-4 px-1 py-1">Sort by</p>
            <div className="space-y-1">
              {SORT_OPTIONS.map((o) => {
                const existing = sort.find((s) => s.field === o.value)
                return (
                  <div key={o.value} className="flex items-center gap-2">
                    <button
                      onClick={() => {
                        if (existing) setSort(sort.filter((s) => s.field !== o.value))
                        else setSort([...sort, { field: o.value, desc: false }])
                      }}
                      className={cn(
                        'flex-1 text-left px-2 py-1.5 rounded text-xs hover:bg-ink-1/5',
                        existing ? 'text-brand-600 font-semibold' : 'text-ink-2',
                      )}
                    >
                      {o.label}
                    </button>
                    {existing && (
                      <button
                        onClick={() => {
                          setSort(sort.map((s) => s.field === o.value ? { ...s, desc: !s.desc } : s))
                        }}
                        className="px-1.5 py-1 rounded text-[10px] font-bold bg-brand-50 text-brand-700 hover:bg-brand-100"
                      >
                        {existing.desc ? 'DESC' : 'ASC'}
                      </button>
                    )}
                  </div>
                )
              })}
            </div>
          </Popover.Content>
        </Popover.Portal>
      </Popover.Root>

      {/* Filters */}
      <FilterPopover
        filters={filters}
        setFilters={setFilters}
        statuses={statuses}
        tags={tags}
      />

      {/* Active-filter chips */}
      <div className="flex items-center gap-1.5 flex-wrap">
        {filters.map((f, idx) => (
          <span
            key={idx}
            className="inline-flex items-center gap-1 bg-brand-50 border border-brand-200 text-brand-700 rounded-full px-2 py-0.5 text-[11px]"
          >
            {describeFilter(f, { statuses, tags })}
            <button
              onClick={() => setFilters(filters.filter((_, i) => i !== idx))}
              className="hover:text-brand-900"
            >
              <X className="w-3 h-3" />
            </button>
          </span>
        ))}
        {sort.map((s, idx) => (
          <span
            key={`s-${idx}`}
            className="inline-flex items-center gap-1 bg-ink-5/30 text-ink-2 rounded-full px-2 py-0.5 text-[11px]"
          >
            {s.field} {s.desc ? '↓' : '↑'}
            <button
              onClick={() => setSort(sort.filter((_, i) => i !== idx))}
              className="hover:text-ink-1"
            >
              <X className="w-3 h-3" />
            </button>
          </span>
        ))}
      </div>
    </div>
  )
}

function FilterPopover({
  filters,
  setFilters,
  statuses,
  tags,
}: {
  filters: FilterClause[]
  setFilters: (f: FilterClause[]) => void
  statuses: Status[]
  tags: Tag[]
}) {
  const [open, setOpen] = useState(false)

  const addFilter = (f: FilterClause) => {
    setFilters([...filters.filter((x) => x.field !== f.field), f])
    setOpen(false)
  }

  return (
    <Popover.Root open={open} onOpenChange={setOpen}>
      <Popover.Trigger asChild>
        <button className="inline-flex items-center gap-1.5 h-7 px-2.5 rounded-lg text-ink-2 hover:bg-surface hover:text-ink-1 border border-transparent hover:border-ink-5/30 transition-colors font-medium">
          <Filter className="w-3.5 h-3.5" />
          Filter
          {filters.length > 0 && (
            <span className="text-[10px] bg-brand-500 text-white rounded-full px-1.5">{filters.length}</span>
          )}
        </button>
      </Popover.Trigger>
      <Popover.Portal>
        <Popover.Content sideOffset={4} className="bg-surface border border-ink-5/30 rounded-xl shadow-modal p-2 w-72 max-h-96 overflow-y-auto z-50">
          <p className="text-[10px] font-bold uppercase tracking-wider text-ink-4 px-1 py-1">Add filter</p>

          <details open className="mb-2">
            <summary className="text-xs font-semibold text-ink-2 cursor-pointer py-1 px-1 hover:bg-ink-1/5 rounded">Status</summary>
            <div className="pl-2 space-y-0.5 mt-1">
              {statuses.map((s) => (
                <button
                  key={s.id}
                  onClick={() => addFilter({ field: 'status_id', op: 'in', value: [s.id] })}
                  className="flex items-center gap-2 w-full px-2 py-1 rounded text-xs text-ink-1 hover:bg-ink-1/5"
                >
                  <span className="w-2 h-2 rounded-full" style={{ backgroundColor: s.color }} />
                  {s.name}
                </button>
              ))}
              {statuses.length === 0 && <p className="text-[11px] text-ink-4 px-2 py-1">No statuses defined</p>}
            </div>
          </details>

          <details className="mb-2">
            <summary className="text-xs font-semibold text-ink-2 cursor-pointer py-1 px-1 hover:bg-ink-1/5 rounded">Priority</summary>
            <div className="pl-2 space-y-0.5 mt-1">
              {Object.entries(PRIORITY_LABELS).map(([v, l]) => (
                <button
                  key={v}
                  onClick={() => addFilter({ field: 'priority', op: 'eq', value: Number(v) })}
                  className="flex items-center gap-2 w-full px-2 py-1 rounded text-xs text-ink-1 hover:bg-ink-1/5"
                >
                  {l}
                </button>
              ))}
            </div>
          </details>

          <details className="mb-2">
            <summary className="text-xs font-semibold text-ink-2 cursor-pointer py-1 px-1 hover:bg-ink-1/5 rounded">Tags</summary>
            <div className="pl-2 space-y-0.5 mt-1">
              {tags.map((t) => (
                <button
                  key={t.id}
                  onClick={() => addFilter({ field: 'tag_id', op: 'in', value: [t.id] })}
                  className="flex items-center gap-2 w-full px-2 py-1 rounded text-xs text-ink-1 hover:bg-ink-1/5"
                >
                  <span className="w-2 h-2 rounded-full" style={{ backgroundColor: t.color ?? '#94a3b8' }} />
                  {t.name}
                </button>
              ))}
              {tags.length === 0 && <p className="text-[11px] text-ink-4 px-2 py-1">No tags in this workspace</p>}
            </div>
          </details>

          <details className="mb-1">
            <summary className="text-xs font-semibold text-ink-2 cursor-pointer py-1 px-1 hover:bg-ink-1/5 rounded">Due date</summary>
            <div className="pl-2 space-y-0.5 mt-1">
              <button onClick={() => addFilter({ field: 'due_at', op: 'notempty' })} className="block w-full text-left px-2 py-1 rounded text-xs text-ink-1 hover:bg-ink-1/5">Has due date</button>
              <button onClick={() => addFilter({ field: 'due_at', op: 'empty' })} className="block w-full text-left px-2 py-1 rounded text-xs text-ink-1 hover:bg-ink-1/5">No due date</button>
              <button onClick={() => addFilter({ field: 'due_at', op: 'before', value: new Date().toISOString() })} className="block w-full text-left px-2 py-1 rounded text-xs text-ink-1 hover:bg-ink-1/5">Overdue</button>
            </div>
          </details>
        </Popover.Content>
      </Popover.Portal>
    </Popover.Root>
  )
}

function describeFilter(f: FilterClause, ctx: { statuses: Status[]; tags: Tag[] }): string {
  if (f.field === 'status_id' && Array.isArray(f.value)) {
    const names = (f.value as string[]).map((id) => ctx.statuses.find((s) => s.id === id)?.name ?? id.slice(0, 6))
    return `Status: ${names.join(', ')}`
  }
  if (f.field === 'priority') {
    return `Priority: ${PRIORITY_LABELS[Number(f.value)] ?? f.value}`
  }
  if (f.field === 'tag_id' && Array.isArray(f.value)) {
    const names = (f.value as string[]).map((id) => ctx.tags.find((t) => t.id === id)?.name ?? id.slice(0, 6))
    return `Tag: ${names.join(', ')}`
  }
  if (f.field === 'due_at') {
    if (f.op === 'empty') return 'No due date'
    if (f.op === 'notempty') return 'Has due date'
    if (f.op === 'before') return 'Overdue'
  }
  return `${f.field} ${f.op}`
}
