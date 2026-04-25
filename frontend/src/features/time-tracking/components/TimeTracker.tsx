import { useEffect, useState } from 'react'
import { Play, Square } from 'lucide-react'
import { cn } from '../../../lib/utils'
import { useActiveTimers, useStartTimer, useStopTimer } from '../hooks/useTimeEntries'

/**
 * Small button on the task detail page. Exactly one running timer is allowed
 * per user (enforced server-side); if a different task's timer is running we
 * stop it and start a new one when the user clicks play.
 */
export function TimeTracker({ taskId }: { taskId: string }) {
  const { data: active = [] } = useActiveTimers()
  const start = useStartTimer()
  const stop = useStopTimer()

  const mine = active.find((e) => e.task_id === taskId)
  const other = active.find((e) => e.task_id !== taskId)
  const running = mine ?? null

  const [elapsed, setElapsed] = useState(0)
  useEffect(() => {
    if (!running) {
      setElapsed(0)
      return
    }
    const startedAtMs = new Date(running.started_at).getTime()
    const tick = () => setElapsed(Math.max(0, Math.floor((Date.now() - startedAtMs) / 1000)))
    tick()
    const iv = setInterval(tick, 1000)
    return () => clearInterval(iv)
  }, [running])

  const handleClick = () => {
    if (running) {
      stop.mutate(running.id)
    } else {
      start.mutate({ taskId })
    }
  }

  return (
    <button
      onClick={handleClick}
      disabled={start.isPending || stop.isPending}
      className={cn(
        'inline-flex items-center gap-1.5 h-7 px-2.5 rounded-lg text-xs font-medium transition-all',
        running
          ? 'bg-red-50 text-red-600 border border-red-200 hover:bg-red-100'
          : 'bg-canvas/50 text-ink-2 border border-ink-5/30 hover:border-brand-300 hover:text-brand-600',
      )}
      title={other ? `Stops current timer on a different task first` : undefined}
    >
      {running ? <Square className="w-3 h-3 fill-current" /> : <Play className="w-3 h-3 fill-current" />}
      {running ? formatHMS(elapsed) : 'Start timer'}
      {other && !running && <span className="text-[10px] text-ink-4">(switch)</span>}
    </button>
  )
}

export function formatHMS(totalSeconds: number): string {
  const s = Math.max(0, Math.floor(totalSeconds))
  const h = Math.floor(s / 3600)
  const m = Math.floor((s % 3600) / 60)
  const sec = s % 60
  if (h > 0) return `${h}:${String(m).padStart(2, '0')}:${String(sec).padStart(2, '0')}`
  return `${m}:${String(sec).padStart(2, '0')}`
}
