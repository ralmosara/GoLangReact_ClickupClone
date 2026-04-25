import { useEffect, useState } from 'react'
import { Command } from 'cmdk'
import { useHotkeys } from 'react-hotkeys-hook'
import { useNavigate, useParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { FileText, MessageSquare, MessageCircle, CheckSquare, Search } from 'lucide-react'

import { api } from '../../lib/api'
import type { SearchHit } from '../../types'
import { cn } from '../../lib/utils'

/**
 * Global command palette. Triggered by Cmd/Ctrl+K. Debounces queries to the
 * FTS endpoint and routes results by entity type.
 */
export function SearchPalette() {
  const { workspaceId } = useParams<{ workspaceId?: string }>()
  const navigate = useNavigate()
  const [open, setOpen] = useState(false)
  const [query, setQuery] = useState('')
  const [debounced, setDebounced] = useState('')

  useHotkeys('meta+k, ctrl+k', (e) => {
    e.preventDefault()
    setOpen(true)
  }, { enableOnFormTags: true })

  useEffect(() => {
    const t = setTimeout(() => setDebounced(query), 180)
    return () => clearTimeout(t)
  }, [query])

  const { data: hits = [], isLoading } = useQuery({
    queryKey: ['search', workspaceId, debounced],
    queryFn: () =>
      api.get(`workspaces/${workspaceId}/search?q=${encodeURIComponent(debounced)}&limit=40`).json<SearchHit[]>(),
    enabled: !!workspaceId && debounced.trim().length >= 2,
  })

  if (!open || !workspaceId) return null

  const go = (hit: SearchHit) => {
    setOpen(false)
    switch (hit.entity_type) {
      case 'task':
        navigate(`/workspaces/${workspaceId}/tasks/${hit.task_id ?? hit.entity_id}`)
        break
      case 'doc':
        navigate(`/workspaces/${workspaceId}/docs?doc=${hit.doc_id ?? hit.entity_id}`)
        break
      case 'comment':
        if (hit.task_id) navigate(`/workspaces/${workspaceId}/tasks/${hit.task_id}`)
        break
      case 'message':
        if (hit.channel_id) navigate(`/workspaces/${workspaceId}/chat?channel=${hit.channel_id}`)
        break
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
          <kbd className="text-[10px] text-ink-4 bg-ink-5/20 rounded px-1.5 py-0.5 font-mono">ESC</kbd>
        </div>
        <Command.List className="max-h-[420px] overflow-y-auto p-2">
          {!debounced && <p className="text-xs text-ink-4 px-2 py-3">Type at least 2 characters to search.</p>}
          {debounced && isLoading && (
            <p className="text-xs text-ink-4 px-2 py-3">Searching…</p>
          )}
          {debounced && !isLoading && hits.length === 0 && (
            <Command.Empty className="text-xs text-ink-4 px-2 py-3">No results.</Command.Empty>
          )}
          {hits.map((hit) => (
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
