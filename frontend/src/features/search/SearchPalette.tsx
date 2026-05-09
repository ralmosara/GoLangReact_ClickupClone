import { useEffect, useMemo, useState } from 'react'
import { Command } from 'cmdk'
import { useHotkeys } from 'react-hotkeys-hook'
import { useNavigate, useParams } from 'react-router-dom'
import { useMutation, useQuery } from '@tanstack/react-query'
import {
  Bookmark, BookmarkPlus, CheckSquare, FileText, MessageCircle, MessageSquare,
  Search, Star, Trash2, X,
} from 'lucide-react'

import { api } from '../../lib/api'
import { queryClient } from '../../lib/queryClient'
import type { SavedSearch, SearchHit } from '../../types'
import { cn } from '../../lib/utils'

// EntityFilter — single-select facet over the entity_type vocabulary the
// FTS service emits. "" means "all entities".
type EntityFilter = '' | 'task' | 'doc' | 'comment' | 'message'
const ENTITY_FILTERS: { value: EntityFilter; label: string; icon: typeof Search }[] = [
  { value: '',        label: 'All',      icon: Search },
  { value: 'task',    label: 'Tasks',    icon: CheckSquare },
  { value: 'doc',     label: 'Docs',     icon: FileText },
  { value: 'comment', label: 'Comments', icon: MessageSquare },
  { value: 'message', label: 'Chat',     icon: MessageCircle },
]

/**
 * Global command palette (Cmd/Ctrl+K). Now with:
 *   - entity_type facet filter (tabs across the top)
 *   - saved searches (pinned + recent) above the live results
 *   - "save current query" inline action
 *
 * The actual FTS server still does whole-workspace search; the entity
 * filter is applied client-side from the unified result list. Cheap and
 * keeps the server contract unchanged.
 */
export function SearchPalette() {
  const { workspaceId } = useParams<{ workspaceId?: string }>()
  const navigate = useNavigate()
  const [open, setOpen] = useState(false)
  const [query, setQuery] = useState('')
  const [debounced, setDebounced] = useState('')
  const [entity, setEntity] = useState<EntityFilter>('')
  const [showSaveBar, setShowSaveBar] = useState(false)
  const [saveName, setSaveName] = useState('')

  useHotkeys('meta+k, ctrl+k', (e) => {
    e.preventDefault()
    setOpen(true)
  }, { enableOnFormTags: true })

  useEffect(() => {
    const t = setTimeout(() => setDebounced(query), 180)
    return () => clearTimeout(t)
  }, [query])

  // Reset transient state on close so reopening looks fresh.
  useEffect(() => {
    if (!open) {
      setShowSaveBar(false)
      setSaveName('')
    }
  }, [open])

  const { data: hits = [], isLoading } = useQuery({
    queryKey: ['search', workspaceId, debounced],
    queryFn: () =>
      api.get(`workspaces/${workspaceId}/search?q=${encodeURIComponent(debounced)}&limit=40`).json<SearchHit[]>(),
    enabled: !!workspaceId && debounced.trim().length >= 2,
  })

  const { data: saved = [] } = useQuery({
    queryKey: ['saved-searches', workspaceId],
    queryFn: () => api.get(`workspaces/${workspaceId}/saved-searches`).json<SavedSearch[]>(),
    enabled: !!workspaceId,
  })

  const filteredHits = useMemo(() => {
    if (!entity) return hits
    return hits.filter((h) => h.entity_type === entity)
  }, [hits, entity])

  // Per-entity counts for the facet badges. Computed off the raw hits
  // (not filteredHits) so each tab shows its own population.
  const counts = useMemo(() => {
    const c: Record<string, number> = {}
    for (const h of hits) c[h.entity_type] = (c[h.entity_type] ?? 0) + 1
    return c
  }, [hits])

  const create = useMutation({
    mutationFn: () =>
      api.post(`workspaces/${workspaceId}/saved-searches`, { json: {
        name: saveName.trim(),
        entity_type: entity || null,
        query_text: query.trim(),
        filters: [],
        pinned: false,
      } }).json<SavedSearch>(),
    onSuccess: () => {
      setShowSaveBar(false); setSaveName('')
      queryClient.invalidateQueries({ queryKey: ['saved-searches', workspaceId] })
    },
  })

  const togglePin = useMutation({
    mutationFn: (s: SavedSearch) =>
      api.patch(`saved-searches/${s.id}`, { json: {
        name: s.name, entity_type: s.entity_type, query_text: s.query_text,
        filters: s.filters, pinned: !s.pinned,
      } }).json<SavedSearch>(),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['saved-searches', workspaceId] }),
  })

  const remove = useMutation({
    mutationFn: (id: string) => api.delete(`saved-searches/${id}`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['saved-searches', workspaceId] }),
  })

  const markUsed = useMutation({
    mutationFn: (id: string) => api.post(`saved-searches/${id}/used`),
    // No invalidate — last_used_at change matters for ordering only and
    // we don't want to flicker the dropdown. Next palette open refetches.
  })

  if (!open || !workspaceId) return null

  const runSaved = (s: SavedSearch) => {
    setQuery(s.query_text)
    setDebounced(s.query_text)
    setEntity((s.entity_type as EntityFilter) ?? '')
    markUsed.mutate(s.id)
  }

  const go = (hit: SearchHit) => {
    setOpen(false)
    switch (hit.entity_type) {
      case 'task':    navigate(`/workspaces/${workspaceId}/tasks/${hit.task_id ?? hit.entity_id}`); break
      case 'doc':     navigate(`/workspaces/${workspaceId}/docs?doc=${hit.doc_id ?? hit.entity_id}`); break
      case 'comment': if (hit.task_id) navigate(`/workspaces/${workspaceId}/tasks/${hit.task_id}`); break
      case 'message': if (hit.channel_id) navigate(`/workspaces/${workspaceId}/chat?channel=${hit.channel_id}`); break
    }
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-start justify-center p-4 pt-28 animate-fade-in"
      onClick={() => setOpen(false)}
    >
      <div className="absolute inset-0 bg-ink-1/30 backdrop-blur-sm" />
      <Command
        className="relative bg-surface w-full max-w-xl rounded-2xl shadow-modal border border-ink-5/30 overflow-hidden animate-scale-in"
        onClick={(e) => e.stopPropagation()}
        shouldFilter={false}
      >
        <div className="flex items-center gap-2 px-4 border-b border-ink-5/20">
          <Search className="w-4 h-4 text-ink-4" />
          <Command.Input
            value={query}
            onValueChange={setQuery}
            placeholder="Search tasks, docs, comments, messages…"
            className="flex-1 h-12 bg-transparent text-sm text-ink-1 focus:outline-none placeholder:text-ink-4"
            autoFocus
          />
          {query.trim().length >= 2 && !showSaveBar && (
            <button
              onClick={() => { setShowSaveBar(true); setSaveName(query.trim()) }}
              title="Save this search"
              className="inline-flex items-center gap-1 text-[11px] text-ink-3 hover:text-brand-600"
            >
              <BookmarkPlus className="w-3.5 h-3.5" /> Save
            </button>
          )}
          <kbd className="text-[10px] text-ink-4 bg-ink-5/20 rounded px-1.5 py-0.5 font-mono">ESC</kbd>
        </div>

        {/* Save-bar (inline, shown only after Save click) */}
        {showSaveBar && (
          <div className="flex items-center gap-2 px-4 py-2 bg-brand-50/60 border-b border-brand-100">
            <BookmarkPlus className="w-3.5 h-3.5 text-brand-500 shrink-0" />
            <input
              autoFocus
              value={saveName}
              onChange={(e) => setSaveName(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' && saveName.trim()) create.mutate()
                if (e.key === 'Escape') setShowSaveBar(false)
              }}
              placeholder="Name this search…"
              className="flex-1 h-7 bg-surface border border-ink-5/30 rounded px-2 text-xs focus:outline-none focus:ring-2 focus:ring-brand-400/40"
            />
            <button
              onClick={() => create.mutate()}
              disabled={!saveName.trim() || create.isPending}
              className="text-[11px] font-medium text-brand-700 hover:text-brand-900 disabled:opacity-40"
            >
              Save
            </button>
            <button onClick={() => setShowSaveBar(false)} className="text-ink-4 hover:text-ink-1">
              <X className="w-3.5 h-3.5" />
            </button>
          </div>
        )}

        {/* Facet tabs */}
        <div className="flex items-center gap-0.5 px-3 pt-2 border-b border-ink-5/10">
          {ENTITY_FILTERS.map(({ value, label, icon: Icon }) => {
            const active = entity === value
            const cnt = value ? counts[value] : hits.length
            return (
              <button
                key={value || 'all'}
                onClick={() => setEntity(value)}
                className={cn(
                  'inline-flex items-center gap-1.5 px-2.5 h-7 mb-2 rounded-md text-xs font-medium transition-colors',
                  active
                    ? 'bg-brand-50 text-brand-700'
                    : 'text-ink-3 hover:bg-ink-1/[0.04] hover:text-ink-1',
                )}
              >
                <Icon className="w-3 h-3" />
                {label}
                {cnt > 0 && (
                  <span className="ml-0.5 text-[10px] tabular-nums opacity-70">{cnt}</span>
                )}
              </button>
            )
          })}
        </div>

        <Command.List className="max-h-[440px] overflow-y-auto p-2">
          {/* Saved searches — only visible when the input is empty */}
          {!debounced && saved.length > 0 && (
            <>
              <p className="text-[10px] font-bold uppercase tracking-wider text-ink-4 px-2 py-1.5">Saved</p>
              {saved.map((s) => (
                <div
                  key={s.id}
                  className="group flex items-center gap-2 px-2 py-1.5 rounded-lg hover:bg-ink-1/[0.04]"
                >
                  <button
                    onClick={() => togglePin.mutate(s)}
                    className={cn(
                      'shrink-0',
                      s.pinned ? 'text-amber-500' : 'text-ink-4 opacity-0 group-hover:opacity-100',
                    )}
                    title={s.pinned ? 'Unpin' : 'Pin'}
                  >
                    <Star className={cn('w-3.5 h-3.5', s.pinned && 'fill-current')} />
                  </button>
                  <button
                    onClick={() => runSaved(s)}
                    className="flex-1 min-w-0 text-left flex items-center gap-2"
                  >
                    <Bookmark className="w-3.5 h-3.5 text-brand-400 shrink-0" />
                    <div className="min-w-0">
                      <p className="text-sm text-ink-1 truncate">{s.name}</p>
                      <p className="text-[10px] text-ink-4 truncate">
                        {s.entity_type ?? 'all'}{s.query_text ? ` · "${s.query_text}"` : ''}
                      </p>
                    </div>
                  </button>
                  <button
                    onClick={() => { if (confirm(`Delete saved search "${s.name}"?`)) remove.mutate(s.id) }}
                    className="opacity-0 group-hover:opacity-100 text-ink-4 hover:text-red-500"
                    title="Delete"
                  >
                    <Trash2 className="w-3 h-3" />
                  </button>
                </div>
              ))}
              <div className="h-px bg-ink-5/20 my-2" />
            </>
          )}

          {!debounced && saved.length === 0 && (
            <p className="text-xs text-ink-4 px-2 py-3">
              Type at least 2 characters to search. Saved searches appear here once you create one.
            </p>
          )}

          {debounced && isLoading && (
            <p className="text-xs text-ink-4 px-2 py-3">Searching…</p>
          )}
          {debounced && !isLoading && filteredHits.length === 0 && (
            <Command.Empty className="text-xs text-ink-4 px-2 py-3">
              No results{entity ? ` in ${entity}s` : ''}.
            </Command.Empty>
          )}

          {filteredHits.map((hit) => (
            <Command.Item
              key={`${hit.entity_type}-${hit.entity_id}`}
              value={hit.entity_id + hit.title + hit.snippet}
              onSelect={() => go(hit)}
              className={cn(
                'flex items-start gap-3 px-3 py-2 rounded-lg cursor-pointer',
                'data-[selected=true]:bg-brand-50 data-[selected=true]:text-brand-700',
              )}
            >
              <span className="mt-0.5 shrink-0 text-ink-4">{iconFor(hit.entity_type)}</span>
              <div className="min-w-0 flex-1">
                <p className="text-sm font-medium text-ink-1 truncate">{hit.title}</p>
                {hit.snippet && (
                  <p
                    className="text-[11px] text-ink-4 mt-0.5 truncate"
                    dangerouslySetInnerHTML={{ __html: highlight(hit.snippet) }}
                  />
                )}
              </div>
              <span className="text-[10px] text-ink-5 self-start capitalize shrink-0">{hit.entity_type}</span>
            </Command.Item>
          ))}
        </Command.List>
      </Command>
    </div>
  )
}

function iconFor(kind: string) {
  switch (kind) {
    case 'task':    return <CheckSquare className="w-4 h-4" />
    case 'doc':     return <FileText className="w-4 h-4" />
    case 'comment': return <MessageSquare className="w-4 h-4" />
    case 'message': return <MessageCircle className="w-4 h-4" />
    default:        return <Search className="w-4 h-4" />
  }
}

// ts_headline delimits matches with <b>; let them render.
function highlight(s: string): string {
  return s
    .replace(/&/g, '&amp;')
    .replace(/<(?!\/?b>)/g, '&lt;')
    .replace(/<b>/g, '<mark class="bg-brand-200 text-brand-900 px-0.5 rounded">')
    .replace(/<\/b>/g, '</mark>')
}
