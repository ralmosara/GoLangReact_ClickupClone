import { useQuery } from '@tanstack/react-query'
import { useParams } from 'react-router-dom'
import { api } from '../../lib/api'
import { PageSpinner } from '../../components/ui'
import { formatRelative } from '../../lib/utils'
import type { AuditEntry } from '../../types'

export function AuditPage() {
  const { workspaceId } = useParams<{ workspaceId: string }>()

  const { data: entries, isLoading } = useQuery({
    queryKey: ['audit', workspaceId],
    queryFn: () => api.get(`workspaces/${workspaceId}/audit?limit=200`).json<AuditEntry[]>(),
    enabled: !!workspaceId,
  })

  return (
    <div className="max-w-3xl mx-auto px-6 py-10 animate-slide-up">
      <h1 className="text-xl font-bold text-ink-1 mb-2">Activity log</h1>
      <p className="text-sm text-ink-4 mb-6">Every mutation in this workspace is recorded here.</p>

      {isLoading && <PageSpinner />}

      {!isLoading && (entries?.length ?? 0) === 0 && (
        <div className="bg-surface/50 border border-ink-5/20 rounded-2xl p-10 text-center">
          <p className="text-sm text-ink-4">No activity yet. Make a change to see it here.</p>
        </div>
      )}

      {(entries?.length ?? 0) > 0 && (
        <div className="bg-surface border border-ink-5/30 rounded-2xl shadow-card divide-y divide-ink-5/20">
          {(entries ?? []).map((e) => (
            <div key={e.id} className="px-5 py-3 flex items-start gap-4">
              <div className="w-8 h-8 rounded-full bg-brand-50 text-brand-600 text-[11px] font-bold flex items-center justify-center shrink-0">
                {e.entity_type.slice(0, 2).toUpperCase()}
              </div>
              <div className="flex-1 min-w-0">
                <p className="text-sm text-ink-1">
                  <span className="font-semibold">{e.verb}</span>{' '}
                  <span className="text-ink-3">{e.entity_type}</span>
                  {e.entity_id && (
                    <span className="text-[11px] text-ink-4 ml-2 font-mono">{e.entity_id.slice(0, 8)}</span>
                  )}
                </p>
                {Boolean(e.before || e.after) && (
                  <details className="mt-1">
                    <summary className="text-[11px] text-ink-4 cursor-pointer hover:text-ink-2">diff</summary>
                    <pre className="mt-1 p-2 bg-canvas/60 border border-ink-5/20 rounded text-[10px] text-ink-2 whitespace-pre-wrap overflow-x-auto">
                      {String(JSON.stringify({ before: e.before, after: e.after }, null, 2) ?? '')}
                    </pre>
                  </details>
                )}
              </div>
              <span className="text-[11px] text-ink-4 shrink-0">{formatRelative(e.created_at)}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
