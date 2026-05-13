import { useState } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import { ChevronDown, ChevronRight, MessageSquareReply, Pencil, SmilePlus, Trash2 } from 'lucide-react'

import { api } from '../../../lib/api'
import { queryClient } from '../../../lib/queryClient'
import { Button } from '../../../components/ui'
import { cn, formatRelative } from '../../../lib/utils'
import { useAuthStore } from '../../../store/auth'
import type { Comment, CommentReaction } from '../../../types'

// Quick-react palette. Six emojis covers the 80% case without showing a
// full picker — clicking the (+) opens the bigger picker for the rest.
const QUICK_EMOJIS = ['👍', '❤️', '🎉', '👀', '🚀', '😄']

export function CommentsThread({ taskId }: { taskId: string }) {
  const me = useAuthStore((s) => s.user)
  const [body, setBody] = useState('')

  const { data: comments = [] } = useQuery({
    queryKey: ['comments', taskId],
    queryFn: () => api.get(`tasks/${taskId}/comments`).json<Comment[]>().then((r) => r ?? []),
  })

  const post = useMutation({
    mutationFn: () =>
      api.post('comments', { json: { task_id: taskId, body } }).json<Comment>(),
    onSuccess: () => {
      setBody('')
      queryClient.invalidateQueries({ queryKey: ['comments', taskId] })
    },
  })

  return (
    <div>
      <div className="flex items-center gap-2 mb-5">
        <h3 className="text-sm font-bold text-ink-1">Comments</h3>
        {comments.length > 0 && (
          <span className="text-[10px] font-bold bg-ink-5/30 text-ink-3 rounded-full px-2 py-0.5">
            {comments.length}
          </span>
        )}
      </div>

      <ul className="space-y-4 mb-5">
        {comments.map((c) => (
          <CommentRow key={c.id} comment={c} taskId={taskId} viewerId={me?.id} />
        ))}
        {comments.length === 0 && (
          <li className="text-sm text-ink-4 py-4 text-center border-2 border-dashed border-ink-5/20 rounded-2xl">
            No comments yet. Start the conversation.
          </li>
        )}
      </ul>

      <div className="flex gap-2">
        <input
          className="flex-1 h-10 bg-canvas/50 border border-ink-5/30 rounded-xl px-4 text-sm text-ink-1 focus:outline-none focus:ring-2 focus:ring-brand-400/40 focus:border-brand-400 placeholder:text-ink-5 transition-all"
          placeholder="Write a comment…"
          value={body}
          onChange={(e) => setBody(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && !e.shiftKey && body.trim()) {
              e.preventDefault()
              post.mutate()
            }
          }}
        />
        <Button onClick={() => post.mutate()} disabled={!body.trim() || post.isPending}>
          Send
        </Button>
      </div>
    </div>
  )
}

function CommentRow({
  comment, taskId, viewerId,
}: {
  comment: Comment
  taskId: string
  viewerId?: string
}) {
  const [editing, setEditing] = useState(false)
  const [draft, setDraft] = useState(comment.body)
  const [replying, setReplying] = useState(false)
  const [replyBody, setReplyBody] = useState('')
  const [showReplies, setShowReplies] = useState(false)
  const [pickerOpen, setPickerOpen] = useState(false)

  const isAuthor = !!viewerId && viewerId === comment.author_id
  const isDeleted = !!comment.deleted_at

  const { data: replies = [] } = useQuery({
    queryKey: ['comment-replies', comment.id],
    queryFn: () => api.get(`comments/${comment.id}/replies`).json<Comment[]>().then((r) => r ?? []),
    enabled: showReplies && comment.reply_count > 0,
  })

  const updateMut = useMutation({
    mutationFn: () => api.patch(`comments/${comment.id}`, { json: { body: draft } }).json<Comment>(),
    onSuccess: () => {
      setEditing(false)
      queryClient.invalidateQueries({ queryKey: ['comments', taskId] })
    },
  })
  const deleteMut = useMutation({
    mutationFn: () => api.delete(`comments/${comment.id}`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['comments', taskId] }),
  })
  const replyMut = useMutation({
    mutationFn: () => api.post('comments', { json: {
      task_id: taskId, parent_comment_id: comment.id, body: replyBody,
    } }).json<Comment>(),
    onSuccess: () => {
      setReplyBody(''); setReplying(false); setShowReplies(true)
      queryClient.invalidateQueries({ queryKey: ['comments', taskId] })
      queryClient.invalidateQueries({ queryKey: ['comment-replies', comment.id] })
    },
  })

  const reactMut = useMutation({
    mutationFn: ({ emoji, on }: { emoji: string; on: boolean }) =>
      on
        ? api.post(`comments/${comment.id}/reactions`, { json: { emoji } })
        : api.delete(`comments/${comment.id}/reactions`, { json: { emoji } }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['comments', taskId] })
      // Replies live in their own cache entry so refetch them too.
      if (comment.parent_comment_id) {
        queryClient.invalidateQueries({ queryKey: ['comment-replies', comment.parent_comment_id] })
      }
    },
  })

  const toggleReact = (emoji: string) => {
    const existing = comment.reactions?.find((r) => r.emoji === emoji)
    reactMut.mutate({ emoji, on: !existing?.reacted })
    setPickerOpen(false)
  }

  return (
    <li className="flex gap-3">
      <div className="w-8 h-8 rounded-full bg-brand-100 flex items-center justify-center text-[10px] font-bold text-brand-600 shrink-0 uppercase">
        {comment.author_id?.slice(0, 1) || '?'}
      </div>

      <div className="flex-1 min-w-0">
        <div className={cn(
          'border rounded-2xl px-4 py-3',
          isDeleted
            ? 'bg-ink-5/10 border-dashed border-ink-5/30 text-ink-4 italic'
            : 'bg-canvas/60 border-ink-5/20',
        )}>
          {isDeleted ? (
            <p className="text-sm">[Comment removed]</p>
          ) : editing ? (
            <div className="space-y-2">
              <textarea
                className="w-full text-sm bg-surface border border-ink-5/30 rounded-lg p-2 focus:outline-none focus:ring-2 focus:ring-brand-400/40"
                value={draft}
                onChange={(e) => setDraft(e.target.value)}
                rows={3}
                autoFocus
              />
              <div className="flex gap-2 justify-end">
                <button
                  onClick={() => { setEditing(false); setDraft(comment.body) }}
                  className="text-xs text-ink-3 hover:text-ink-1"
                >
                  Cancel
                </button>
                <Button size="sm" onClick={() => updateMut.mutate()} disabled={!draft.trim() || updateMut.isPending}>
                  Save
                </Button>
              </div>
            </div>
          ) : (
            <p className="text-sm text-ink-1 leading-relaxed whitespace-pre-wrap">{comment.body}</p>
          )}
        </div>

        {/* Reactions row */}
        {!isDeleted && (
          <div className="flex flex-wrap items-center gap-1 mt-1.5 ml-1">
            {(comment.reactions ?? []).map((r) => (
              <button
                key={r.emoji}
                onClick={() => toggleReact(r.emoji)}
                className={cn(
                  'inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[12px] border transition-colors',
                  r.reacted
                    ? 'bg-brand-50 border-brand-200 text-brand-700'
                    : 'bg-surface border-ink-5/30 text-ink-2 hover:bg-ink-1/[0.04]',
                )}
              >
                <span>{r.emoji}</span>
                <span className="font-medium tabular-nums">{r.count}</span>
              </button>
            ))}
            <div className="relative">
              <button
                onClick={() => setPickerOpen((v) => !v)}
                className="inline-flex items-center justify-center w-6 h-6 rounded-full text-ink-4 hover:text-brand-600 hover:bg-brand-50"
                title="Add reaction"
                aria-label="Add reaction"
              >
                <SmilePlus className="w-3.5 h-3.5" />
              </button>
              {pickerOpen && (
                <div
                  className="absolute z-20 mt-1 left-0 bg-surface border border-ink-5/30 rounded-xl shadow-modal p-1 flex gap-0.5"
                  onMouseLeave={() => setPickerOpen(false)}
                >
                  {QUICK_EMOJIS.map((e) => (
                    <button
                      key={e}
                      onClick={() => toggleReact(e)}
                      className="h-7 w-7 grid place-items-center rounded-md hover:bg-ink-1/[0.04] text-base"
                    >
                      {e}
                    </button>
                  ))}
                </div>
              )}
            </div>
          </div>
        )}

        {/* Meta row */}
        <div className="flex items-center gap-3 text-[10px] text-ink-4 mt-1.5 ml-1">
          <span>{formatRelative(comment.created_at)}</span>
          {comment.edited_at && <span title={`Edited ${new Date(comment.edited_at).toLocaleString()}`}>· edited</span>}
          {!isDeleted && !comment.parent_comment_id && (
            <button
              onClick={() => setReplying((v) => !v)}
              className="inline-flex items-center gap-1 hover:text-brand-600"
            >
              <MessageSquareReply className="w-3 h-3" /> Reply
            </button>
          )}
          {!isDeleted && isAuthor && (
            <button
              onClick={() => setEditing(true)}
              className="inline-flex items-center gap-1 hover:text-brand-600"
            >
              <Pencil className="w-3 h-3" /> Edit
            </button>
          )}
          {!isDeleted && isAuthor && (
            <button
              onClick={() => { if (confirm('Remove this comment?')) deleteMut.mutate() }}
              className="inline-flex items-center gap-1 hover:text-red-500"
            >
              <Trash2 className="w-3 h-3" /> Delete
            </button>
          )}
        </div>

        {/* Replies disclosure */}
        {!comment.parent_comment_id && comment.reply_count > 0 && (
          <button
            onClick={() => setShowReplies((v) => !v)}
            className="mt-2 inline-flex items-center gap-1 text-[11px] font-medium text-brand-600 hover:text-brand-700"
          >
            {showReplies ? <ChevronDown className="w-3 h-3" /> : <ChevronRight className="w-3 h-3" />}
            {comment.reply_count} {comment.reply_count === 1 ? 'reply' : 'replies'}
          </button>
        )}

        {showReplies && replies.length > 0 && (
          <ul className="mt-3 ml-4 pl-4 border-l border-ink-5/30 space-y-3">
            {replies.map((rc) => (
              <CommentRow key={rc.id} comment={rc} taskId={taskId} viewerId={viewerId} />
            ))}
          </ul>
        )}

        {/* Reply composer */}
        {replying && (
          <div className="mt-2 flex gap-2">
            <input
              autoFocus
              className="flex-1 h-9 bg-canvas/50 border border-ink-5/30 rounded-lg px-3 text-sm focus:outline-none focus:ring-2 focus:ring-brand-400/40"
              placeholder="Reply…"
              value={replyBody}
              onChange={(e) => setReplyBody(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' && replyBody.trim()) replyMut.mutate()
                if (e.key === 'Escape') { setReplying(false); setReplyBody('') }
              }}
            />
            <Button size="sm" onClick={() => replyMut.mutate()} disabled={!replyBody.trim() || replyMut.isPending}>
              Reply
            </Button>
          </div>
        )}
      </div>
    </li>
  )
}
