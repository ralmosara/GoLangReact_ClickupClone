import { useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useParams, useSearchParams } from 'react-router-dom'
import { Flag } from 'lucide-react'

import { api } from '../../lib/api'
import { cn } from '../../lib/utils'
import { PageSpinner } from '../../components/ui'
import type { List, Space } from '../../types'
import { SprintsPanel } from './SprintsPanel'

// A lightweight two-pane layout: pick a list on the left, see its sprints on
// the right. Saves us from having to embed the panel inside the board route.
export function SprintsPage() {
  const { workspaceId } = useParams<{ workspaceId: string }>()
  const [params, setParams] = useSearchParams()
  const activeListId = params.get('list') ?? ''

  const { data: spaces = [], isLoading } = useQuery({
    queryKey: ['spaces', workspaceId],
    queryFn: () => api.get(`workspaces/${workspaceId}/spaces`).json<Space[]>(),
    enabled: !!workspaceId,
  })

  const listsBySpace = useWorkspaceLists(spaces)
  const activeList = useMemo(() => {
    for (const sp of spaces) {
      const list = (listsBySpace[sp.id] ?? []).find((l) => l.id === activeListId)
      if (list) return list
    }
    return null
  }, [spaces, listsBySpace, activeListId])

  return (
    <div className="flex h-full bg-canvas">
      <aside className="w-56 shrink-0 border-r border-ink-5/20 bg-surface/50 flex flex-col">
        <div className="px-4 py-3 border-b border-ink-5/20">
          <h2 className="text-sm font-semibold text-ink-1 flex items-center gap-1.5">
            <Flag className="w-3.5 h-3.5 text-brand-500" />
            Sprints
          </h2>
          <p className="text-[11px] text-ink-4 mt-0.5">Pick a list</p>
        </div>
        <div className="flex-1 overflow-y-auto p-2 space-y-3">
          {isLoading && <PageSpinner />}
          {spaces.map((sp) => (
            <div key={sp.id}>
              <p className="text-[10px] font-bold uppercase tracking-wider text-ink-4 px-2 mb-1">{sp.name}</p>
              <div className="space-y-0.5">
                {(listsBySpace[sp.id] ?? []).map((l) => (
                  <button
                    key={l.id}
                    onClick={() => setParams({ list: l.id }, { replace: true })}
                    className={cn(
                      'block w-full text-left px-3 py-1.5 rounded text-xs truncate transition-colors',
                      activeListId === l.id ? 'bg-brand-50 text-brand-700 font-semibold' : 'text-ink-2 hover:bg-ink-1/5',
                    )}
                  >
                    {l.name}
                  </button>
                ))}
              </div>
            </div>
          ))}
        </div>
      </aside>

      <main className="flex-1 overflow-y-auto p-6">
        {activeList ? (
          <>
            <h1 className="text-xl font-bold text-ink-1 mb-4">{activeList.name}</h1>
            <SprintsPanel listId={activeList.id} />
          </>
        ) : (
          <div className="h-full flex items-center justify-center text-sm text-ink-4">
            Pick a list to manage its sprints.
          </div>
        )}
      </main>
    </div>
  )
}

function useWorkspaceLists(spaces: Space[]): Record<string, List[]> {
  const spaceIds = spaces.map((s) => s.id).join(',')
  const { data } = useQuery({
    queryKey: ['sprints-page-lists', spaceIds],
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
