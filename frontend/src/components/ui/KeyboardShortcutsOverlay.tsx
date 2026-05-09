import { useState } from 'react'
import { useHotkeys } from 'react-hotkeys-hook'
import { Keyboard, X } from 'lucide-react'
import { t } from '../../lib/i18n'
import { cn } from '../../lib/utils'

// Mac shows ⌘, everyone else shows Ctrl. The keys themselves are the same —
// react-hotkeys-hook normalises meta/ctrl on the listening side.
const isMac = typeof navigator !== 'undefined' && /Mac/.test(navigator.platform)
const MOD = isMac ? '⌘' : 'Ctrl'

type Shortcut = { keys: string[]; label: string }
type Group    = { label: string; items: Shortcut[] }

const GROUPS: Group[] = [
  {
    label: 'General',
    items: [
      { keys: [MOD, 'K'], label: 'Open command palette / search' },
      { keys: ['?'],      label: 'Show this shortcuts overlay' },
      { keys: ['Esc'],    label: 'Close any open dialog or palette' },
    ],
  },
  {
    label: 'Navigation',
    items: [
      { keys: ['G', 'I'], label: 'Go to Inbox / Notifications' },
      { keys: ['G', 'D'], label: 'Go to Docs' },
      { keys: ['G', 'C'], label: 'Go to Chat' },
      { keys: ['G', 'B'], label: 'Go to Dashboards' },
      { keys: ['G', 'S'], label: 'Go to Settings' },
    ],
  },
  {
    label: 'Tasks',
    items: [
      { keys: ['N'],       label: 'Create new task in current list' },
      { keys: ['E'],       label: 'Edit focused task' },
      { keys: ['Space'],   label: 'Toggle complete on focused task' },
      { keys: [MOD, 'Enter'], label: 'Submit / save current form' },
      { keys: ['Del'],     label: 'Delete focused task' },
    ],
  },
  {
    label: 'Comments',
    items: [
      { keys: ['R'],       label: 'Reply to focused comment' },
      { keys: [MOD, 'B'],  label: 'Bold selection' },
      { keys: [MOD, 'I'],  label: 'Italic selection' },
      { keys: ['@'],       label: 'Mention a teammate' },
    ],
  },
]

/**
 * Press `?` anywhere in the app to open this overlay. Lives at the layout
 * level so it's available everywhere, including pages that haven't bound
 * their own hotkeys.
 *
 * Hotkey ergonomics: the listener uses `enableOnFormTags: false` so typing
 * "?" inside an input box doesn't accidentally pop the overlay. That means
 * users in inputs won't trigger it — which is the right tradeoff (the
 * sidebar trigger button below is the keyboard-trapped escape hatch).
 */
export function KeyboardShortcutsOverlay() {
  const [open, setOpen] = useState(false)

  useHotkeys('shift+/', () => setOpen(true), { preventDefault: true })
  useHotkeys('escape', () => setOpen(false), { enabled: open, enableOnFormTags: true })

  if (!open) return null

  return (
    <div
      className="fixed inset-0 z-[60] flex items-center justify-center p-4 animate-fade-in"
      onClick={() => setOpen(false)}
      role="dialog"
      aria-modal="true"
      aria-label={t('shortcuts.title')}
    >
      <div className="absolute inset-0 bg-ink-1/40 backdrop-blur-sm dark:bg-black/60" />
      <div
        onClick={(e) => e.stopPropagation()}
        className={cn(
          'relative w-full max-w-2xl bg-surface rounded-2xl shadow-modal border border-ink-5/30 overflow-hidden animate-scale-in',
          'dark:bg-[#13131F] dark:border-white/10',
        )}
      >
        <div className="flex items-center justify-between px-5 py-3 border-b border-ink-5/20 dark:border-white/5">
          <h2 className="text-sm font-semibold text-ink-1 flex items-center gap-2 dark:text-white">
            <Keyboard className="w-4 h-4 text-brand-500" />
            {t('shortcuts.title')}
          </h2>
          <button
            onClick={() => setOpen(false)}
            className="h-7 w-7 grid place-items-center rounded-lg text-ink-3 hover:text-ink-1 hover:bg-ink-1/5 dark:text-white/60 dark:hover:text-white dark:hover:bg-white/5"
            aria-label="Close"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-2 gap-x-8 gap-y-5 p-5 max-h-[70vh] overflow-y-auto">
          {GROUPS.map((g) => (
            <div key={g.label}>
              <p className="text-[10px] font-bold uppercase tracking-wider text-ink-4 mb-2 dark:text-white/40">{g.label}</p>
              <ul className="space-y-1.5">
                {g.items.map((s) => (
                  <li key={s.label} className="flex items-center justify-between gap-3">
                    <span className="text-[13px] text-ink-2 dark:text-white/80">{s.label}</span>
                    <span className="flex items-center gap-1 shrink-0">
                      {s.keys.map((k, i) => (
                        <kbd
                          key={i}
                          className="font-mono text-[10px] font-semibold px-1.5 py-0.5 min-w-[20px] text-center rounded bg-ink-5/30 border border-ink-5/40 text-ink-1 dark:bg-white/10 dark:border-white/10 dark:text-white/90"
                        >
                          {k}
                        </kbd>
                      ))}
                    </span>
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>

        <div className="px-5 py-2.5 text-[11px] text-ink-4 border-t border-ink-5/20 dark:text-white/40 dark:border-white/5">
          Tip: Press <kbd className="font-mono text-[10px] px-1 rounded bg-ink-5/20 dark:bg-white/10">?</kbd> any time outside a text field to reopen.
        </div>
      </div>
    </div>
  )
}
