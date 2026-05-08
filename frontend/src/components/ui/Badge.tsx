import { cn } from '../../lib/utils'

interface BadgeProps {
  children: React.ReactNode
  className?: string
  dot?: boolean
  dotColor?: string
}

export function Badge({ children, className, dot, dotColor }: BadgeProps) {
  return (
    <span className={cn(
      'inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-[11px] font-semibold tracking-wide',
      className,
    )}>
      {dot && (
        <span
          className="w-1.5 h-1.5 rounded-full shrink-0"
          style={dotColor ? { backgroundColor: dotColor } : undefined}
        />
      )}
      {children}
    </span>
  )
}

const STATUS_MAP: Record<string, { bg: string; text: string; dot: string; label: string }> = {
  open:        { bg: 'bg-[#F0F4FF]', text: 'text-[#4F63C8]', dot: '#6375E8', label: 'To Do' },
  in_progress: { bg: 'bg-[#FFF7ED]', text: 'text-[#C05621]', dot: '#F97316', label: 'In Progress' },
  review:      { bg: 'bg-[#FEFCE8]', text: 'text-[#A16207]', dot: '#EAB308', label: 'In Review' },
  completed:   { bg: 'bg-[#F0FDF4]', text: 'text-[#166534]', dot: '#22C55E', label: 'Completed' },
  cancelled:   { bg: 'bg-[#FEF2F2]', text: 'text-[#991B1B]', dot: '#EF4444', label: 'Cancelled' },
}

export function StatusBadge({ status }: { status: string }) {
  const s = STATUS_MAP[status] ?? STATUS_MAP.open
  return (
    <Badge className={cn(s.bg, s.text)} dot dotColor={s.dot}>
      {s.label}
    </Badge>
  )
}

export function PriorityBadge({ priority }: { priority: number }) {
  const map: Record<number, { label: string; className: string }> = {
    0: { label: 'No priority',  className: 'bg-ink-5/50 text-ink-3' },
    1: { label: 'Low',          className: 'bg-blue-50 text-blue-600' },
    2: { label: 'Normal',       className: 'bg-slate-100 text-slate-600' },
    3: { label: 'High',         className: 'bg-orange-50 text-orange-600' },
    4: { label: 'Urgent',       className: 'bg-red-50 text-red-600' },
  }
  const p = map[priority] ?? map[0]
  return <Badge className={p.className}>{p.label}</Badge>
}
