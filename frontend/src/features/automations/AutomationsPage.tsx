import { useState } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import { useParams } from 'react-router-dom'
import { Plus, Zap, Trash2, Power } from 'lucide-react'
import { HTTPError } from 'ky'

import { api } from '../../lib/api'
import { queryClient } from '../../lib/queryClient'
import { Button, Input, Modal, PageSpinner } from '../../components/ui'
import { cn, formatRelative, PRIORITY_LABELS } from '../../lib/utils'

// extractErrorMessage turns ky errors into something a user can act on.
// ky's HTTPError stringifies as "Request failed with status code 400" by
// default, swallowing the JSON body — we need the server's actual message.
async function extractErrorMessage(err: unknown): Promise<string> {
  if (err instanceof HTTPError) {
    try {
      const body = await err.response.clone().json() as { error?: string }
      if (body?.error) return body.error
    } catch {
      // not JSON — fall through to text
    }
    try {
      const text = await err.response.clone().text()
      if (text) return text.slice(0, 240)
    } catch { /* swallow */ }
    return `${err.response.status} ${err.response.statusText}`
  }
  if (err instanceof Error) return err.message
  return String(err)
}
import type {
  Automation, AutomationAction, AutomationActionType, AutomationCondition, AutomationTrigger, AutomationTriggerType,
  Member, Space, Status, List as TaskList, Tag,
} from '../../types'

const TRIGGER_LABELS: Record<AutomationTriggerType, string> = {
  'task.created':         'Task is created',
  'task.status_changed':  'Task status changes',
  'task.assigned':        'Task is assigned',
  'task.completed':       'Task is completed',
  'task.due_soon':        'Task due date approaches',
}

const ACTION_LABELS: Record<AutomationActionType, string> = {
  change_status: 'Change status',
  assign_user:   'Assign user',
  add_tag:       'Add tag',
  add_comment:   'Post a comment',
  notify:        'Notify user',
}

export function AutomationsPage() {
  const { workspaceId } = useParams<{ workspaceId: string }>()
  const [showEditor, setShowEditor] = useState<Automation | null>(null)
  const [showCreate, setShowCreate] = useState(false)
  const [pageError, setPageError] = useState<string | null>(null)

  const { data: automations = [], isLoading, error: listError } = useQuery({
    queryKey: ['automations', workspaceId],
    queryFn: () => api.get(`workspaces/${workspaceId}/automations`).json<Automation[]>(),
    enabled: !!workspaceId,
    retry: false,
  })

  const toggle = useMutation({
    mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) =>
      api.patch(`automations/${id}`, { json: { enabled } }).json<Automation>(),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['automations', workspaceId] }),
    onError: async (err) => setPageError(await extractErrorMessage(err)),
  })

  const del = useMutation({
    mutationFn: (id: string) => api.delete(`automations/${id}`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['automations', workspaceId] }),
    onError: async (err) => setPageError(await extractErrorMessage(err)),
  })

  return (
    <div className="max-w-3xl mx-auto px-6 py-10 animate-slide-up">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-xl font-bold text-ink-1 flex items-center gap-2">
            <Zap className="w-5 h-5 text-brand-500" />
            Automations
          </h1>
          <p className="text-sm text-ink-4 mt-0.5">
            Rules that run on task events. Due-date triggers are scanned every minute.
          </p>
        </div>
        <Button onClick={() => setShowCreate(true)} size="sm">
          <Plus className="w-3.5 h-3.5 mr-1.5" />
          New automation
        </Button>
      </div>

      {pageError && (
        <div className="mb-4 bg-red-50 border border-red-200 rounded-lg px-3 py-2 flex items-start justify-between gap-2">
          <p className="text-red-600 text-xs font-medium">{pageError}</p>
          <button onClick={() => setPageError(null)} className="text-red-500 hover:text-red-700 text-xs">dismiss</button>
        </div>
      )}

      {listError && (
        <div className="mb-4 bg-amber-50 border border-amber-200 rounded-lg px-3 py-2">
          <p className="text-amber-700 text-xs font-medium">
            Couldn't load automations. Are you a member of this workspace?
          </p>
        </div>
      )}

      {isLoading && <PageSpinner />}

      {!isLoading && automations.length === 0 && (
        <div className="text-center py-16 bg-surface/50 border border-ink-5/20 rounded-2xl">
          <Zap className="w-10 h-10 text-brand-400 mx-auto mb-3" />
          <p className="font-semibold text-ink-2">No automations yet</p>
          <p className="text-sm text-ink-4 mt-1">Try: <em>When status changes to Done → notify the PM.</em></p>
        </div>
      )}

      {automations.length > 0 && (
        <div className="bg-surface border border-ink-5/30 rounded-2xl shadow-card divide-y divide-ink-5/20">
          {automations.map((a) => (
            <div key={a.id} className="p-4 flex items-start gap-3">
              <button
                onClick={() => toggle.mutate({ id: a.id, enabled: !a.enabled })}
                className={cn(
                  'mt-0.5 w-8 h-5 rounded-full p-0.5 transition-colors shrink-0',
                  a.enabled ? 'bg-brand-500' : 'bg-ink-5/40',
                )}
                title={a.enabled ? 'Enabled' : 'Disabled'}
              >
                <div className={cn('w-4 h-4 rounded-full bg-white transition-transform', a.enabled && 'translate-x-3')} />
              </button>

              <button
                onClick={() => setShowEditor(a)}
                className="flex-1 text-left min-w-0"
              >
                <p className="text-sm font-semibold text-ink-1 truncate">{a.name}</p>
                <p className="text-[11px] text-ink-4 mt-0.5">
                  <span className="font-medium">When:</span> {TRIGGER_LABELS[a.trigger.type] ?? a.trigger.type}
                  {' · '}
                  <span className="font-medium">Then:</span>{' '}
                  {a.actions.map((ac) => ACTION_LABELS[ac.type] ?? ac.type).join(' + ') || '(no actions)'}
                </p>
                <p className="text-[10px] text-ink-5 mt-1">
                  {a.run_count > 0
                    ? `Fired ${a.run_count}x · last run ${a.last_run_at ? formatRelative(a.last_run_at) : 'never'}`
                    : 'Never fired'}
                </p>
              </button>

              <button
                onClick={() => {
                  if (confirm(`Delete automation "${a.name}"?`)) del.mutate(a.id)
                }}
                className="text-ink-4 hover:text-red-500 p-2 rounded hover:bg-red-50"
              >
                <Trash2 className="w-3.5 h-3.5" />
              </button>
            </div>
          ))}
        </div>
      )}

      {showCreate && workspaceId && (
        <AutomationEditor
          workspaceId={workspaceId}
          onClose={() => setShowCreate(false)}
        />
      )}
      {showEditor && workspaceId && (
        <AutomationEditor
          workspaceId={workspaceId}
          automation={showEditor}
          onClose={() => setShowEditor(null)}
        />
      )}
    </div>
  )
}

/* ----- Editor modal -------------------------------------------------------- */

function AutomationEditor({
  workspaceId,
  automation,
  onClose,
}: {
  workspaceId: string
  automation?: Automation
  onClose: () => void
}) {
  const isEdit = !!automation
  const [name, setName] = useState(automation?.name ?? '')
  const [description, setDescription] = useState(automation?.description ?? '')
  const [listId, setListId] = useState<string | null>(automation?.list_id ?? null)
  const [trigger, setTrigger] = useState<AutomationTrigger>(automation?.trigger ?? { type: 'task.status_changed' })
  const [conditions, setConditions] = useState<AutomationCondition[]>(automation?.conditions ?? [])
  const [actions, setActions] = useState<AutomationAction[]>(
    automation?.actions && automation.actions.length > 0 ? automation.actions : [{ type: 'notify' }],
  )
  const [errorMsg, setErrorMsg] = useState<string | null>(null)

  // Ref data for dropdowns.
  const { data: spaces = [] } = useQuery({
    queryKey: ['spaces', workspaceId],
    queryFn: () => api.get(`workspaces/${workspaceId}/spaces`).json<Space[]>(),
  })
  const { data: lists = [] } = useQuery({
    queryKey: ['all-lists', workspaceId, spaces.map((s) => s.id).join(',')],
    queryFn: async () => {
      const results = await Promise.all(spaces.map((s) => api.get(`spaces/${s.id}/lists`).json<TaskList[]>()))
      return results.flat()
    },
    enabled: spaces.length > 0,
  })
  const { data: statuses = [] } = useQuery({
    queryKey: ['statuses-scoped', listId],
    queryFn: () => api.get(`lists/${listId}/statuses`).json<Status[]>(),
    enabled: !!listId,
  })
  const { data: members = [] } = useQuery({
    queryKey: ['workspace-members', workspaceId],
    queryFn: () => api.get(`workspaces/${workspaceId}/members`).json<Member[]>(),
  })
  const { data: tags = [] } = useQuery({
    queryKey: ['workspace-tags', workspaceId],
    queryFn: () => api.get(`workspaces/${workspaceId}/tags`).json<Tag[]>(),
  })

  // Light client-side validation. Mirrors what the backend rejects so users
  // see the failure inline before round-tripping.
  function validate(): string | null {
    if (!name.trim()) return 'Name is required.'
    if (trigger.type === 'task.status_changed' && trigger.to_status_id && !listId) {
      return 'Pick a list scope when targeting a specific status.'
    }
    if (actions.length === 0) return 'Add at least one action.'
    for (const a of actions) {
      if (a.type === 'change_status' && !a.status_id) return 'Each "change status" action needs a target status.'
      if (a.type === 'assign_user'  && !a.user_id)   return 'Each "assign user" action needs a member.'
      if (a.type === 'add_tag'      && !a.tag_id)    return 'Each "add tag" action needs a tag.'
      if (a.type === 'add_comment'  && !a.body?.trim()) return 'Each comment action needs a body.'
      if (a.type === 'notify'       && !a.user_id)   return 'Each "notify" action needs a recipient.'
    }
    return null
  }

  const save = useMutation({
    mutationFn: async () => {
      const v = validate()
      if (v) throw new Error(v)
      // On edit, distinguish "no scope change" from "switch to workspace-wide".
      // The backend treats list_id==undefined as untouched, so we send an
      // explicit clear_list_id when the user chose "Entire workspace" on a
      // previously list-scoped automation.
      const wasListScoped = !!automation?.list_id
      const body: Record<string, unknown> = {
        name: name.trim(),
        description,
        trigger,
        conditions,
        actions,
        enabled: automation?.enabled ?? true,
      }
      if (listId) {
        body.list_id = listId
      } else if (isEdit && wasListScoped) {
        body.clear_list_id = true
      }
      if (isEdit) {
        return api.patch(`automations/${automation!.id}`, { json: body }).json<Automation>()
      }
      return api.post(`workspaces/${workspaceId}/automations`, { json: body }).json<Automation>()
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['automations', workspaceId] })
      onClose()
    },
    onError: async (err) => {
      setErrorMsg(await extractErrorMessage(err))
    },
    onMutate: () => setErrorMsg(null),
  })

  return (
    <Modal
      open
      width="max-w-2xl"
      title={isEdit ? 'Edit automation' : 'New automation'}
      description="A rule has a trigger, optional conditions, and one or more actions."
      onClose={onClose}
    >
      <div className="space-y-4 max-h-[70vh] overflow-y-auto pr-1">
        <Input label="Name" placeholder="e.g. Urgent bugs notify PM" value={name} onChange={(e) => setName(e.target.value)} autoFocus />
        <div>
          <label className="block text-xs font-medium text-ink-2 mb-1">Description (optional)</label>
          <textarea
            className="w-full bg-surface border border-ink-5/40 rounded-xl px-3 py-2 text-sm h-16 resize-none focus:outline-none focus:ring-2 focus:ring-brand-400/40"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
          />
        </div>

        <div>
          <label className="block text-xs font-medium text-ink-2 mb-1">Scope</label>
          <select
            className="w-full h-9 bg-surface border border-ink-5/40 rounded-lg px-3 text-sm focus:outline-none focus:ring-2 focus:ring-brand-400/40"
            value={listId ?? ''}
            onChange={(e) => setListId(e.target.value || null)}
          >
            <option value="">Entire workspace</option>
            {lists.map((l) => <option key={l.id} value={l.id}>{l.name}</option>)}
          </select>
        </div>

        {/* Trigger */}
        <Section title="When (trigger)">
          <select
            className="w-full h-9 bg-surface border border-ink-5/40 rounded-lg px-3 text-sm"
            value={trigger.type}
            onChange={(e) => setTrigger({ type: e.target.value as AutomationTriggerType })}
          >
            {Object.entries(TRIGGER_LABELS).map(([v, l]) => (
              <option key={v} value={v}>{l}</option>
            ))}
          </select>
          {trigger.type === 'task.status_changed' && (
            <select
              className="mt-2 w-full h-9 bg-surface border border-ink-5/40 rounded-lg px-3 text-sm"
              value={trigger.to_status_id ?? ''}
              onChange={(e) => setTrigger({ ...trigger, to_status_id: e.target.value || undefined })}
              disabled={!listId}
            >
              <option value="">Any status (scope must be a list)</option>
              {statuses.map((s) => <option key={s.id} value={s.id}>to: {s.name}</option>)}
            </select>
          )}
          {trigger.type === 'task.assigned' && (
            <select
              className="mt-2 w-full h-9 bg-surface border border-ink-5/40 rounded-lg px-3 text-sm"
              value={trigger.user_id ?? ''}
              onChange={(e) => setTrigger({ ...trigger, user_id: e.target.value || undefined })}
            >
              <option value="">Anyone</option>
              {members.map((m) => <option key={m.user_id} value={m.user_id}>to: {m.name || m.email}</option>)}
            </select>
          )}
          {trigger.type === 'task.due_soon' && (
            <div className="mt-2 flex items-center gap-2 text-xs text-ink-3">
              <span>Fire when due within</span>
              <input
                type="number"
                min={1}
                max={30}
                value={trigger.lead_days ?? 1}
                onChange={(e) =>
                  setTrigger({ ...trigger, lead_days: Math.max(1, Number(e.target.value || 1)) })
                }
                className="w-16 h-8 bg-surface border border-ink-5/40 rounded-lg px-2 text-sm"
              />
              <span>day(s).</span>
              <span className="text-ink-5 ml-auto italic">(scanned every minute)</span>
            </div>
          )}
        </Section>

        {/* Conditions */}
        <Section
          title="And (conditions, all must match)"
          onAdd={() => setConditions([...conditions, { field: 'priority', op: 'eq', value: 3 }])}
          addLabel="Add condition"
        >
          {conditions.length === 0 ? (
            <p className="text-[11px] text-ink-4 italic">No conditions — trigger fires every time.</p>
          ) : (
            conditions.map((c, i) => (
              <ConditionRow
                key={i}
                condition={c}
                statuses={statuses}
                members={members}
                onChange={(next) => setConditions(conditions.map((x, idx) => idx === i ? next : x))}
                onRemove={() => setConditions(conditions.filter((_, idx) => idx !== i))}
              />
            ))
          )}
        </Section>

        {/* Actions */}
        <Section
          title="Then (actions)"
          onAdd={() => setActions([...actions, { type: 'notify' }])}
          addLabel="Add action"
        >
          {actions.map((a, i) => (
            <ActionRow
              key={i}
              action={a}
              statuses={statuses}
              members={members}
              tags={tags}
              onChange={(next) => setActions(actions.map((x, idx) => idx === i ? next : x))}
              onRemove={() => actions.length > 1 && setActions(actions.filter((_, idx) => idx !== i))}
            />
          ))}
        </Section>

        {errorMsg && (
          <div className="bg-red-50 border border-red-200 rounded-lg px-3 py-2">
            <p className="text-red-600 text-xs font-medium">{errorMsg}</p>
          </div>
        )}
      </div>

      <div className="flex justify-end gap-2 pt-4 border-t border-ink-5/20 mt-3">
        <Button variant="secondary" onClick={onClose}>Cancel</Button>
        <Button onClick={() => save.mutate()} disabled={!name.trim() || save.isPending} loading={save.isPending}>
          {isEdit ? 'Save' : 'Create'}
        </Button>
      </div>
    </Modal>
  )
}

function Section({
  title,
  children,
  onAdd,
  addLabel,
}: {
  title: string
  children: React.ReactNode
  onAdd?: () => void
  addLabel?: string
}) {
  return (
    <div className="bg-canvas/50 border border-ink-5/20 rounded-xl p-3">
      <div className="flex items-center justify-between mb-2">
        <p className="text-[10px] font-bold uppercase tracking-wider text-ink-3">{title}</p>
        {onAdd && (
          <button onClick={onAdd} className="text-[11px] font-medium text-brand-600 hover:text-brand-700 inline-flex items-center gap-1">
            <Plus className="w-3 h-3" /> {addLabel}
          </button>
        )}
      </div>
      <div className="space-y-2">{children}</div>
    </div>
  )
}

function ConditionRow({
  condition, statuses, members, onChange, onRemove,
}: {
  condition: AutomationCondition
  statuses: Status[]
  members: Member[]
  onChange: (c: AutomationCondition) => void
  onRemove: () => void
}) {
  return (
    <div className="flex items-center gap-1.5 bg-surface border border-ink-5/30 rounded-lg p-1.5">
      <select
        className="h-7 px-2 rounded border border-ink-5/30 text-xs bg-transparent"
        value={condition.field}
        onChange={(e) => onChange({ ...condition, field: e.target.value, value: null })}
      >
        <option value="priority">Priority</option>
        <option value="status_id">Status</option>
        <option value="assignee_id">Assignee</option>
      </select>
      <select
        className="h-7 px-2 rounded border border-ink-5/30 text-xs bg-transparent"
        value={condition.op}
        onChange={(e) => onChange({ ...condition, op: e.target.value })}
      >
        <option value="eq">is</option>
        <option value="neq">is not</option>
        {condition.field === 'priority' && (<>
          <option value="gte">≥</option>
          <option value="lte">≤</option>
        </>)}
      </select>
      <div className="flex-1">
        {condition.field === 'priority' && (
          <select
            className="w-full h-7 px-2 rounded border border-ink-5/30 text-xs bg-transparent"
            value={condition.value == null ? '' : String(condition.value)}
            onChange={(e) => onChange({ ...condition, value: Number(e.target.value) })}
          >
            {Object.entries(PRIORITY_LABELS).map(([v, l]) => (
              <option key={v} value={v}>{l}</option>
            ))}
          </select>
        )}
        {condition.field === 'status_id' && (
          <select
            className="w-full h-7 px-2 rounded border border-ink-5/30 text-xs bg-transparent"
            value={(condition.value as string) ?? ''}
            onChange={(e) => onChange({ ...condition, value: e.target.value || null })}
          >
            <option value="">Pick status…</option>
            {statuses.map((s) => <option key={s.id} value={s.id}>{s.name}</option>)}
          </select>
        )}
        {condition.field === 'assignee_id' && (
          <select
            className="w-full h-7 px-2 rounded border border-ink-5/30 text-xs bg-transparent"
            value={(condition.value as string) ?? ''}
            onChange={(e) => onChange({ ...condition, value: e.target.value || null })}
          >
            <option value="">Pick member…</option>
            {members.map((m) => <option key={m.user_id} value={m.user_id}>{m.name || m.email}</option>)}
          </select>
        )}
      </div>
      <button onClick={onRemove} className="text-ink-4 hover:text-red-500">
        <Trash2 className="w-3.5 h-3.5" />
      </button>
    </div>
  )
}

function ActionRow({
  action, statuses, members, tags, onChange, onRemove,
}: {
  action: AutomationAction
  statuses: Status[]
  members: Member[]
  tags: Tag[]
  onChange: (a: AutomationAction) => void
  onRemove: () => void
}) {
  return (
    <div className="flex items-start gap-1.5 bg-surface border border-ink-5/30 rounded-lg p-1.5">
      <select
        className="h-7 px-2 rounded border border-ink-5/30 text-xs bg-transparent"
        value={action.type}
        onChange={(e) => onChange({ type: e.target.value as AutomationActionType })}
      >
        {Object.entries(ACTION_LABELS).map(([v, l]) => (
          <option key={v} value={v}>{l}</option>
        ))}
      </select>

      <div className="flex-1">
        {action.type === 'change_status' && (
          <select
            className="w-full h-7 px-2 rounded border border-ink-5/30 text-xs bg-transparent"
            value={action.status_id ?? ''}
            onChange={(e) => onChange({ ...action, status_id: e.target.value || undefined })}
          >
            <option value="">Pick status…</option>
            {statuses.map((s) => <option key={s.id} value={s.id}>{s.name}</option>)}
          </select>
        )}
        {action.type === 'assign_user' && (
          <select
            className="w-full h-7 px-2 rounded border border-ink-5/30 text-xs bg-transparent"
            value={action.user_id ?? ''}
            onChange={(e) => onChange({ ...action, user_id: e.target.value || undefined })}
          >
            <option value="">Pick member…</option>
            {members.map((m) => <option key={m.user_id} value={m.user_id}>{m.name || m.email}</option>)}
          </select>
        )}
        {action.type === 'add_tag' && (
          <select
            className="w-full h-7 px-2 rounded border border-ink-5/30 text-xs bg-transparent"
            value={action.tag_id ?? ''}
            onChange={(e) => onChange({ ...action, tag_id: e.target.value || undefined })}
          >
            <option value="">Pick tag…</option>
            {tags.map((t) => <option key={t.id} value={t.id}>{t.name}</option>)}
          </select>
        )}
        {action.type === 'add_comment' && (
          <input
            placeholder="Comment body"
            className="w-full h-7 px-2 rounded border border-ink-5/30 text-xs bg-transparent"
            value={action.body ?? ''}
            onChange={(e) => onChange({ ...action, body: e.target.value })}
          />
        )}
        {action.type === 'notify' && (
          <select
            className="w-full h-7 px-2 rounded border border-ink-5/30 text-xs bg-transparent"
            value={action.user_id ?? ''}
            onChange={(e) => onChange({ ...action, user_id: e.target.value || undefined })}
          >
            <option value="">Notify…</option>
            {members.map((m) => <option key={m.user_id} value={m.user_id}>{m.name || m.email}</option>)}
          </select>
        )}
      </div>

      <button onClick={onRemove} className="text-ink-4 hover:text-red-500 mt-1">
        <Trash2 className="w-3.5 h-3.5" />
      </button>
    </div>
  )
}

// kept for compact imports — avoids `'Power'/'AlertCircle' unused` warnings on minor edits
void Power
