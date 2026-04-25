import { Calendar, dateFnsLocalizer, type Event } from 'react-big-calendar'
import { useNavigate } from 'react-router-dom'
import { format, parse, startOfWeek, getDay } from 'date-fns'
import { enUS } from 'date-fns/locale/en-US'
import 'react-big-calendar/lib/css/react-big-calendar.css'

import type { Status, Task, ViewConfig } from '../../../types'
import { applyView } from '../lib/applyView'

const locales = { 'en-US': enUS }
const localizer = dateFnsLocalizer({
  format,
  parse,
  startOfWeek,
  getDay,
  locales,
})

interface Props {
  tasks: Task[]
  statuses: Status[]
  workspaceId: string
  config: ViewConfig
  taskAssignees: Record<string, string[]>
  taskTagIds: Record<string, string[]>
}

interface TaskEvent extends Event {
  taskId: string
  priority: number
  statusColor: string
}

export function CalendarView({ tasks, statuses, workspaceId, config, taskAssignees, taskTagIds }: Props) {
  const navigate = useNavigate()
  const statusById = new Map(statuses.map((s) => [s.id, s]))
  const filtered = applyView(tasks, config, taskTagIds, taskAssignees)

  const events: TaskEvent[] = filtered
    .filter((t) => !!t.due_at)
    .map((t) => {
      const end = new Date(t.due_at!)
      const start = t.start_at ? new Date(t.start_at) : end
      return {
        taskId: t.id,
        title: t.name,
        start,
        end,
        allDay: true,
        priority: t.priority,
        statusColor: t.status_id ? (statusById.get(t.status_id)?.color ?? '#94a3b8') : '#94a3b8',
      }
    })

  return (
    <div className="px-7 py-4">
      <style>{`
        .rbc-calendar { font-family: inherit; }
        .rbc-toolbar button { border-radius: 8px; }
        .rbc-toolbar button.rbc-active { background: #5B5CF8; color: #fff; border-color: #5B5CF8; }
        .rbc-event { border: none !important; border-radius: 6px !important; padding: 2px 6px !important; font-size: 11px !important; }
        .rbc-today { background: rgba(91,92,248,0.06); }
      `}</style>
      <div className="bg-surface border border-ink-5/30 rounded-2xl shadow-card p-4" style={{ height: 'calc(100vh - 230px)' }}>
        <Calendar<TaskEvent>
          localizer={localizer}
          events={events}
          startAccessor="start"
          endAccessor="end"
          views={['month', 'week', 'day', 'agenda']}
          defaultView="month"
          popup
          onSelectEvent={(e) => navigate(`/workspaces/${workspaceId}/tasks/${e.taskId}`)}
          eventPropGetter={(e) => ({
            style: {
              backgroundColor: e.statusColor,
              opacity: 0.85,
            },
          })}
          style={{ height: '100%' }}
        />
      </div>
    </div>
  )
}
