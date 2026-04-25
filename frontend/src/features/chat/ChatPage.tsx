import { Fragment, useEffect, useMemo, useRef, useState } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import { useParams, useSearchParams } from 'react-router-dom'
import {
  Hash, Plus, Send, MessageSquare, X, Search, Users, Paperclip, Smile,
  Copy, Edit3, Trash2, MoreHorizontal, CornerDownRight, ArrowDown, Pencil, Check,
} from 'lucide-react'

import { api } from '../../lib/api'
import { queryClient } from '../../lib/queryClient'
import { rooms } from '../../lib/ws'
import { useWsEvent, useWsRooms } from '../../hooks/useWebSocket'
import { Button, Input, Modal, PageSpinner } from '../../components/ui'
import { useAuthStore } from '../../store/auth'
import { cn } from '../../lib/utils'
import type { Channel, ChatMessage, Member } from '../../types'

/* -------------------------------------------------------------------------- */
/* Page shell                                                                 */
/* -------------------------------------------------------------------------- */

export function ChatPage() {
  const { workspaceId } = useParams<{ workspaceId: string }>()
  const [searchParams, setSearchParams] = useSearchParams()
  const activeId = searchParams.get('channel') ?? ''

  const [showCreate, setShowCreate] = useState(false)
  const [threadParent, setThreadParent] = useState<ChatMessage | null>(null)

  const { data: channels = [], isLoading } = useQuery({
    queryKey: ['channels', workspaceId],
    queryFn: () => api.get(`workspaces/${workspaceId}/channels`).json<Channel[]>(),
    enabled: !!workspaceId,
  })

  useWsRooms(workspaceId ? [rooms.workspace(workspaceId)] : [])
  useWsEvent('channel.created', () => queryClient.invalidateQueries({ queryKey: ['channels', workspaceId] }))
  useWsEvent('channel.updated', () => queryClient.invalidateQueries({ queryKey: ['channels', workspaceId] }))
  useWsEvent('channel.deleted', () => queryClient.invalidateQueries({ queryKey: ['channels', workspaceId] }))

  const active = channels.find((c) => c.id === activeId)
  const selectChannel = (id: string) => {
    const p = new URLSearchParams(searchParams)
    p.set('channel', id)
    setSearchParams(p, { replace: true })
    setThreadParent(null)
  }

  return (
    <div className="flex h-full bg-canvas">
      <ChannelSidebar
        channels={channels}
        activeId={activeId}
        loading={isLoading}
        onSelect={selectChannel}
        onCreate={() => setShowCreate(true)}
      />

      <main className="flex-1 flex min-w-0">
        {active ? (
          <div className="flex-1 flex min-w-0">
            <ChannelView
              channel={active}
              workspaceId={workspaceId!}
              onOpenThread={(msg) => setThreadParent(msg)}
            />
            {threadParent && (
              <ThreadPanel
                parent={threadParent}
                workspaceId={workspaceId!}
                onClose={() => setThreadParent(null)}
              />
            )}
          </div>
        ) : (
          <EmptyPicker />
        )}
      </main>

      {showCreate && workspaceId && (
        <CreateChannelModal
          workspaceId={workspaceId}
          onClose={() => setShowCreate(false)}
          onCreated={(c) => selectChannel(c.id)}
        />
      )}
    </div>
  )
}

/* -------------------------------------------------------------------------- */
/* Channel sidebar                                                            */
/* -------------------------------------------------------------------------- */

function ChannelSidebar({
  channels, activeId, loading, onSelect, onCreate,
}: {
  channels: Channel[]
  activeId: string
  loading: boolean
  onSelect: (id: string) => void
  onCreate: () => void
}) {
  const [query, setQuery] = useState('')
  const filtered = useMemo(
    () => channels.filter((c) => c.name.toLowerCase().includes(query.toLowerCase())),
    [channels, query],
  )

  return (
    <aside className="w-64 shrink-0 border-r border-ink-5/20 bg-surface/60 flex flex-col">
      <div className="flex items-center justify-between px-4 h-12 border-b border-ink-5/20">
        <h2 className="text-[13px] font-semibold text-ink-1 tracking-tight">Channels</h2>
        <button
          onClick={onCreate}
          className="h-7 w-7 flex items-center justify-center rounded-lg text-ink-3 hover:text-brand-600 hover:bg-brand-50 transition-colors"
          title="New channel"
          aria-label="New channel"
        >
          <Plus className="w-3.5 h-3.5" />
        </button>
      </div>

      <div className="px-3 pt-3 pb-1">
        <div className="relative">
          <Search className="w-3.5 h-3.5 text-ink-4 absolute left-2.5 top-1/2 -translate-y-1/2" />
          <input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Find a channel"
            className="w-full h-8 pl-8 pr-2 text-[12px] bg-canvas/60 border border-ink-5/25 rounded-lg focus:outline-none focus:border-brand-400 focus:bg-surface focus:ring-2 focus:ring-brand-400/30 placeholder:text-ink-4 transition-all"
          />
        </div>
      </div>

      <nav className="flex-1 overflow-y-auto px-2 pb-3 pt-1">
        {loading && <div className="px-3 py-6"><PageSpinner /></div>}

        {!loading && channels.length === 0 && (
          <div className="px-3 py-6 text-center">
            <Hash className="w-6 h-6 text-ink-5 mx-auto mb-2" />
            <p className="text-[12px] text-ink-3 font-medium mb-1">No channels yet</p>
            <p className="text-[11px] text-ink-4">Create the first one to start chatting.</p>
          </div>
        )}

        {filtered.map((c) => (
          <button
            key={c.id}
            onClick={() => onSelect(c.id)}
            className={cn(
              'flex items-center gap-2 w-full h-8 px-2.5 rounded-md text-[13px] text-left transition-colors',
              activeId === c.id
                ? 'bg-brand-50 text-brand-700 font-semibold'
                : 'text-ink-2 hover:bg-ink-1/[0.04]',
            )}
          >
            <Hash className={cn('w-3.5 h-3.5 shrink-0', activeId === c.id ? 'text-brand-500' : 'text-ink-4')} />
            <span className="truncate">{c.name}</span>
          </button>
        ))}

        {!loading && channels.length > 0 && filtered.length === 0 && (
          <p className="text-[11px] text-ink-4 px-3 py-4">No channels match "{query}".</p>
        )}
      </nav>
    </aside>
  )
}

/* -------------------------------------------------------------------------- */
/* Channel view                                                               */
/* -------------------------------------------------------------------------- */

function ChannelView({
  channel, workspaceId, onOpenThread,
}: {
  channel: Channel
  workspaceId: string
  onOpenThread: (msg: ChatMessage) => void
}) {
  const user = useAuthStore((s) => s.user)

  const { data: messages = [], isLoading } = useQuery({
    queryKey: ['messages', channel.id],
    queryFn: () => api.get(`channels/${channel.id}/messages?limit=100`).json<ChatMessage[]>(),
  })

  const { data: members = [] } = useQuery({
    queryKey: ['workspace-members', workspaceId],
    queryFn: () => api.get(`workspaces/${workspaceId}/members`).json<Member[]>(),
  })
  const memberByID = useMemo(() => new Map(members.map((m) => [m.user_id, m])), [members])

  useWsRooms([rooms.channel(channel.id)])
  useWsEvent<ChatMessage>('message.created', (e) => {
    if (!e.payload || e.payload.channel_id === channel.id) {
      queryClient.invalidateQueries({ queryKey: ['messages', channel.id] })
    }
  })
  useWsEvent('message.updated', () => queryClient.invalidateQueries({ queryKey: ['messages', channel.id] }))
  useWsEvent('message.deleted', () => queryClient.invalidateQueries({ queryKey: ['messages', channel.id] }))

  // Messages come newest-first from the API; flip for oldest→newest display.
  const ordered = useMemo(() => messages.slice().reverse(), [messages])

  return (
    <div className="flex-1 flex flex-col min-w-0 bg-canvas">
      <ChannelHeader channel={channel} memberCount={members.length} />

      <MessageList
        messages={ordered}
        loading={isLoading}
        channelName={channel.name}
        memberByID={memberByID}
        onOpenThread={onOpenThread}
        currentUserID={user?.id}
      />

      <Composer
        channelID={channel.id}
        members={members}
        placeholder={`Message #${channel.name}`}
      />
    </div>
  )
}

function ChannelHeader({ channel, memberCount }: { channel: Channel; memberCount: number }) {
  return (
    <header className="flex items-center gap-3 px-6 h-14 border-b border-ink-5/20 bg-surface/80 backdrop-blur-sm shrink-0">
      <div className="flex items-center gap-2 min-w-0 flex-1">
        <div className="w-8 h-8 rounded-lg bg-brand-50 border border-brand-100 flex items-center justify-center shrink-0">
          <Hash className="w-4 h-4 text-brand-500" />
        </div>
        <div className="min-w-0">
          <h1 className="text-[14px] font-semibold text-ink-1 truncate leading-tight">{channel.name}</h1>
          {channel.topic ? (
            <p className="text-[11px] text-ink-4 truncate leading-tight mt-0.5">{channel.topic}</p>
          ) : (
            <p className="text-[11px] text-ink-5 truncate leading-tight mt-0.5 italic">Add a topic</p>
          )}
        </div>
      </div>

      <div className="flex items-center gap-1">
        <button
          className="inline-flex items-center gap-1.5 h-8 px-2.5 rounded-lg text-[12px] text-ink-3 hover:text-ink-1 hover:bg-ink-1/[0.05] transition-colors"
          title="Members"
        >
          <Users className="w-3.5 h-3.5" />
          <span className="tabular-nums">{memberCount}</span>
        </button>
        <button
          className="h-8 w-8 flex items-center justify-center rounded-lg text-ink-3 hover:text-ink-1 hover:bg-ink-1/[0.05] transition-colors"
          title="Search in channel"
          aria-label="Search in channel"
        >
          <Search className="w-4 h-4" />
        </button>
      </div>
    </header>
  )
}

/* -------------------------------------------------------------------------- */
/* Message list + grouping                                                    */
/* -------------------------------------------------------------------------- */

interface MessageGroup {
  id: string
  authorID: string | null
  firstAt: string
  messages: ChatMessage[]
}

interface RenderRow {
  kind: 'date' | 'group'
  key: string
  date?: string
  group?: MessageGroup
}

const GROUP_GAP_MS = 5 * 60 * 1000 // 5 minutes

function buildRows(messages: ChatMessage[]): RenderRow[] {
  const rows: RenderRow[] = []
  let lastDate = ''
  let current: MessageGroup | null = null
  let lastAt = 0

  for (const m of messages) {
    const at = new Date(m.created_at).getTime()
    const d = dayKey(m.created_at)

    // Day separator breaks the current group.
    if (d !== lastDate) {
      if (current) { rows.push({ kind: 'group', key: current.id, group: current }); current = null }
      rows.push({ kind: 'date', key: `date:${d}`, date: d })
      lastDate = d
    }

    const sameAuthor = current && current.authorID === (m.author_id ?? null)
    const withinGap = at - lastAt < GROUP_GAP_MS
    if (current && sameAuthor && withinGap) {
      current.messages.push(m)
    } else {
      if (current) rows.push({ kind: 'group', key: current.id, group: current })
      current = {
        id: `g:${m.id}`,
        authorID: m.author_id ?? null,
        firstAt: m.created_at,
        messages: [m],
      }
    }
    lastAt = at
  }
  if (current) rows.push({ kind: 'group', key: current.id, group: current })
  return rows
}

function dayKey(iso: string): string {
  const d = new Date(iso)
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}

function formatDayLabel(iso: string): string {
  const d = new Date(iso)
  const today = new Date()
  const yesterday = new Date(today)
  yesterday.setDate(today.getDate() - 1)
  const sameDay = (a: Date, b: Date) =>
    a.getFullYear() === b.getFullYear() && a.getMonth() === b.getMonth() && a.getDate() === b.getDate()

  if (sameDay(d, today)) return 'Today'
  if (sameDay(d, yesterday)) return 'Yesterday'
  const sixDays = 6 * 24 * 3600 * 1000
  if (today.getTime() - d.getTime() < sixDays && d < today) {
    return d.toLocaleDateString(undefined, { weekday: 'long' })
  }
  return d.toLocaleDateString(undefined, { weekday: 'long', month: 'long', day: 'numeric' })
}

function formatTimeShort(iso: string): string {
  const d = new Date(iso)
  return d.toLocaleTimeString(undefined, { hour: 'numeric', minute: '2-digit' })
}

function formatFullTimestamp(iso: string): string {
  const d = new Date(iso)
  return d.toLocaleString(undefined, {
    weekday: 'short', month: 'short', day: 'numeric',
    hour: 'numeric', minute: '2-digit',
  })
}

function MessageList({
  messages, loading, channelName, memberByID, onOpenThread, currentUserID,
}: {
  messages: ChatMessage[]
  loading: boolean
  channelName: string
  memberByID: Map<string, Member>
  onOpenThread?: (msg: ChatMessage) => void
  currentUserID?: string
}) {
  const scrollRef = useRef<HTMLDivElement>(null)
  const wasAtBottomRef = useRef(true)
  const [newCount, setNewCount] = useState(0)
  const prevLenRef = useRef(messages.length)

  // Track bottom-ness so we only auto-scroll when the user is already following.
  useEffect(() => {
    const el = scrollRef.current
    if (!el) return
    const handle = () => {
      const nearBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 80
      wasAtBottomRef.current = nearBottom
      if (nearBottom) setNewCount(0)
    }
    el.addEventListener('scroll', handle, { passive: true })
    return () => el.removeEventListener('scroll', handle)
  }, [])

  // Auto-scroll on new messages if user was near the bottom; else increment pill.
  useEffect(() => {
    const el = scrollRef.current
    if (!el) return
    const grew = messages.length > prevLenRef.current
    prevLenRef.current = messages.length
    if (!grew) return
    if (wasAtBottomRef.current) {
      el.scrollTo({ top: el.scrollHeight, behavior: 'smooth' })
    } else {
      setNewCount((n) => n + 1)
    }
  }, [messages])

  // Initial render: snap to the bottom.
  useEffect(() => {
    if (!loading && scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [loading])

  const rows = useMemo(() => buildRows(messages), [messages])

  const jumpToBottom = () => {
    const el = scrollRef.current
    if (!el) return
    el.scrollTo({ top: el.scrollHeight, behavior: 'smooth' })
    setNewCount(0)
  }

  return (
    <div className="flex-1 relative min-h-0">
      <div ref={scrollRef} className="absolute inset-0 overflow-y-auto px-6 py-4">
        {loading && (
          <div className="pt-16"><PageSpinner /></div>
        )}

        {!loading && messages.length === 0 && <ChannelIntro channelName={channelName} />}

        {!loading && messages.length > 0 && (
          <>
            <ChannelIntroCompact channelName={channelName} />
            {rows.map((row) =>
              row.kind === 'date' ? (
                <DateSeparator key={row.key} iso={row.date!} />
              ) : (
                <MessageGroupRow
                  key={row.key}
                  group={row.group!}
                  memberByID={memberByID}
                  onOpenThread={onOpenThread}
                  currentUserID={currentUserID}
                />
              ),
            )}
          </>
        )}
      </div>

      {newCount > 0 && (
        <button
          onClick={jumpToBottom}
          className="absolute bottom-3 left-1/2 -translate-x-1/2 inline-flex items-center gap-1.5 h-8 px-3 rounded-full bg-brand-600 text-white text-[12px] font-medium shadow-modal hover:bg-brand-700 transition-colors animate-fade-in"
        >
          <ArrowDown className="w-3.5 h-3.5" />
          {newCount} new message{newCount === 1 ? '' : 's'}
        </button>
      )}
    </div>
  )
}

function ChannelIntro({ channelName }: { channelName: string }) {
  return (
    <div className="max-w-xl py-8">
      <div className="w-12 h-12 rounded-2xl bg-brand-50 border border-brand-100 flex items-center justify-center mb-4">
        <Hash className="w-6 h-6 text-brand-500" />
      </div>
      <h2 className="text-xl font-bold text-ink-1 tracking-tight">
        Welcome to <span className="text-brand-600">#{channelName}</span>
      </h2>
      <p className="text-sm text-ink-3 mt-1 leading-relaxed">
        This is the very beginning of the channel. Start the conversation — you can @mention teammates,
        attach files, and reply in threads.
      </p>
    </div>
  )
}

function ChannelIntroCompact({ channelName }: { channelName: string }) {
  return (
    <div className="pb-4 mb-2">
      <div className="w-10 h-10 rounded-xl bg-brand-50 border border-brand-100 flex items-center justify-center mb-2">
        <Hash className="w-5 h-5 text-brand-500" />
      </div>
      <p className="text-sm font-semibold text-ink-1">
        Welcome to <span className="text-brand-600">#{channelName}</span>
      </p>
      <p className="text-xs text-ink-4 mt-0.5">This is the beginning of the channel.</p>
    </div>
  )
}

function DateSeparator({ iso }: { iso: string }) {
  return (
    <div className="relative my-3 flex items-center" role="separator">
      <div className="flex-1 h-px bg-ink-5/25" />
      <span className="px-3 text-[11px] font-semibold text-ink-3 bg-canvas tracking-wide">
        {formatDayLabel(iso)}
      </span>
      <div className="flex-1 h-px bg-ink-5/25" />
    </div>
  )
}

function MessageGroupRow({
  group, memberByID, onOpenThread, currentUserID,
}: {
  group: MessageGroup
  memberByID: Map<string, Member>
  onOpenThread?: (msg: ChatMessage) => void
  currentUserID?: string
}) {
  const author = group.authorID ? memberByID.get(group.authorID) : undefined
  const displayName = author?.name || author?.email || 'Unknown'
  const initial = (displayName || '?').slice(0, 1).toUpperCase()

  return (
    <div className="pt-3 first:pt-0">
      {group.messages.map((m, i) => (
        <MessageItem
          key={m.id}
          message={m}
          isFirstInGroup={i === 0}
          authorName={displayName}
          authorInitial={initial}
          onOpenThread={onOpenThread}
          currentUserID={currentUserID}
        />
      ))}
    </div>
  )
}

function MessageItem({
  message, isFirstInGroup, authorName, authorInitial, onOpenThread, currentUserID,
}: {
  message: ChatMessage
  isFirstInGroup: boolean
  authorName: string
  authorInitial: string
  onOpenThread?: (msg: ChatMessage) => void
  currentUserID?: string
}) {
  const [editing, setEditing] = useState(false)
  const [draft, setDraft] = useState(message.body)
  const deleted = !!message.deleted_at
  const isOwn = !!currentUserID && message.author_id === currentUserID

  const edit = useMutation({
    mutationFn: () => api.patch(`messages/${message.id}`, { json: { body: draft } }).json<ChatMessage>(),
    onSuccess: () => {
      setEditing(false)
      queryClient.invalidateQueries({ queryKey: ['messages', message.channel_id] })
    },
  })

  const del = useMutation({
    mutationFn: () => api.delete(`messages/${message.id}`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['messages', message.channel_id] }),
  })

  if (deleted) {
    return (
      <div className={cn('group relative flex items-start gap-3', isFirstInGroup ? 'pt-0' : 'pt-0.5')}>
        <div className="w-9 shrink-0" />
        <p className="text-[12px] text-ink-5 italic py-1">
          Message deleted
        </p>
      </div>
    )
  }

  return (
    <div
      className={cn(
        'group relative flex items-start gap-3 px-2 -mx-2 rounded-lg',
        'hover:bg-ink-1/[0.025] transition-colors',
        isFirstInGroup ? 'pt-0' : 'pt-0.5',
      )}
    >
      {/* Gutter: avatar on first-in-group, hover-only timestamp on continuations */}
      {isFirstInGroup ? (
        <div className="w-9 h-9 rounded-lg bg-brand-gradient text-[13px] font-semibold text-white flex items-center justify-center shrink-0 shadow-card">
          {authorInitial}
        </div>
      ) : (
        <div className="w-9 shrink-0 flex items-start justify-center pt-1">
          <span className="text-[10px] text-ink-5 opacity-0 group-hover:opacity-100 tabular-nums transition-opacity">
            {formatTimeShort(message.created_at)}
          </span>
        </div>
      )}

      <div className="flex-1 min-w-0 py-0.5">
        {isFirstInGroup && (
          <div className="flex items-baseline gap-2 mb-0.5">
            <span className="text-[13.5px] font-semibold text-ink-1 leading-tight">{authorName}</span>
            <span
              className="text-[11px] text-ink-4 tabular-nums"
              title={formatFullTimestamp(message.created_at)}
            >
              {formatTimeShort(message.created_at)}
            </span>
          </div>
        )}

        {editing ? (
          <div className="mt-0.5">
            <textarea
              autoFocus
              className="w-full text-[13.5px] bg-surface border border-brand-400 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-brand-400/40 resize-none"
              rows={Math.max(1, draft.split('\n').length)}
              value={draft}
              onChange={(e) => setDraft(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' && !e.shiftKey) {
                  e.preventDefault()
                  if (draft.trim() && draft !== message.body) edit.mutate()
                  else setEditing(false)
                }
                if (e.key === 'Escape') { setDraft(message.body); setEditing(false) }
              }}
            />
            <div className="flex items-center gap-2 mt-1.5">
              <Button
                size="xs"
                onClick={() => (draft.trim() && draft !== message.body ? edit.mutate() : setEditing(false))}
              >
                <Check className="w-3 h-3 mr-1" />
                Save
              </Button>
              <button
                onClick={() => { setDraft(message.body); setEditing(false) }}
                className="text-[11px] text-ink-4 hover:text-ink-2"
              >
                Cancel
              </button>
              <span className="text-[10px] text-ink-5 ml-auto">Enter to save · Esc to cancel</span>
            </div>
          </div>
        ) : (
          <p className="text-[13.5px] text-ink-1 leading-[1.55] whitespace-pre-wrap break-words">
            {renderRichBody(message.body)}
            {message.edited_at && (
              <span className="text-[10px] text-ink-4 ml-1.5" title={formatFullTimestamp(message.edited_at)}>
                (edited)
              </span>
            )}
          </p>
        )}
      </div>

      {!editing && (
        <HoverToolbar
          onReplyInThread={onOpenThread ? () => onOpenThread(message) : undefined}
          onCopy={() => navigator.clipboard?.writeText(message.body)}
          onEdit={isOwn ? () => setEditing(true) : undefined}
          onDelete={isOwn ? () => {
            if (confirm('Delete this message?')) del.mutate()
          } : undefined}
        />
      )}
    </div>
  )
}

function HoverToolbar({
  onReplyInThread, onCopy, onEdit, onDelete,
}: {
  onReplyInThread?: () => void
  onCopy?: () => void
  onEdit?: () => void
  onDelete?: () => void
}) {
  const [menuOpen, setMenuOpen] = useState(false)

  return (
    <div
      className={cn(
        'absolute -top-3 right-3 opacity-0 group-hover:opacity-100 focus-within:opacity-100',
        'transition-opacity z-10',
      )}
    >
      <div className="inline-flex items-center bg-surface border border-ink-5/30 rounded-lg shadow-card overflow-hidden">
        <ToolbarButton title="Add reaction (coming soon)" disabled>
          <Smile className="w-3.5 h-3.5" />
        </ToolbarButton>
        {onReplyInThread && (
          <ToolbarButton title="Reply in thread" onClick={onReplyInThread}>
            <CornerDownRight className="w-3.5 h-3.5" />
          </ToolbarButton>
        )}
        <div className="relative">
          <ToolbarButton title="More" onClick={() => setMenuOpen((v) => !v)}>
            <MoreHorizontal className="w-3.5 h-3.5" />
          </ToolbarButton>
          {menuOpen && (
            <>
              <div className="fixed inset-0 z-10" onClick={() => setMenuOpen(false)} />
              <div className="absolute top-full right-0 mt-1 bg-surface border border-ink-5/30 rounded-lg shadow-modal py-1 min-w-[160px] z-20 animate-fade-in">
                {onCopy && (
                  <MenuItem icon={<Copy className="w-3.5 h-3.5" />} onClick={() => { onCopy(); setMenuOpen(false) }}>
                    Copy text
                  </MenuItem>
                )}
                {onEdit && (
                  <MenuItem icon={<Edit3 className="w-3.5 h-3.5" />} onClick={() => { onEdit(); setMenuOpen(false) }}>
                    Edit message
                  </MenuItem>
                )}
                {onDelete && (
                  <MenuItem
                    icon={<Trash2 className="w-3.5 h-3.5" />}
                    onClick={() => { onDelete(); setMenuOpen(false) }}
                    danger
                  >
                    Delete message
                  </MenuItem>
                )}
              </div>
            </>
          )}
        </div>
      </div>
    </div>
  )
}

function ToolbarButton({
  children, onClick, title, disabled,
}: {
  children: React.ReactNode
  onClick?: () => void
  title: string
  disabled?: boolean
}) {
  return (
    <button
      onClick={onClick}
      disabled={disabled}
      title={title}
      aria-label={title}
      className={cn(
        'h-7 w-7 flex items-center justify-center text-ink-3',
        disabled
          ? 'opacity-40 cursor-not-allowed'
          : 'hover:text-ink-1 hover:bg-ink-1/[0.05] active:scale-95 transition-all',
      )}
    >
      {children}
    </button>
  )
}

function MenuItem({
  icon, children, onClick, danger,
}: {
  icon: React.ReactNode
  children: React.ReactNode
  onClick: () => void
  danger?: boolean
}) {
  return (
    <button
      onClick={onClick}
      className={cn(
        'flex items-center gap-2 w-full px-3 py-1.5 text-[12.5px] text-left transition-colors',
        danger ? 'text-red-600 hover:bg-red-50' : 'text-ink-2 hover:bg-ink-1/[0.04]',
      )}
    >
      {icon}
      {children}
    </button>
  )
}

/* -------------------------------------------------------------------------- */
/* Rich body: mentions + URLs                                                 */
/* -------------------------------------------------------------------------- */

// Matches @email@host style mentions and bare http(s) URLs.
const RICH_RE = /(@[A-Za-z0-9._+-]+@[A-Za-z0-9.-]+|https?:\/\/[^\s]+)/g

function renderRichBody(text: string): React.ReactNode {
  const parts = text.split(RICH_RE)
  return parts.map((p, i) => {
    if (!p) return null
    if (p.startsWith('@')) {
      return (
        <span
          key={i}
          className="inline-flex items-center px-1 py-[1px] rounded-[4px] bg-brand-50 text-brand-700 font-medium align-baseline"
        >
          {p}
        </span>
      )
    }
    if (p.startsWith('http://') || p.startsWith('https://')) {
      return (
        <a
          key={i}
          href={p}
          target="_blank"
          rel="noreferrer"
          className="text-brand-600 underline decoration-brand-200 hover:decoration-brand-400"
        >
          {p}
        </a>
      )
    }
    return <span key={i}>{p}</span>
  })
}

/* -------------------------------------------------------------------------- */
/* Composer                                                                   */
/* -------------------------------------------------------------------------- */

function Composer({
  channelID, members, parentMessageID, placeholder,
}: {
  channelID: string
  members: Member[]
  parentMessageID?: string
  placeholder: string
}) {
  const [body, setBody] = useState('')
  const textareaRef = useRef<HTMLTextAreaElement>(null)

  const post = useMutation({
    mutationFn: () =>
      api.post(`channels/${channelID}/messages`, {
        json: {
          body,
          parent_message_id: parentMessageID,
          mention_ids: extractMentions(body, members),
        },
      }).json<ChatMessage>(),
    onSuccess: () => {
      setBody('')
      if (parentMessageID) {
        queryClient.invalidateQueries({ queryKey: ['replies', parentMessageID] })
      } else {
        queryClient.invalidateQueries({ queryKey: ['messages', channelID] })
      }
    },
  })

  // Auto-grow textarea up to a max.
  useEffect(() => {
    const el = textareaRef.current
    if (!el) return
    el.style.height = 'auto'
    el.style.height = Math.min(el.scrollHeight, 160) + 'px'
  }, [body])

  const canSend = body.trim().length > 0 && !post.isPending

  return (
    <div className="px-6 pb-5 pt-3 shrink-0">
      <div
        className={cn(
          'relative bg-surface border border-ink-5/30 rounded-2xl shadow-card',
          'focus-within:border-brand-400 focus-within:ring-2 focus-within:ring-brand-400/30 transition-all',
        )}
      >
        <textarea
          ref={textareaRef}
          rows={1}
          className="block w-full bg-transparent text-[13.5px] text-ink-1 focus:outline-none resize-none px-4 pt-3 pb-11 placeholder:text-ink-4 leading-[1.55]"
          placeholder={placeholder}
          value={body}
          onChange={(e) => setBody(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && !e.shiftKey && canSend) {
              e.preventDefault()
              post.mutate()
            }
          }}
        />

        <div className="absolute left-2 right-2 bottom-1.5 flex items-center gap-1">
          <ComposerIcon title="Attach file (coming soon)" disabled>
            <Paperclip className="w-4 h-4" />
          </ComposerIcon>
          <ComposerIcon title="Emoji (coming soon)" disabled>
            <Smile className="w-4 h-4" />
          </ComposerIcon>

          <div className="flex-1" />

          <span className="text-[10px] text-ink-4 mr-2 hidden sm:inline">
            <kbd className="font-mono">Enter</kbd> to send ·{' '}
            <kbd className="font-mono">Shift+Enter</kbd> for new line
          </span>

          <button
            onClick={() => canSend && post.mutate()}
            disabled={!canSend}
            aria-label="Send message"
            className={cn(
              'h-8 w-8 flex items-center justify-center rounded-xl transition-all',
              canSend
                ? 'bg-brand-gradient text-white shadow-button hover:opacity-90 active:scale-95'
                : 'bg-ink-5/40 text-ink-4 cursor-not-allowed',
            )}
          >
            <Send className="w-3.5 h-3.5" />
          </button>
        </div>
      </div>
    </div>
  )
}

function ComposerIcon({
  children, title, disabled, onClick,
}: {
  children: React.ReactNode
  title: string
  disabled?: boolean
  onClick?: () => void
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      title={title}
      aria-label={title}
      className={cn(
        'h-8 w-8 flex items-center justify-center rounded-lg text-ink-3 transition-colors',
        disabled
          ? 'opacity-40 cursor-not-allowed'
          : 'hover:text-ink-1 hover:bg-ink-1/[0.05]',
      )}
    >
      {children}
    </button>
  )
}

function extractMentions(body: string, members: Member[]): string[] {
  const handles = Array.from(body.matchAll(/@([A-Za-z0-9._+-]+@[A-Za-z0-9.-]+)/g)).map((m) => m[1])
  return handles
    .map((h) => members.find((m) => m.email === h)?.user_id)
    .filter(Boolean) as string[]
}

/* -------------------------------------------------------------------------- */
/* Thread panel                                                               */
/* -------------------------------------------------------------------------- */

function ThreadPanel({
  parent, workspaceId, onClose,
}: {
  parent: ChatMessage
  workspaceId: string
  onClose: () => void
}) {
  const user = useAuthStore((s) => s.user)

  const { data: replies = [] } = useQuery({
    queryKey: ['replies', parent.id],
    queryFn: () => api.get(`messages/${parent.id}/replies?limit=100`).json<ChatMessage[]>(),
  })
  const { data: members = [] } = useQuery({
    queryKey: ['workspace-members', workspaceId],
    queryFn: () => api.get(`workspaces/${workspaceId}/members`).json<Member[]>(),
  })
  const memberByID = useMemo(() => new Map(members.map((m) => [m.user_id, m])), [members])

  useWsEvent<ChatMessage>('message.created', (e) => {
    if (e.payload?.parent_message_id === parent.id) {
      queryClient.invalidateQueries({ queryKey: ['replies', parent.id] })
    }
  })

  // Replies are newest-first from the server; display oldest→newest.
  const ordered = useMemo(() => replies.slice().reverse(), [replies])
  const rows = useMemo(() => buildRows(ordered), [ordered])

  return (
    <aside className="w-[400px] shrink-0 border-l border-ink-5/20 bg-surface flex flex-col animate-fade-in">
      <header className="flex items-center justify-between px-5 h-14 border-b border-ink-5/20 shrink-0">
        <div>
          <p className="text-[13px] font-semibold text-ink-1">Thread</p>
          <p className="text-[11px] text-ink-4 mt-0.5">
            {replies.length} {replies.length === 1 ? 'reply' : 'replies'}
          </p>
        </div>
        <button
          onClick={onClose}
          className="h-8 w-8 flex items-center justify-center rounded-lg text-ink-3 hover:text-ink-1 hover:bg-ink-1/[0.05] transition-colors"
          aria-label="Close thread"
        >
          <X className="w-4 h-4" />
        </button>
      </header>

      <div className="flex-1 overflow-y-auto px-4 py-4">
        {/* Parent message — rendered as a solo group so it gets the same polish */}
        <MessageGroupRow
          group={{
            id: `g:${parent.id}`,
            authorID: parent.author_id ?? null,
            firstAt: parent.created_at,
            messages: [parent],
          }}
          memberByID={memberByID}
          currentUserID={user?.id}
        />

        <div className="relative my-4 flex items-center" role="separator">
          <div className="flex-1 h-px bg-ink-5/25" />
          <span className="px-3 text-[11px] font-semibold text-ink-3 bg-surface tracking-wide">
            {replies.length} {replies.length === 1 ? 'reply' : 'replies'}
          </span>
          <div className="flex-1 h-px bg-ink-5/25" />
        </div>

        {rows.map((row) =>
          row.kind === 'date' ? (
            <DateSeparator key={row.key} iso={row.date!} />
          ) : (
            <MessageGroupRow
              key={row.key}
              group={row.group!}
              memberByID={memberByID}
              currentUserID={user?.id}
            />
          ),
        )}

        {replies.length === 0 && (
          <p className="text-[12px] text-ink-4 text-center py-6">Be the first to reply.</p>
        )}
      </div>

      <div className="border-t border-ink-5/20">
        <Composer
          channelID={parent.channel_id}
          members={members}
          parentMessageID={parent.id}
          placeholder="Reply to thread…"
        />
      </div>
    </aside>
  )
}

/* -------------------------------------------------------------------------- */
/* Empty picker + Create modal                                                */
/* -------------------------------------------------------------------------- */

function EmptyPicker() {
  return (
    <div className="flex-1 flex items-center justify-center text-center p-12">
      <div className="max-w-sm">
        <div className="w-16 h-16 rounded-2xl bg-brand-50 border border-brand-100 flex items-center justify-center mx-auto mb-4">
          <MessageSquare className="w-8 h-8 text-brand-500" />
        </div>
        <p className="text-[15px] font-semibold text-ink-1 mb-1">Pick a channel</p>
        <p className="text-[13px] text-ink-4 leading-relaxed">
          Channels are where your team communicates. They work best when organised around a topic — #design,
          #marketing, #oncall.
        </p>
      </div>
    </div>
  )
}

function CreateChannelModal({
  workspaceId, onClose, onCreated,
}: {
  workspaceId: string
  onClose: () => void
  onCreated: (c: Channel) => void
}) {
  const [name, setName] = useState('')
  const [topic, setTopic] = useState('')
  const create = useMutation({
    mutationFn: () =>
      api.post(`workspaces/${workspaceId}/channels`, { json: { name, topic } }).json<Channel>(),
    onSuccess: (c) => {
      queryClient.invalidateQueries({ queryKey: ['channels', workspaceId] })
      onCreated(c)
      onClose()
    },
  })
  return (
    <Modal open title="Create a channel" description="Channels are for team conversations around a topic." onClose={onClose}>
      <div className="space-y-3">
        <div>
          <label className="block text-xs font-medium text-ink-2 mb-1">Name</label>
          <div className="relative">
            <Hash className="w-3.5 h-3.5 text-ink-4 absolute left-3 top-1/2 -translate-y-1/2" />
            <input
              autoFocus
              value={name}
              onChange={(e) => setName(e.target.value.toLowerCase().replace(/\s+/g, '-'))}
              onKeyDown={(e) => e.key === 'Enter' && name.trim() && create.mutate()}
              placeholder="e.g. design"
              className="w-full h-9 pl-8 pr-3 bg-surface border border-ink-5/40 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-brand-400/40 focus:border-brand-400 transition-all"
            />
          </div>
          <p className="text-[11px] text-ink-4 mt-1">Lowercase, no spaces. Hyphens are fine.</p>
        </div>
        <Input
          label="Topic (optional)"
          placeholder="Where the design team hangs out"
          value={topic}
          onChange={(e) => setTopic(e.target.value)}
        />
        <div className="flex justify-end gap-2 pt-1">
          <Button variant="secondary" onClick={onClose}>Cancel</Button>
          <Button onClick={() => create.mutate()} disabled={!name.trim() || create.isPending} loading={create.isPending}>
            <Pencil className="w-3.5 h-3.5 mr-1.5" />
            Create channel
          </Button>
        </div>
      </div>
    </Modal>
  )
}
