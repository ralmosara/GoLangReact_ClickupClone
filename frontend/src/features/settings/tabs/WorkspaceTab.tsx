import { useParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'

import { api } from '../../../lib/api'
import type { Workspace } from '../../../types'

export function WorkspaceTab() {
  const { workspaceId } = useParams<{ workspaceId: string }>()

  const { data: workspace } = useQuery({
    queryKey: ['workspace', workspaceId],
    queryFn: () => api.get(`workspaces/${workspaceId}`).json<Workspace>().catch(() => null as unknown as Workspace | null),
    enabled: !!workspaceId,
  })

  // Workspace mutation endpoints (rename/transfer/delete) aren't all
  // wired yet — surface what we have read-only and link the admin paths
  // for the rest. Keeps the tab honest about capability rather than
  // showing greyed-out controls.
  return (
    <section className="bg-surface border border-ink-5/30 rounded-2xl shadow-card p-6 space-y-5 dark:bg-white/[0.04] dark:border-white/10">
      <div>
        <h2 className="text-sm font-semibold text-ink-1 mb-1 dark:text-white">Workspace</h2>
        <p className="text-xs text-ink-4 dark:text-white/50">Membership, roles, and audit live in their own pages.</p>
      </div>

      <dl className="grid grid-cols-2 gap-4 text-xs">
        <div>
          <dt className="text-ink-4 dark:text-white/40">Name</dt>
          <dd className="text-ink-1 font-medium mt-0.5 dark:text-white">{workspace?.name ?? '—'}</dd>
        </div>
        <div>
          <dt className="text-ink-4 dark:text-white/40">Slug</dt>
          <dd className="text-ink-1 font-mono mt-0.5 dark:text-white">{workspace?.slug ?? '—'}</dd>
        </div>
        <div>
          <dt className="text-ink-4 dark:text-white/40">Workspace ID</dt>
          <dd className="text-ink-1 font-mono mt-0.5 truncate dark:text-white">{workspace?.id ?? workspaceId}</dd>
        </div>
        <div>
          <dt className="text-ink-4 dark:text-white/40">Created</dt>
          <dd className="text-ink-1 mt-0.5 dark:text-white">
            {workspace?.created_at ? new Date(workspace.created_at).toLocaleDateString() : '—'}
          </dd>
        </div>
      </dl>

      <div className="pt-4 border-t border-ink-5/20 dark:border-white/10">
        <p className="text-[11px] text-ink-4 dark:text-white/40">
          Quick links: <a className="text-brand-600 hover:text-brand-700" href={`/workspaces/${workspaceId}/members`}>Members</a> ·{' '}
          <a className="text-brand-600 hover:text-brand-700" href={`/workspaces/${workspaceId}/users`}>User management</a> ·{' '}
          <a className="text-brand-600 hover:text-brand-700" href={`/workspaces/${workspaceId}/audit`}>Activity log</a>
        </p>
      </div>
    </section>
  )
}
