import { useEffect, useMemo, useState } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import { useParams, useSearchParams } from 'react-router-dom'
import { Plus, Presentation, Trash2 } from 'lucide-react'
import { Tldraw, getSnapshot, loadSnapshot, type Editor } from 'tldraw'
import 'tldraw/tldraw.css'

import { api } from '../../lib/api'
import { queryClient } from '../../lib/queryClient'
import { rooms } from '../../lib/ws'
import { useWsEvent, useWsRooms } from '../../hooks/useWebSocket'
import { Button, Input, Modal, PageSpinner } from '../../components/ui'
import { cn } from '../../lib/utils'
import type { Whiteboard } from '../../types'

export function WhiteboardsPage() {
  const { workspaceId } = useParams<{ workspaceId: string }>()
  const [params, setParams] = useSearchParams()
  const activeId = params.get('id') ?? ''
  const [showCreate, setShowCreate] = useState(false)

  const { data: boards = [], isLoading } = useQuery({
    queryKey: ['whiteboards', workspaceId],
    queryFn: () => api.get(`workspaces/${workspaceId}/whiteboards`).json<Whiteboard[]>(),
    enabled: !!workspaceId,
  })

  useWsRooms(workspaceId ? [rooms.workspace(workspaceId)] : [])
  useWsEvent('whiteboard.created', () => queryClient.invalidateQueries({ queryKey: ['whiteboards', workspaceId] }))
  useWsEvent('whiteboard.deleted', () => queryClient.invalidateQueries({ queryKey: ['whiteboards', workspaceId] }))

  const active = boards.find((b) => b.id === activeId) ?? boards[0]

  return (
    <div className="flex h-full bg-canvas">
      <aside className="w-56 shrink-0 border-r border-ink-5/20 bg-surface/50 flex flex-col">
        <div className="flex items-center justify-between px-4 py-3 border-b border-ink-5/20">
          <h2 className="text-sm font-semibold text-ink-1 flex items-center gap-1.5">
            <Presentation className="w-3.5 h-3.5 text-brand-500" />
            Whiteboards
          </h2>
          <button
            onClick={() => setShowCreate(true)}
            className="h-7 w-7 flex items-center justify-center rounded-lg text-ink-4 hover:text-brand-600 hover:bg-brand-50"
            title="New whiteboard"
          >
            <Plus className="w-3.5 h-3.5" />
          </button>
        </div>
        <div className="flex-1 overflow-y-auto p-1">
          {isLoading && <PageSpinner />}
          {boards.map((b) => (
            <button
              key={b.id}
              onClick={() => setParams({ id: b.id }, { replace: true })}
              className={cn(
                'block w-full text-left px-3 py-1.5 rounded text-xs truncate transition-colors',
                active?.id === b.id ? 'bg-brand-50 text-brand-700 font-semibold' : 'text-ink-2 hover:bg-ink-1/5',
              )}
            >
              {b.name}
            </button>
          ))}
          {!isLoading && boards.length === 0 && (
            <p className="text-xs text-ink-4 px-3 py-4">No whiteboards yet.</p>
          )}
        </div>
      </aside>

      <main className="flex-1 flex flex-col min-w-0">
        {active && workspaceId ? (
          <BoardEditor key={active.id} board={active} workspaceId={workspaceId} />
        ) : (
          <div className="h-full flex items-center justify-center text-sm text-ink-4">
            Create a whiteboard to get started.
          </div>
        )}
      </main>

      {showCreate && workspaceId && (
        <CreateBoardModal
          workspaceId={workspaceId}
          onCreated={(b) => setParams({ id: b.id }, { replace: true })}
          onClose={() => setShowCreate(false)}
        />
      )}
    </div>
  )
}

function BoardEditor({ board, workspaceId }: { board: Whiteboard; workspaceId: string }) {
  const [editor, setEditor] = useState<Editor | null>(null)

  const update = useMutation({
    mutationFn: (snapshot: unknown) =>
      api.patch(`whiteboards/${board.id}`, { json: { snapshot } }).json<Whiteboard>(),
    onSuccess: (updated) => {
      queryClient.setQueryData<Whiteboard[]>(['whiteboards', workspaceId], (prev) =>
        (prev ?? []).map((b) => (b.id === updated.id ? updated : b)),
      )
    },
  })

  const del = useMutation({
    mutationFn: () => api.delete(`whiteboards/${board.id}`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['whiteboards', workspaceId] }),
  })

  // Debounce outgoing snapshot saves so every keystroke/drag doesn't trigger a PATCH.
  const debouncer = useMemo(
    () => ({ current: null as ReturnType<typeof setTimeout> | null }),
    [board.id],
  )
  useEffect(() => () => { if (debouncer.current) clearTimeout(debouncer.current) }, [debouncer])

  // Feed the persisted snapshot into the editor once it mounts.
  useEffect(() => {
    if (!editor) return
    const snap = board.snapshot as Parameters<typeof loadSnapshot>[1] | null | undefined
    if (snap && typeof snap === 'object') {
      try {
        loadSnapshot(editor.store, snap)
      } catch {
        // ignore incompatible snapshots (format changes across tldraw versions)
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [editor])

  // Debounced save on any store change.
  useEffect(() => {
    if (!editor) return
    const off = editor.store.listen(() => {
      if (debouncer.current) clearTimeout(debouncer.current)
      debouncer.current = setTimeout(() => {
        const snapshot = getSnapshot(editor.store)
        update.mutate(snapshot)
      }, 900)
    }, { source: 'user', scope: 'document' })
    return () => off()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [editor])

  return (
    <>
      <header className="flex items-center justify-between px-6 py-3 border-b border-ink-5/20 bg-surface/50 shrink-0">
        <div>
          <h1 className="text-sm font-semibold text-ink-1">{board.name}</h1>
          <p className="text-[11px] text-ink-4">v{board.version} · auto-saves every second after you stop drawing</p>
        </div>
        <Button
          variant="secondary"
          size="sm"
          onClick={() => { if (confirm(`Delete "${board.name}"?`)) del.mutate() }}
        >
          <Trash2 className="w-3.5 h-3.5 mr-1.5" />
          Delete
        </Button>
      </header>
      <div className="flex-1 relative">
        <Tldraw onMount={setEditor} />
      </div>
    </>
  )
}

function CreateBoardModal({
  workspaceId, onCreated, onClose,
}: {
  workspaceId: string
  onCreated: (b: Whiteboard) => void
  onClose: () => void
}) {
  const [name, setName] = useState('')
  const create = useMutation({
    mutationFn: () => api.post(`workspaces/${workspaceId}/whiteboards`, { json: { name } }).json<Whiteboard>(),
    onSuccess: (b) => {
      queryClient.invalidateQueries({ queryKey: ['whiteboards', workspaceId] })
      onCreated(b)
      onClose()
    },
  })
  return (
    <Modal open title="New whiteboard" onClose={onClose}>
      <div className="space-y-3">
        <Input label="Name" placeholder="Roadmap sketch" value={name} onChange={(e) => setName(e.target.value)} autoFocus />
        <div className="flex justify-end gap-2 pt-1">
          <Button variant="secondary" onClick={onClose}>Cancel</Button>
          <Button onClick={() => create.mutate()} disabled={!name.trim() || create.isPending} loading={create.isPending}>
            Create
          </Button>
        </div>
      </div>
    </Modal>
  )
}
