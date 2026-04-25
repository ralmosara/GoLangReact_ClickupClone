import { useMemo } from 'react'
import {
  flexRender,
  getCoreRowModel,
  getSortedRowModel,
  useReactTable,
  type ColumnDef,
} from '@tanstack/react-table'
import { useNavigate } from 'react-router-dom'
import { cn, PRIORITY_COLORS, PRIORITY_LABELS } from '../../../lib/utils'
import type { Status, Task, ViewConfig } from '../../../types'
import { applyView } from '../lib/applyView'

interface Props {
  tasks: Task[]
  statuses: Status[]
  workspaceId: string
  config: ViewConfig
  taskAssignees: Record<string, string[]>
  taskTagIds: Record<string, string[]>
}

export function TableView({ tasks, statuses, workspaceId, config, taskAssignees, taskTagIds }: Props) {
  const navigate = useNavigate()
  const statusById = useMemo(() => new Map(statuses.map((s) => [s.id, s])), [statuses])
  const visible = applyView(tasks, config, taskTagIds, taskAssignees)

  const columns = useMemo<ColumnDef<Task>[]>(
    () => [
      {
        accessorKey: 'name',
        header: 'Name',
        cell: (info) => <span className="font-medium text-ink-1">{info.getValue() as string}</span>,
      },
      {
        accessorKey: 'status_id',
        header: 'Status',
        cell: (info) => {
          const id = info.getValue() as string | undefined
          const st = id ? statusById.get(id) : undefined
          if (!st) return <span className="text-[11px] text-ink-4">—</span>
          return (
            <span
              className="inline-flex items-center gap-1 text-[11px] font-medium px-2 py-0.5 rounded-full"
              style={{ backgroundColor: st.color + '22', color: st.color }}
            >
              <span className="w-1.5 h-1.5 rounded-full" style={{ backgroundColor: st.color }} />
              {st.name}
            </span>
          )
        },
      },
      {
        accessorKey: 'priority',
        header: 'Priority',
        cell: (info) => {
          const p = info.getValue() as number
          return (
            <span className={cn('text-[11px] font-medium px-2 py-0.5 rounded-full inline-flex', PRIORITY_COLORS[p] ?? PRIORITY_COLORS[0])}>
              {PRIORITY_LABELS[p] ?? 'No priority'}
            </span>
          )
        },
      },
      {
        id: 'assignees',
        header: 'Assignees',
        cell: (info) => {
          const ids = taskAssignees[info.row.original.id] ?? []
          if (ids.length === 0) return <span className="text-[11px] text-ink-4">—</span>
          return (
            <div className="flex -space-x-1.5">
              {ids.slice(0, 3).map((uid) => (
                <div key={uid} className="w-6 h-6 rounded-full bg-brand-gradient text-[10px] font-bold text-white flex items-center justify-center border-2 border-surface">
                  {uid.slice(0, 1).toUpperCase()}
                </div>
              ))}
              {ids.length > 3 && <div className="w-6 h-6 rounded-full bg-ink-5/60 text-[10px] font-bold text-ink-2 flex items-center justify-center border-2 border-surface">+{ids.length - 3}</div>}
            </div>
          )
        },
      },
      {
        accessorKey: 'due_at',
        header: 'Due',
        cell: (info) => {
          const v = info.getValue() as string | undefined
          if (!v) return <span className="text-[11px] text-ink-4">—</span>
          return <span className="text-xs text-ink-2">{new Date(v).toLocaleDateString(undefined, { month: 'short', day: 'numeric' })}</span>
        },
      },
      {
        accessorKey: 'updated_at',
        header: 'Updated',
        cell: (info) => (
          <span className="text-[11px] text-ink-4">
            {new Date(info.getValue() as string).toLocaleDateString()}
          </span>
        ),
      },
    ],
    [statusById, taskAssignees],
  )

  const table = useReactTable({
    data: visible,
    columns,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
  })

  return (
    <div className="px-7 py-4">
      <div className="bg-surface border border-ink-5/30 rounded-2xl shadow-card overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            {table.getHeaderGroups().map((hg) => (
              <tr key={hg.id} className="border-b border-ink-5/20 bg-canvas/40 text-[10px] font-bold uppercase tracking-wider text-ink-4">
                {hg.headers.map((h) => (
                  <th
                    key={h.id}
                    className="text-left px-4 py-2 cursor-pointer select-none hover:text-ink-2"
                    onClick={h.column.getToggleSortingHandler()}
                  >
                    <span className="inline-flex items-center gap-1">
                      {flexRender(h.column.columnDef.header, h.getContext())}
                      {h.column.getIsSorted() === 'asc' && '↑'}
                      {h.column.getIsSorted() === 'desc' && '↓'}
                    </span>
                  </th>
                ))}
              </tr>
            ))}
          </thead>
          <tbody>
            {table.getRowModel().rows.map((r) => (
              <tr
                key={r.id}
                className="border-b border-ink-5/10 last:border-b-0 hover:bg-ink-1/2 cursor-pointer"
                onClick={() => navigate(`/workspaces/${workspaceId}/tasks/${r.original.id}`)}
              >
                {r.getVisibleCells().map((c) => (
                  <td key={c.id} className="px-4 py-2">
                    {flexRender(c.column.columnDef.cell, c.getContext())}
                  </td>
                ))}
              </tr>
            ))}
            {table.getRowModel().rows.length === 0 && (
              <tr>
                <td colSpan={columns.length} className="px-4 py-6 text-center text-xs text-ink-4">
                  No tasks match current filters.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  )
}
