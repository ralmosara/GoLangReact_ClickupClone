import { useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link, useParams } from 'react-router-dom'
import { CheckCircle2, FileSpreadsheet, FileText } from 'lucide-react'

import { api } from '../../lib/api'
import { Button, PageSpinner } from '../../components/ui'
import { cn, formatRelative } from '../../lib/utils'
import type { AccomplishmentBucket } from '../../types'

type GroupBy = 'day' | 'week'

export function AccomplishmentsPage() {
  const { workspaceId } = useParams<{ workspaceId: string }>()

  const defaults = useMemo(() => {
    const to = new Date()
    const from = new Date(to.getTime() - 30 * 24 * 3600 * 1000)
    return { from: from.toISOString().slice(0, 10), to: to.toISOString().slice(0, 10) }
  }, [])
  const [from, setFrom] = useState(defaults.from)
  const [to, setTo] = useState(defaults.to)
  const [groupBy, setGroupBy] = useState<GroupBy>('day')

  const { data: buckets = [], isLoading } = useQuery({
    queryKey: ['accomplishments', workspaceId, from, to, groupBy],
    queryFn: () =>
      api
        .get(
          `workspaces/${workspaceId}/reports/accomplishments?from=${new Date(from).toISOString()}&to=${new Date(
            to,
          ).toISOString()}&group_by=${groupBy}`,
        )
        .json<AccomplishmentBucket[] | null>(),
    enabled: !!workspaceId,
    select: (rows) => rows ?? [],
  })

  const total = buckets.reduce((s, b) => s + b.count, 0)

  const [exporting, setExporting] = useState<'pdf' | 'xlsx' | null>(null)
  const handleExport = async (fmt: 'pdf' | 'xlsx') => {
    if (!workspaceId) return
    setExporting(fmt)
    try {
      const path = `workspaces/${workspaceId}/reports/accomplishments.${fmt}` +
        `?from=${new Date(from).toISOString()}` +
        `&to=${new Date(to).toISOString()}` +
        `&group_by=${groupBy}`
      const blob = await api.get(path).blob()
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `accomplishments_${groupBy}_${from}_to_${to}.${fmt}`
      document.body.appendChild(a)
      a.click()
      a.remove()
      URL.revokeObjectURL(url)
    } finally {
      setExporting(null)
    }
  }

  return (
    <div className="max-w-3xl mx-auto px-6 py-10 animate-slide-up">
      <div className="flex items-start justify-between gap-4 mb-2">
        <div className="flex items-start gap-3">
          <div className="w-11 h-11 rounded-xl bg-brand-50 border border-brand-100 flex items-center justify-center shrink-0">
            <CheckCircle2 className="w-5 h-5 text-brand-500" />
          </div>
          <div>
            <h1 className="text-xl font-bold text-ink-1">Accomplishments</h1>
            <p className="text-sm text-ink-4 mt-0.5">
              Tasks you completed in this workspace, grouped by day or week. Archived tasks still show here.
            </p>
          </div>
        </div>
        <div className="flex items-center gap-2 shrink-0">
          <Button
            variant="secondary"
            size="sm"
            onClick={() => handleExport('pdf')}
            disabled={exporting !== null || total === 0}
            loading={exporting === 'pdf'}
            title="Download executive-style PDF"
          >
            <FileText className="w-3.5 h-3.5 mr-1.5" />
            Export PDF
          </Button>
          <Button
            variant="secondary"
            size="sm"
            onClick={() => handleExport('xlsx')}
            disabled={exporting !== null || total === 0}
            loading={exporting === 'xlsx'}
            title="Download Excel workbook"
          >
            <FileSpreadsheet className="w-3.5 h-3.5 mr-1.5" />
            Export Excel
          </Button>
        </div>
      </div>

      <div className="bg-surface border border-ink-5/30 rounded-2xl p-4 shadow-card mb-4 flex flex-wrap items-end gap-3">
        <div>
          <label className="block text-xs font-medium text-ink-2 mb-1">From</label>
          <input
            type="date"
            value={from}
            onChange={(e) => setFrom(e.target.value)}
            className="h-9 bg-surface border border-ink-5/40 rounded-lg px-3 text-sm focus:outline-none focus:ring-2 focus:ring-brand-400/40"
          />
        </div>
        <div>
          <label className="block text-xs font-medium text-ink-2 mb-1">To</label>
          <input
            type="date"
            value={to}
            onChange={(e) => setTo(e.target.value)}
            className="h-9 bg-surface border border-ink-5/40 rounded-lg px-3 text-sm focus:outline-none focus:ring-2 focus:ring-brand-400/40"
          />
        </div>
        <div>
          <label className="block text-xs font-medium text-ink-2 mb-1">Group by</label>
          <div className="inline-flex rounded-lg border border-ink-5/40 overflow-hidden">
            {(['day', 'week'] as GroupBy[]).map((opt) => (
              <button
                key={opt}
                onClick={() => setGroupBy(opt)}
                className={cn(
                  'h-9 px-3 text-xs font-semibold transition-colors',
                  groupBy === opt
                    ? 'bg-brand-500 text-white'
                    : 'bg-surface text-ink-2 hover:bg-canvas',
                )}
              >
                {opt === 'day' ? 'Day' : 'Week'}
              </button>
            ))}
          </div>
        </div>
        <div className="ml-auto text-right">
          <p className="text-[10px] font-semibold uppercase tracking-wider text-ink-4">Total</p>
          <p className="text-2xl font-bold text-ink-1 tabular-nums">{total}</p>
          <p className="text-[11px] text-ink-4">{total === 1 ? 'task completed' : 'tasks completed'}</p>
        </div>
      </div>

      {isLoading && <PageSpinner />}

      {!isLoading && buckets.length === 0 && (
        <div className="text-center py-14 bg-surface/50 border border-ink-5/20 rounded-2xl text-sm text-ink-4">
          Nothing completed in this range.
        </div>
      )}

      {!isLoading && buckets.length > 0 && (
        <div className="space-y-3 animate-fade-in">
          {buckets.map((b) => (
            <BucketCard key={b.period} bucket={b} groupBy={groupBy} workspaceId={workspaceId} />
          ))}
        </div>
      )}
    </div>
  )
}

function BucketCard({
  bucket,
  groupBy,
  workspaceId,
}: {
  bucket: AccomplishmentBucket
  groupBy: GroupBy
  workspaceId: string | undefined
}) {
  return (
    <div className="bg-surface border border-ink-5/30 rounded-2xl shadow-card overflow-hidden">
      <div className="flex items-center justify-between px-4 py-3 border-b border-ink-5/20 bg-canvas/40">
        <div>
          <p className="text-sm font-semibold text-ink-1">
            {groupBy === 'day' ? formatBucketDay(bucket.starts_at) : formatBucketWeek(bucket)}
          </p>
          <p className="text-[11px] text-ink-4">{bucket.period}</p>
        </div>
        <span className="text-xs font-semibold text-ink-2 tabular-nums">
          {bucket.count} {bucket.count === 1 ? 'task' : 'tasks'}
        </span>
      </div>
      <ul className="divide-y divide-ink-5/10">
        {bucket.tasks.map((t) => (
          <li key={t.id} className="px-4 py-3 flex items-start gap-3">
            <CheckCircle2 className="w-4 h-4 text-brand-500 mt-0.5 shrink-0" />
            <div className="min-w-0 flex-1">
              <div className="flex items-center gap-2 flex-wrap">
                <Link
                  to={workspaceId ? `/workspaces/${workspaceId}/tasks/${t.id}` : '#'}
                  className="text-sm font-medium text-ink-1 hover:text-brand-600 truncate"
                >
                  {t.name}
                </Link>
                {t.archived && (
                  <span className="text-[10px] font-semibold uppercase tracking-wider text-ink-4 bg-ink-5/20 rounded px-1.5 py-0.5">
                    Archived
                  </span>
                )}
              </div>
              <p className="text-[11px] text-ink-4 mt-0.5 truncate">
                in {t.list_name} · completed {formatAbsolute(t.completed_at)} ({formatRelative(t.completed_at)})
              </p>
            </div>
          </li>
        ))}
      </ul>
    </div>
  )
}

function formatBucketDay(iso: string): string {
  const d = new Date(iso)
  return d.toLocaleDateString(undefined, { weekday: 'long', month: 'short', day: 'numeric' })
}

function formatAbsolute(iso: string): string {
  const d = new Date(iso)
  return d.toLocaleString(undefined, {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
    hour: 'numeric',
    minute: '2-digit',
  })
}

function formatBucketWeek(b: AccomplishmentBucket): string {
  const start = new Date(b.starts_at)
  const end = new Date(new Date(b.ends_at).getTime() - 24 * 3600 * 1000)
  const fmt = (d: Date) => d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' })
  return `${fmt(start)} – ${fmt(end)}`
}
