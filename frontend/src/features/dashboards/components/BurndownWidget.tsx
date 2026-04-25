import { CartesianGrid, Line, LineChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts'
import type { BurndownPoint } from '../../../types'

export function BurndownWidget({ data }: { data: BurndownPoint[] }) {
  if (!data || data.length === 0) {
    return <p className="text-xs text-ink-4 h-48 flex items-center justify-center">No burndown data — sprint has no pointed tasks.</p>
  }
  const total = data[0]?.total_points ?? 0
  // Ideal line: linear from total → 0 across the sprint.
  const enriched = data.map((p, i) => ({
    day: new Date(p.day).toLocaleDateString(undefined, { month: 'short', day: 'numeric' }),
    remaining: p.remaining_points,
    ideal: Math.max(0, Math.round(total - (total * i) / Math.max(1, data.length - 1))),
  }))
  return (
    <div className="h-52">
      <ResponsiveContainer>
        <LineChart data={enriched} margin={{ top: 5, right: 10, left: -10, bottom: 0 }}>
          <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
          <XAxis dataKey="day" stroke="#9ca3af" fontSize={10} />
          <YAxis stroke="#9ca3af" fontSize={10} />
          <Tooltip
            contentStyle={{ fontSize: 11, borderRadius: 8, border: '1px solid #e5e7eb' }}
            labelStyle={{ color: '#374151', fontWeight: 600 }}
          />
          <Line type="monotone" dataKey="ideal" stroke="#9ca3af" strokeDasharray="4 4" dot={false} name="Ideal" />
          <Line type="monotone" dataKey="remaining" stroke="#5B5CF8" strokeWidth={2} dot={{ r: 2 }} name="Remaining" />
        </LineChart>
      </ResponsiveContainer>
    </div>
  )
}
