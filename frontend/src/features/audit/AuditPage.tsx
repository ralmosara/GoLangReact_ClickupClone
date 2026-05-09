import { useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useParams } from 'react-router-dom'
import { Activity, Download, Filter as FilterIcon, RefreshCw, X } from 'lucide-react'

import { api } from '../../lib/api'
import { Button, PageSpinner } from '../../components/ui'
import { formatRelative } from '../../lib/utils'
import type { AuditEntry, AuditFacets } from '../../types'
import type { WorkspaceUser } from '../../hooks/useUsers'

// VERB_TONE colours rows so the eye finds creates/deletes quickly. Falls
// back to neutral grey for verbs not in the map (which keeps custom verbs
// like "comment.mentioned" rendering sensibly without a code change).
const VERB_TONE: Record<string, { bg: string; text: string; ring: string }> = {
  created:  { bg: 'bg-green-50',  text: 'text-green-700',  ring: 'ring-green-200' },
  updated:  { bg: 'bg-blue-50',   text: 'text-blue-700',   ring: 'ring-blue-200' },
  deleted:  { bg: 'bg-red-50',    text: 'text-red-700',    ring: 'ring-red-200' },
  archived: { bg: 'bg-amber-50',  text: 'text-amber-700',  ring: 'ring-amber-200' },
  invited:  { bg: 'bg-purple-50', text: 'text-purple-700', ring: 'ring-purple-200' },
}
function toneFor(verb: string) {
  return VERB_TONE[verb] ?? { bg: 'bg-ink-5/20', text: 'text-ink-2', ring: 'ring-ink-5/30' }
}

interface Filters {
  entity_type: string
  verb: string
  actor_id: string
  since: string   // YYYY-MM-DD
  before: string  // YYYY-MM-DD
}
const EMPTY: Filters = { entity_type: '', verb: '', actor_id: '', since: '', before: '' }

export function AuditPage() {
  const { workspaceId } = useParams<{ workspaceId: string }>()
  const [filters, setFilters] = useState<Filters>(EMPTY)
  const [limit, setLimit] = useState(100)

  // Build the query string only from non-empty filters so the cache key
  // collapses to "no filters" when the user clears them. Otherwise every
  // partial filter combo would be its own cache entry.
  const queryString = useMemo(() => {
    const p = new URLSearchParams()
    p.set('limit', String(limit))
    if (filters.entity_type) p.set('entity_type', filters.entity_type)
    if (filters.verb)        p.set('verb', filters.verb)
    if (filters.actor_id)    p.set('actor_id', filters.actor_id)
    if (filters.since)       p.set('since', filters.since)
    if (filters.before)      p.set('before', filters.before)
    return p.toString()
  }, [filters, limit])

  const { data: entries, isLoading, isFetching, refetch } = useQuery({
    queryKey: ['audit', workspaceId, queryString],
    queryFn: () => api.get(`workspaces/${workspaceId}/audit?${queryString}`).json<AuditEntry[]>(),
    enabled: !!workspaceId,
  })

  const { data: facets } = useQuery({
    queryKey: ['audit-facets', workspaceId],
    queryFn: () => api.get(`workspaces/${workspaceId}/audit/facets`).json<AuditFacets>(),
    enabled: !!workspaceId,
    staleTime: 60_000,
  })

  // Member roster powers the actor name lookup. /workspaces/{id}/users is
  // already cached by the User Management page so this is a free hit on
  // most navigations.
  const { data: users } = useQuery({
    queryKey: ['workspace-users', workspaceId],
    queryFn: () => api.get(`workspaces/${workspaceId}/users`).json<WorkspaceUser[]>(),
    enabled: !!workspaceId,
    staleTime: 60_000,
  })
  const userById = useMemo(() => {
    const m = new Map<string, WorkspaceUser>()
    ;(users ?? []).forEach((u) => m.set(u.id, u))
    return m
  }, [users])

  const exportURL = (fmt: 'csv' | 'json') =>
    `/api/v1/workspaces/${workspaceId}/audit/export?format=${fmt}`

  const dirty = JSON.stringify(filters) !== JSON.stringify(EMPTY)

  return (
    <div className="max-w-5xl mx-auto px-6 py-8 animate-slide-up">
      <header className="flex items-start justify-between gap-4 mb-6">
        <div>
          <h1 className="text-xl font-bold text-ink-1 flex items-center gap-2">
            <Activity className="w-4 h-4 text-brand-500" />
            Activity log
          </h1>
          <p className="text-sm text-ink-4 mt-1">
            Every mutation in this workspace is recorded here. Filters narrow the view; the export covers the full retained range.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="secondary"
            size="sm"
            onClick={() => refetch()}
            disabled={isFetching}
          >
            <RefreshCw className={`w-3.5 h-3.5 mr-1 ${isFetching ? 'animate-spin' : ''}`} />
            Refresh
          </Button>
          <a
            href={exportURL('csv')}
            className="inline-flex items-center gap-1.5 h-8 px-3 rounded-lg text-sm font-medium border border-ink-5/40 text-ink-2 hover:text-brand-600 hover:bg-brand-50 hover:border-brand-200 transition-colors"
            title="Download CSV (admin only)"
          >
            <Download className="w-3.5 h-3.5" />
            CSV
          </a>
          <a
            href={exportURL('json')}
            className="inline-flex items-center gap-1.5 h-8 px-3 rounded-lg text-sm font-medium border border-ink-5/40 text-ink-2 hover:text-brand-600 hover:bg-brand-50 hover:border-brand-200 transition-colors"
            title="Download newline-delimited JSON (admin only)"
          >
            <Download className="w-3.5 h-3.5" />
            JSON
          </a>
        </div>
      </header>

      {/* Filter bar */}
      <section className="bg-surface border border-ink-5/30 rounded-2xl shadow-card p-4 mb-4">
        <div className="flex items-center gap-2 mb-3">
          <FilterIcon className="w-3.5 h-3.5 text-ink-3" />
          <p className="text-xs font-semibold text-ink-2 uppercase tracking-wider">Filters</p>
          {dirty && (
            <button
              onClick={() => setFilters(EMPTY)}
              className="ml-auto inline-flex items-center gap-1 text-[11px] text-ink-3 hover:text-red-500"
            >
              <X className="w-3 h-3" /> Clear
            </button>
          )}
        </div>
        <div className="grid grid-cols-2 md:grid-cols-5 gap-2">
          <FilterSelect
            label="Entity type"
            value={filters.entity_type}
            onChange={(v) => setFilters((f) => ({ ...f, entity_type: v }))}
            options={facets?.entity_types ?? []}
          />
          <FilterSelect
            label="Verb"
            value={filters.verb}
            onChange={(v) => setFilters((f) => ({ ...f, verb: v }))}
            options={facets?.verbs ?? []}
          />
          <FilterSelect
            label="Actor"
            value={filters.actor_id}
            onChange={(v) => setFilters((f) => ({ ...f, actor_id: v }))}
            options={(facets?.actor_ids ?? []).map((id) => ({
              value: id,
              label: userById.get(id)?.name ?? userById.get(id)?.email ?? `${id.slice(0, 8)}…`,
            }))}
          />
          <FilterDate
            label="Since"
            value={filters.since}
            onChange={(v) => setFilters((f) => ({ ...f, since: v }))}
          />
          <FilterDate
            label="Before"
            value={filters.before}
            onChange={(v) => setFilters((f) => ({ ...f, before: v }))}
          />
        </div>
      </section>

      {/* Result list */}
      {isLoading && <PageSpinner />}

      {!isLoading && (entries?.length ?? 0) === 0 && (
        <div className="bg-surface/50 border border-ink-5/20 rounded-2xl p-12 text-center">
          <p className="text-sm text-ink-3">No activity matches these filters.</p>
          {dirty && (
            <button
              onClick={() => setFilters(EMPTY)}
              className="mt-2 text-xs font-medium text-brand-600 hover:text-brand-700"
            >
              Clear filters
            </button>
          )}
        </div>
      )}

      {(entries?.length ?? 0) > 0 && (
        <div className="bg-surface border border-ink-5/30 rounded-2xl shadow-card divide-y divide-ink-5/20 overflow-hidden">
          {(entries ?? []).map((e) => {
            const tone = toneFor(e.verb)
            const actor = e.actor_id ? userById.get(e.actor_id) : undefined
            return (
              <article key={e.id} className="px-5 py-3 flex items-start gap-4 hover:bg-canvas/40 transition-colors">
                <span
                  className={`inline-flex items-center justify-center text-[10px] font-bold uppercase tracking-wider px-2 py-0.5 rounded-md ring-1 ${tone.bg} ${tone.text} ${tone.ring}`}
                >
                  {e.verb}
                </span>
                <div className="flex-1 min-w-0">
                  <p className="text-sm text-ink-1">
                    <span className="font-semibold text-ink-2">{actor?.name ?? actor?.email ?? 'system'}</span>
                    <span className="text-ink-3"> {e.verb} a </span>
                    <span className="font-mono text-[12px] bg-ink-5/15 px-1.5 py-0.5 rounded text-ink-2">{e.entity_type}</span>
                    {e.entity_id && (
                      <span className="text-[10px] text-ink-4 ml-2 font-mono">{e.entity_id.slice(0, 8)}</span>
                    )}
                  </p>
                  {Boolean(e.before || e.after) && (
                    <details className="mt-1.5">
                      <summary className="text-[11px] text-ink-4 cursor-pointer hover:text-ink-2 select-none">
                        diff
                      </summary>
                      <pre className="mt-1 p-2 bg-canvas/60 border border-ink-5/20 rounded text-[10px] text-ink-2 whitespace-pre-wrap overflow-x-auto max-h-64">
                        {JSON.stringify({ before: e.before, after: e.after }, null, 2)}
                      </pre>
                    </details>
                  )}
                </div>
                <time
                  className="text-[11px] text-ink-4 shrink-0 tabular-nums"
                  title={new Date(e.created_at).toLocaleString()}
                >
                  {formatRelative(e.created_at)}
                </time>
              </article>
            )
          })}
        </div>
      )}

      {/* Pagination — list page is fixed-size; loading more re-issues the
          same query with a higher limit. Cheaper than offset paging for the
          common "scroll a bit further" usage and avoids stale-cursor bugs. */}
      {(entries?.length ?? 0) >= limit && (
        <div className="flex justify-center mt-4">
          <Button variant="secondary" size="sm" onClick={() => setLimit((n) => n + 100)}>
            Load 100 more
          </Button>
        </div>
      )}
    </div>
  )
}

/* --- filter primitives --------------------------------------------------- */

type SelectOpt = string | { value: string; label: string }

function FilterSelect({
  label, value, onChange, options,
}: {
  label: string
  value: string
  onChange: (v: string) => void
  options: SelectOpt[]
}) {
  return (
    <label className="block text-[11px] font-medium text-ink-3">
      {label}
      <select
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="mt-1 h-8 w-full rounded-md border border-ink-5/40 bg-surface px-2 text-xs text-ink-1 focus:outline-none focus:ring-2 focus:ring-brand-400/40"
      >
        <option value="">Any</option>
        {options.map((o) => {
          const v = typeof o === 'string' ? o : o.value
          const l = typeof o === 'string' ? o : o.label
          return <option key={v} value={v}>{l}</option>
        })}
      </select>
    </label>
  )
}

function FilterDate({
  label, value, onChange,
}: {
  label: string
  value: string
  onChange: (v: string) => void
}) {
  return (
    <label className="block text-[11px] font-medium text-ink-3">
      {label}
      <input
        type="date"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="mt-1 h-8 w-full rounded-md border border-ink-5/40 bg-surface px-2 text-xs text-ink-1 focus:outline-none focus:ring-2 focus:ring-brand-400/40"
      />
    </label>
  )
}
