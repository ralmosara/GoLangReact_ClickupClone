import { useQuery, useMutation } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { useState } from 'react'
import { api } from '../../lib/api'
import { queryClient } from '../../lib/queryClient'
import { useUIStore } from '../../store/ui'
import { Modal, Button, Input } from '../../components/ui'
import { PageSpinner } from '../../components/ui'
import type { Workspace } from '../../types'

function CreateWorkspaceModal({ onClose }: { onClose: () => void }) {
  const [name, setName] = useState('')
  const mut = useMutation({
    mutationFn: (body: { name: string }) =>
      api.post('workspaces', { json: body }).json<Workspace>(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['workspaces'] })
      onClose()
    },
  })

  return (
    <Modal open title="New Workspace" description="Create a workspace to organize your projects." onClose={onClose}>
      <div className="space-y-4">
        <Input
          label="Workspace name"
          placeholder="e.g. Acme Inc."
          value={name}
          onChange={(e) => setName(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && name.trim() && mut.mutate({ name })}
          autoFocus
        />
        {mut.error && (
          <p className="text-red-500 text-xs">{String(mut.error)}</p>
        )}
        <div className="flex justify-end gap-2 pt-1">
          <Button variant="secondary" onClick={onClose}>Cancel</Button>
          <Button
            onClick={() => mut.mutate({ name })}
            disabled={!name.trim() || mut.isPending}
            loading={mut.isPending}
          >
            Create workspace
          </Button>
        </div>
      </div>
    </Modal>
  )
}

export function WorkspacesPage() {
  const navigate = useNavigate()
  const setActiveWorkspace = useUIStore((s) => s.setActiveWorkspace)
  const [showCreate, setShowCreate] = useState(false)

  const { data: workspaces, isLoading } = useQuery({
    queryKey: ['workspaces'],
    queryFn: () => api.get('workspaces').json<Workspace[]>(),
  })

  const handleSelect = (w: Workspace) => {
    setActiveWorkspace(w.id)
    navigate(`/workspaces/${w.id}`)
  }

  return (
    <div className="min-h-screen bg-canvas-gradient flex flex-col">
      {/* Top bar */}
      <header className="px-8 py-5 flex items-center justify-between border-b border-ink-5/20 bg-surface/60 backdrop-blur-sm">
        <div className="flex items-center gap-2.5">
          <div className="w-7 h-7 rounded-lg bg-brand-gradient flex items-center justify-center shadow-button">
            <svg className="w-4 h-4 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2.5} d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
            </svg>
          </div>
          <span className="text-ink-1 font-semibold text-sm tracking-wide">ClickUp</span>
        </div>
        <Button size="sm" onClick={() => setShowCreate(true)}>
          + New Workspace
        </Button>
      </header>

      {/* Content */}
      <main className="flex-1 px-8 py-12 max-w-3xl mx-auto w-full">
        <div className="mb-10 animate-slide-up">
          <h1 className="text-2xl font-bold text-ink-1 mb-1">Your Workspaces</h1>
          <p className="text-sm text-ink-3">Select a workspace to continue, or create a new one.</p>
        </div>

        {isLoading && <PageSpinner />}

        {!isLoading && (
          <div className="grid gap-3 animate-fade-in">
            {(workspaces ?? []).map((w, i) => (
              <button
                key={w.id}
                onClick={() => handleSelect(w)}
                className="group w-full text-left bg-surface border border-ink-5/30 rounded-2xl p-5 shadow-card hover:shadow-card-hover hover:border-brand-400/50 transition-all duration-200 flex items-center gap-4"
                style={{ animationDelay: `${i * 40}ms` }}
              >
                {/* Avatar */}
                <div className="w-10 h-10 rounded-xl bg-brand-gradient flex items-center justify-center text-white font-bold text-sm shadow-button shrink-0">
                  {w.name.charAt(0).toUpperCase()}
                </div>
                <div className="flex-1 min-w-0">
                  <p className="font-semibold text-ink-1 group-hover:text-brand-600 transition-colors truncate">
                    {w.name}
                  </p>
                  <p className="text-xs text-ink-4 mt-0.5 truncate">{w.slug}</p>
                </div>
                <svg
                  className="w-4 h-4 text-ink-4 group-hover:text-brand-500 group-hover:translate-x-0.5 transition-all duration-150 shrink-0"
                  fill="none" stroke="currentColor" viewBox="0 0 24 24"
                >
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                </svg>
              </button>
            ))}

            {workspaces?.length === 0 && (
              <div className="text-center py-20 bg-surface/60 rounded-2xl border border-ink-5/20">
                <div className="w-12 h-12 rounded-2xl bg-brand-50 border border-brand-100 flex items-center justify-center mx-auto mb-4">
                  <svg className="w-6 h-6 text-brand-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4" />
                  </svg>
                </div>
                <p className="text-ink-2 font-medium">No workspaces yet</p>
                <p className="text-sm text-ink-4 mt-1 mb-6">Create your first workspace to get started.</p>
                <Button onClick={() => setShowCreate(true)}>Create workspace</Button>
              </div>
            )}
          </div>
        )}
      </main>

      {showCreate && <CreateWorkspaceModal onClose={() => setShowCreate(false)} />}
    </div>
  )
}
