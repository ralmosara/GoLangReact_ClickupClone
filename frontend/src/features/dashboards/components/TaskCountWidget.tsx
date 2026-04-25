import { Bar, BarChart, CartesianGrid, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts'

export function TaskCountWidget({ data }: { data: Record<string, number> }) {
  const rows = Object.entries(data ?? {}).map(([name, count]) => ({ name, count }))
  if (rows.length === 0) {
    return <p className="text-xs text-ink-4 h-48 flex items-center justify-center">No tasks in this list yet.</p>
  }
  return (
    <div className="h-52">
      <ResponsiveContainer>
        <BarChart data={rows} margin={{ top: 5, right: 10, left: -10, bottom: 0 }}>
          <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
          <XAxis dataKey="name" stroke="#9ca3af" fontSize={10} />
          <YAxis stroke="#9ca3af" fontSize={10} allowDecimals={false} />
          <Tooltip contentStyle={{ fontSize: 11, borderRadius: 8, border: '1px solid #e5e7eb' }} />
          <Bar dataKey="count" fill="#5B5CF8" radius={[4, 4, 0, 0]} />
        </BarChart>
      </ResponsiveContainer>
    </div>
  )
}
