import { Bar, BarChart, CartesianGrid, Cell, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts'
import type { VelocityPoint } from '../../../types'

export function VelocityWidget({ data }: { data: VelocityPoint[] }) {
  if (!data || data.length === 0) {
    return <p className="text-xs text-ink-4 h-48 flex items-center justify-center">No sprints with completed points yet.</p>
  }
  // recharts wants newest-last.
  const sorted = data.slice().reverse().map((v) => ({
    ...v,
    label: v.sprint_name,
    short: v.sprint_name.length > 12 ? v.sprint_name.slice(0, 10) + '…' : v.sprint_name,
  }))
  return (
    <div className="h-52">
      <ResponsiveContainer>
        <BarChart data={sorted} margin={{ top: 5, right: 10, left: -10, bottom: 0 }}>
          <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
          <XAxis dataKey="short" stroke="#9ca3af" fontSize={10} />
          <YAxis stroke="#9ca3af" fontSize={10} />
          <Tooltip
            contentStyle={{ fontSize: 11, borderRadius: 8, border: '1px solid #e5e7eb' }}
            labelFormatter={(_label, items) => (items?.[0]?.payload as { label?: string })?.label ?? ''}
          />
          <Bar dataKey="completed_points" name="Completed" radius={[4, 4, 0, 0]}>
            {sorted.map((v, i) => (
              <Cell
                key={i}
                fill={v.completed_points >= v.goal_points && v.goal_points > 0 ? '#22C55E' : '#5B5CF8'}
              />
            ))}
          </Bar>
        </BarChart>
      </ResponsiveContainer>
    </div>
  )
}
