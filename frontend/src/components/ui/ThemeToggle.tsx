import { Monitor, Moon, Sun } from 'lucide-react'
import { setTheme, useTheme, type ThemeMode } from '../../lib/theme'
import { t } from '../../lib/i18n'
import { cn } from '../../lib/utils'

const OPTIONS: { mode: ThemeMode; key: string; Icon: typeof Sun }[] = [
  { mode: 'light',  key: 'theme.light',  Icon: Sun },
  { mode: 'dark',   key: 'theme.dark',   Icon: Moon },
  { mode: 'system', key: 'theme.system', Icon: Monitor },
]

export function ThemeToggle() {
  const mode = useTheme()
  return (
    <div
      role="radiogroup"
      aria-label="Theme"
      className="inline-flex rounded-lg border border-ink-5/40 bg-surface p-0.5 dark:border-white/10 dark:bg-white/5"
    >
      {OPTIONS.map(({ mode: m, key, Icon }) => {
        const active = mode === m
        return (
          <button
            key={m}
            role="radio"
            aria-checked={active}
            onClick={() => setTheme(m)}
            className={cn(
              'inline-flex items-center gap-1.5 px-3 h-8 rounded-md text-xs font-medium transition-colors',
              active
                ? 'bg-brand-50 text-brand-700 dark:bg-brand-500/20 dark:text-brand-200'
                : 'text-ink-3 hover:text-ink-1 dark:text-white/60 dark:hover:text-white',
            )}
          >
            <Icon className="w-3.5 h-3.5" />
            {t(key)}
          </button>
        )
      })}
    </div>
  )
}
