import { useState } from 'react'
import { Plus } from 'lucide-react'
import { Button, Input, Modal } from '../../../components/ui'
import { cn, formatRelative } from '../../../lib/utils'
import type { TimeEntry } from '../../../types'
import { useDeleteTimeEntry, useLogManual, useTimeEntries } from '../hooks/useTimeEntries'
import { formatHMS } from './TimeTracker'

export function TimeEntriesList({ taskId }: { taskId: string }) {
  const { data: entries = [] } = useTimeEntries(taskId)
  const del = useDeleteTimeEntry(taskId)
  const [showLog, setShowLog] = useState(false)

  const totalS = entries.reduce((sum, e) => sum + (e.duration_s ?? 0), 0)
  const billableS = entries.filter((e) => e.billable).reduce((sum, e) => sum + (e.duration_s ?? 0), 0)

  return (
    <div>
      <div className="flex items-center gap-3 mb-3">
        <div className="flex-1">
          <p className="text-sm font-semibold text-ink-1">Total logged: {formatHMS(totalS)}</p>
          {billableS > 0 && <p className="text-[11px] text-ink-4">{formatHMS(billableS)} billable</p>}
        </div>
        <Button variant="secondary" size="sm" onClick={() => setShowLog(true)}>
          <Plus className="w-3 h-3 mr-1" /> Log time
        </Button>
      </div>

      {entries.length === 0 ? (
        <p className="text-xs text-ink-4">No time logged yet.</p>
      ) : (
        <ul className="space-y-1.5">
          {entries.map((e) => (
            <li
              key={e.id}
              className={cn(
                'flex items-center gap-3 px-3 py-2 rounded-lg border group',
                e.stopped_at ? 'bg-canvas/60 border-ink-5/20' : 'bg-red-50 border-red-200',
              )}
            >
              <div className="flex-1 min-w-0">
                <p className="text-xs text-ink-1 truncate">
                  {e.note || <span className="text-ink-4">No note</span>}
                  {e.billable && <span className="ml-2 text-[9px] font-bold uppercase text-green-700 bg-green-50 border border-green-200 rounded px-1">$ billable</span>}
                  {!e.stopped_at && <span className="ml-2 text-[9px] font-bold uppercase text-red-700">running</span>}
                </p>
                <p className="text-[10px] text-ink-4">
                  {formatRange(e)} · {formatRelative(e.started_at)}
                </p>
              </div>
              <span className="text-sm font-semibold text-ink-1 tabular-nums">
                {formatHMS(e.duration_s ?? Math.floor((Date.now() - new Date(e.started_at).getTime()) / 1000))}
              </span>
              <button
                onClick={() => {
                  if (confirm('Delete this time entry?')) del.mutate(e.id)
                }}
                className="opacity-0 group-hover:opacity-100 text-ink-4 hover:text-red-500 p-1 rounded transition-all"
                title="Delete"
              >
                <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </li>
          ))}
        </ul>
      )}

      {showLog && <LogTimeModal taskId={taskId} onClose={() => setShowLog(false)} />}
    </div>
  )
}

function formatRange(e: TimeEntry): string {
  const start = new Date(e.started_at).toLocaleString(undefined, { month: 'short', day: 'numeric', hour: 'numeric', minute: '2-digit' })
  if (!e.stopped_at) return `${start} → running`
  const stop = new Date(e.stopped_at).toLocaleString(undefined, { hour: 'numeric', minute: '2-digit' })
  return `${start} → ${stop}`
}

function LogTimeModal({ taskId, onClose }: { taskId: string; onClose: () => void }) {
  const log = useLogManual(taskId)
  const now = new Date()
  const oneHourAgo = new Date(now.getTime() - 60 * 60 * 1000)

  const [started, setStarted] = useState(toLocal(oneHourAgo))
  const [stopped, setStopped] = useState(toLocal(now))
  const [note, setNote] = useState('')
  const [billable, setBillable] = useState(false)

  const handleSave = () => {
    const startedAt = new Date(started).toISOString()
    const stoppedAt = new Date(stopped).toISOString()
    log.mutate({ started_at: startedAt, stopped_at: stoppedAt, note, billable }, { onSuccess: onClose })
  }

  return (
    <Modal open title="Log time" description="Add a manual time entry." onClose={onClose}>
      <div className="space-y-3">
        <div className="grid grid-cols-2 gap-3">
          <div>
            <label className="block text-xs font-medium text-ink-2 mb-1">Start</label>
            <input
              type="datetime-local"
              value={started}
              onChange={(e) => setStarted(e.target.value)}
              className="w-full h-9 bg-surface border border-ink-5/40 rounded-lg px-3 text-sm focus:outline-none focus:ring-2 focus:ring-brand-400/40"
            />
          </div>
          <div>
            <label className="block text-xs font-medium text-ink-2 mb-1">End</label>
            <input
              type="datetime-local"
              value={stopped}
              onChange={(e) => setStopped(e.target.value)}
              className="w-full h-9 bg-surface border border-ink-5/40 rounded-lg px-3 text-sm focus:outline-none focus:ring-2 focus:ring-brand-400/40"
            />
          </div>
        </div>
        <Input label="Note" placeholder="What were you doing?" value={note} onChange={(e) => setNote(e.target.value)} />
        <label className="flex items-center gap-2 text-sm text-ink-2 cursor-pointer">
          <input type="checkbox" checked={billable} onChange={(e) => setBillable(e.target.checked)} />
          Billable
        </label>
        {log.error && <p className="text-red-500 text-xs">{String(log.error)}</p>}
        <div className="flex justify-end gap-2 pt-1">
          <Button variant="secondary" onClick={onClose}>Cancel</Button>
          <Button onClick={handleSave} disabled={log.isPending} loading={log.isPending}>Save</Button>
        </div>
      </div>
    </Modal>
  )
}

function toLocal(d: Date): string {
  // Format a Date for the datetime-local input (YYYY-MM-DDTHH:MM).
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}
