import { type ClassValue, clsx } from 'clsx'
import { twMerge } from 'tailwind-merge'

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export function formatDate(iso: string) {
  return new Date(iso).toLocaleDateString(undefined, {
    month: 'short', day: 'numeric', year: 'numeric',
  })
}

export function formatRelative(iso: string) {
  const d = new Date(iso)
  const diff = (Date.now() - d.getTime()) / 1000
  if (diff < 60)   return 'just now'
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`
  if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`
  return formatDate(iso)
}

export const PRIORITY_LABELS: Record<number, string> = {
  0: 'No priority',
  1: 'Low',
  2: 'Normal',
  3: 'High',
  4: 'Urgent',
}

export const PRIORITY_COLORS: Record<number, string> = {
  0: 'bg-ink-5/50 text-ink-3',
  1: 'bg-blue-50 text-blue-600',
  2: 'bg-slate-100 text-slate-600',
  3: 'bg-orange-50 text-orange-600',
  4: 'bg-red-50 text-red-600',
}

export const STATUS_COLORS: Record<string, string> = {
  open:        'bg-[#F0F4FF] text-[#4F63C8]',
  in_progress: 'bg-[#FFF7ED] text-[#C05621]',
  review:      'bg-[#FEFCE8] text-[#A16207]',
  completed:   'bg-[#F0FDF4] text-[#166534]',
  cancelled:   'bg-[#FEF2F2] text-[#991B1B]',
}

export const STATUS_DOT: Record<string, string> = {
  open:        '#6375E8',
  in_progress: '#F97316',
  review:      '#EAB308',
  completed:   '#22C55E',
  cancelled:   '#EF4444',
}

export const STATUS_LABEL: Record<string, string> = {
  open:        'Open',
  in_progress: 'In Progress',
  review:      'In Review',
  completed:   'Completed',
  cancelled:   'Cancelled',
}
