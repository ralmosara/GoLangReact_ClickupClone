import { useState } from 'react'
import * as Popover from '@radix-ui/react-popover'
import { Repeat, X } from 'lucide-react'
import { cn } from '../../../lib/utils'

const PRESETS: Array<{ label: string; rule: string }> = [
  { label: 'Daily',                  rule: 'FREQ=DAILY' },
  { label: 'Every weekday (Mon–Fri)', rule: 'FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR' },
  { label: 'Weekly',                 rule: 'FREQ=WEEKLY' },
  { label: 'Every 2 weeks',          rule: 'FREQ=WEEKLY;INTERVAL=2' },
  { label: 'Monthly',                rule: 'FREQ=MONTHLY' },
  { label: 'Yearly',                 rule: 'FREQ=YEARLY' },
]

export function RecurrencePicker({
  value,
  onChange,
}: {
  value: string | null | undefined
  onChange: (rule: string | null) => void
}) {
  const [open, setOpen] = useState(false)
  const label = value ? describeRule(value) : null

  return (
    <Popover.Root open={open} onOpenChange={setOpen}>
      <Popover.Trigger asChild>
        <button
          className={cn(
            'inline-flex items-center gap-1.5 h-7 px-2.5 rounded-full text-[11px] font-medium border transition-colors',
            value
              ? 'bg-brand-50 text-brand-700 border-brand-200 hover:bg-brand-100'
              : 'bg-canvas/50 text-ink-3 border-ink-5/30 hover:border-brand-300 hover:text-brand-600',
          )}
        >
          <Repeat className="w-3 h-3" />
          {label ?? 'Does not repeat'}
          {value && (
            <span
              role="button"
              onClick={(e) => {
                e.stopPropagation()
                onChange(null)
              }}
              className="hover:text-red-600 ml-0.5"
            >
              <X className="w-3 h-3" />
            </span>
          )}
        </button>
      </Popover.Trigger>
      <Popover.Portal>
        <Popover.Content sideOffset={6} className="bg-surface border border-ink-5/30 rounded-xl shadow-modal p-2 w-64 z-50">
          <p className="text-[10px] font-bold uppercase tracking-wider text-ink-4 px-1 py-1">Repeat</p>
          <div className="space-y-0.5">
            {PRESETS.map((p) => (
              <button
                key={p.rule}
                onClick={() => { onChange(p.rule); setOpen(false) }}
                className={cn(
                  'block w-full text-left px-2 py-1.5 rounded text-xs hover:bg-ink-1/5',
                  value === p.rule ? 'text-brand-600 font-semibold' : 'text-ink-1',
                )}
              >
                {p.label}
              </button>
            ))}
          </div>
          <div className="pt-2 mt-2 border-t border-ink-5/20">
            <p className="text-[10px] text-ink-4 px-1">Custom RRULE (iCal format)</p>
            <input
              defaultValue={value ?? ''}
              placeholder="FREQ=WEEKLY;BYDAY=MO"
              className="w-full h-8 mt-1 bg-canvas/50 border border-ink-5/30 rounded-lg px-2 text-xs text-ink-1 focus:outline-none focus:ring-2 focus:ring-brand-400/40"
              onKeyDown={(e) => {
                if (e.key === 'Enter') {
                  const v = (e.target as HTMLInputElement).value.trim()
                  onChange(v === '' ? null : v)
                  setOpen(false)
                }
              }}
            />
          </div>
        </Popover.Content>
      </Popover.Portal>
    </Popover.Root>
  )
}

function describeRule(rule: string): string {
  const preset = PRESETS.find((p) => p.rule === rule)
  if (preset) return `Repeats: ${preset.label.toLowerCase()}`
  // Fallback: show a tight hint parsed from FREQ.
  const m = rule.match(/FREQ=(\w+)/i)
  return `Repeats: ${m ? m[1].toLowerCase() : 'custom'}`
}
