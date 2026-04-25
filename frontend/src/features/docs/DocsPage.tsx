import { useEffect, useMemo, useState } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import { useParams, useSearchParams } from 'react-router-dom'
import { FileText, Plus, ChevronRight, ChevronDown, Trash2 } from 'lucide-react'
import { EditorContent, useEditor } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'

import { api } from '../../lib/api'
import { queryClient } from '../../lib/queryClient'
import { rooms } from '../../lib/ws'
import { useWsEvent, useWsRooms } from '../../hooks/useWebSocket'
import { Button, Input, PageSpinner } from '../../components/ui'
import { cn, formatRelative } from '../../lib/utils'
import type { Doc } from '../../types'

export function DocsPage() {
  const { workspaceId } = useParams<{ workspaceId: string }>()
  const [searchParams, setSearchParams] = useSearchParams()
  const activeId = searchParams.get('doc') ?? ''

  const { data: docs = [], isLoading } = useQuery({
    queryKey: ['docs', workspaceId],
    queryFn: () => api.get(`workspaces/${workspaceId}/docs`).json<Doc[]>(),
    enabled: !!workspaceId,
  })

  useWsRooms(workspaceId ? [rooms.workspace(workspaceId)] : [])
  useWsEvent('doc.created', () => queryClient.invalidateQueries({ queryKey: ['docs', workspaceId] }))
  useWsEvent('doc.updated', () => queryClient.invalidateQueries({ queryKey: ['docs', workspaceId] }))
  useWsEvent('doc.deleted', () => {
    queryClient.invalidateQueries({ queryKey: ['docs', workspaceId] })
  })

  const create = useMutation({
    mutationFn: (parent?: string) =>
      api.post(`workspaces/${workspaceId}/docs`, { json: { parent_id: parent, title: 'Untitled' } }).json<Doc>(),
    onSuccess: (d) => {
      queryClient.invalidateQueries({ queryKey: ['docs', workspaceId] })
      const p = new URLSearchParams(searchParams)
      p.set('doc', d.id)
      setSearchParams(p, { replace: true })
    },
  })

  const tree = useMemo(() => buildTree(docs), [docs])
  const activeDoc = docs.find((d) => d.id === activeId) ?? null

  return (
    <div className="flex h-full bg-canvas">
      {/* Left nav */}
      <aside className="w-64 shrink-0 border-r border-ink-5/20 bg-surface/50 flex flex-col">
        <div className="flex items-center justify-between px-4 py-3 border-b border-ink-5/20">
          <h2 className="text-sm font-semibold text-ink-1">Docs</h2>
          <button
            onClick={() => create.mutate(undefined)}
            disabled={create.isPending}
            className="h-7 w-7 flex items-center justify-center rounded-lg text-ink-4 hover:text-brand-600 hover:bg-brand-50"
            title="New root doc"
          >
            <Plus className="w-3.5 h-3.5" />
          </button>
        </div>
        <div className="flex-1 overflow-y-auto p-2">
          {isLoading && <PageSpinner />}
          {!isLoading && tree.length === 0 && (
            <p className="text-xs text-ink-4 px-2 py-4">No docs yet. Click + to create one.</p>
          )}
          {tree.map((node) => (
            <TreeNode
              key={node.doc.id}
              node={node}
              activeId={activeId}
              depth={0}
              onSelect={(id) => {
                const p = new URLSearchParams(searchParams)
                p.set('doc', id)
                setSearchParams(p, { replace: true })
              }}
              onAddChild={(parentId) => create.mutate(parentId)}
            />
          ))}
        </div>
      </aside>

      <main className="flex-1 overflow-y-auto">
        {activeDoc && workspaceId ? (
          <DocEditor key={activeDoc.id} doc={activeDoc} workspaceId={workspaceId} />
        ) : (
          <div className="h-full flex items-center justify-center text-center p-12">
            <div>
              <FileText className="w-12 h-12 text-brand-400 mx-auto mb-3" />
              <p className="text-sm text-ink-2 font-semibold">Pick a doc from the sidebar</p>
              <p className="text-xs text-ink-4 mt-1">or create a new one with +</p>
            </div>
          </div>
        )}
      </main>
    </div>
  )
}

interface TreeNode { doc: Doc; children: TreeNode[] }

function buildTree(docs: Doc[]): TreeNode[] {
  const byParent = new Map<string, Doc[]>()
  for (const d of docs) {
    const key = d.parent_id ?? '__root'
    if (!byParent.has(key)) byParent.set(key, [])
    byParent.get(key)!.push(d)
  }
  const build = (id: string): TreeNode[] =>
    (byParent.get(id) ?? [])
      .sort((a, b) => a.order_index - b.order_index || a.created_at.localeCompare(b.created_at))
      .map((doc) => ({ doc, children: build(doc.id) }))
  return build('__root')
}

function TreeNode({
  node, activeId, depth, onSelect, onAddChild,
}: {
  node: TreeNode
  activeId: string
  depth: number
  onSelect: (id: string) => void
  onAddChild: (parentId: string) => void
}) {
  const [open, setOpen] = useState(true)
  const hasChildren = node.children.length > 0
  const active = activeId === node.doc.id
  return (
    <div>
      <div
        className={cn(
          'flex items-center gap-0.5 rounded-md cursor-pointer group',
          active ? 'bg-brand-50 text-brand-700' : 'text-ink-2 hover:bg-ink-1/5',
        )}
        style={{ paddingLeft: depth * 10 + 4 }}
      >
        <button
          onClick={() => setOpen((v) => !v)}
          className={cn('w-4 h-6 flex items-center justify-center text-ink-4 shrink-0', !hasChildren && 'invisible')}
        >
          {open ? <ChevronDown className="w-3 h-3" /> : <ChevronRight className="w-3 h-3" />}
        </button>
        <button
          onClick={() => onSelect(node.doc.id)}
          className="flex-1 h-7 flex items-center gap-1.5 min-w-0 text-xs text-left"
        >
          <span className="text-ink-4 shrink-0">{node.doc.icon ?? '📄'}</span>
          <span className={cn('truncate', active && 'font-semibold')}>{node.doc.title || 'Untitled'}</span>
        </button>
        <button
          onClick={() => onAddChild(node.doc.id)}
          className="opacity-0 group-hover:opacity-100 w-5 h-5 flex items-center justify-center text-ink-4 hover:text-brand-600 rounded"
          title="Add child"
        >
          <Plus className="w-3 h-3" />
        </button>
      </div>
      {open && hasChildren && (
        <div>
          {node.children.map((c) => (
            <TreeNode
              key={c.doc.id}
              node={c}
              activeId={activeId}
              depth={depth + 1}
              onSelect={onSelect}
              onAddChild={onAddChild}
            />
          ))}
        </div>
      )}
    </div>
  )
}

function DocEditor({ doc, workspaceId }: { doc: Doc; workspaceId: string }) {
  const [title, setTitle] = useState(doc.title)
  // Subscribe to this doc's room so other editors see our updates
  useWsRooms([rooms.doc(doc.id)])

  useEffect(() => setTitle(doc.title), [doc.title])

  const update = useMutation({
    mutationFn: (patch: { title?: string; content?: unknown; content_text?: string }) =>
      api.patch(`docs/${doc.id}`, { json: patch }).json<Doc>(),
    onSuccess: (updated) => {
      queryClient.setQueryData<Doc[]>(['docs', workspaceId], (prev) =>
        (prev ?? []).map((d) => (d.id === updated.id ? updated : d)),
      )
    },
  })

  const del = useMutation({
    mutationFn: () => api.delete(`docs/${doc.id}`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['docs', workspaceId] }),
  })

  const editor = useEditor({
    extensions: [StarterKit],
    content: doc.content ?? '',
    editorProps: {
      attributes: {
        class: 'prose prose-base max-w-none focus:outline-none text-ink-1 min-h-[60vh]',
      },
    },
    onUpdate: ({ editor }) => {
      // Debounce content save.
      if (debouncer.current) clearTimeout(debouncer.current)
      debouncer.current = setTimeout(() => {
        update.mutate({
          content: editor.getJSON(),
          content_text: editor.getText(),
        })
      }, 600)
    },
  })

  const debouncer = useDebouncerRef()

  useWsEvent('doc.updated', (e) => {
    if (e.entity_id === doc.id && editor && document.activeElement?.tagName !== 'INPUT' && !editor.isFocused) {
      // Refresh only when we aren't actively editing.
      queryClient.invalidateQueries({ queryKey: ['docs', workspaceId] })
    }
  })

  // Sync the editor when the doc itself changes (e.g., remote update while we were not focused)
  useEffect(() => {
    if (!editor) return
    if (!editor.isFocused) editor.commands.setContent(doc.content ?? '')
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [doc.version])

  return (
    <div className="max-w-3xl mx-auto px-10 py-10">
      <div className="flex items-center justify-between mb-2">
        <p className="text-[11px] text-ink-4">Updated {formatRelative(doc.updated_at)} · v{doc.version}</p>
        <Button
          variant="secondary"
          size="sm"
          onClick={() => {
            if (confirm('Delete this doc? Children are deleted too.')) del.mutate()
          }}
        >
          <Trash2 className="w-3.5 h-3.5 mr-1.5" />
          Delete
        </Button>
      </div>
      <input
        className="w-full text-3xl font-bold text-ink-1 bg-transparent focus:outline-none mb-3"
        value={title}
        onChange={(e) => setTitle(e.target.value)}
        onBlur={() => title !== doc.title && update.mutate({ title })}
        placeholder="Untitled"
      />
      <EditorContent editor={editor} />
    </div>
  )
}

// useDebouncerRef is a tiny ref-holding timer so handlers can debounce without useRef boilerplate.
function useDebouncerRef() {
  const ref = useMemo(() => ({ current: null as ReturnType<typeof setTimeout> | null }), [])
  useEffect(() => () => {
    if (ref.current) clearTimeout(ref.current)
  }, [ref])
  return ref
}
