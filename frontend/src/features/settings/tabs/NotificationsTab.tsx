import { useState } from 'react'

// NotificationsTab is currently UI-only — the backend doesn't yet have a
// notification_preferences table, so toggles persist to localStorage and
// inform the in-app rendering only. Wiring to email/push channels is on
// the roadmap; the shape below is intentionally close to what those
// channels will need so swapping the persistence layer is one PR.

type ChannelKey = 'inapp' | 'email' | 'push'
const CHANNELS: { key: ChannelKey; label: string; description: string; soon?: boolean }[] = [
  { key: 'inapp', label: 'In-app',          description: 'Notifications surface in the bell + Inbox.' },
  { key: 'email', label: 'Email',           description: 'Daily / instant email digests.', soon: true },
  { key: 'push',  label: 'Push (browser)',  description: 'Web push notifications when this tab is closed.', soon: true },
]

type EventKey = 'mention' | 'assign' | 'status_change' | 'comment' | 'due_soon'
const EVENTS: { key: EventKey; label: string; description: string }[] = [
  { key: 'mention',       label: '@mentions',         description: 'Someone @-mentions you in a comment or doc.' },
  { key: 'assign',        label: 'Assigned to me',    description: 'A task gets assigned to you.' },
  { key: 'status_change', label: 'Status changes',    description: 'Status of a task you watch changes.' },
  { key: 'comment',       label: 'Comment activity',  description: 'New comment on a task you authored or watch.' },
  { key: 'due_soon',      label: 'Due soon',          description: 'A task you own is due in the next 24h.' },
]

const STORAGE_KEY = 'clickup-notif-prefs'

type Prefs = Record<EventKey, Record<ChannelKey, boolean>>
function loadPrefs(): Prefs {
  if (typeof window === 'undefined') return defaultPrefs()
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY)
    if (!raw) return defaultPrefs()
    return { ...defaultPrefs(), ...JSON.parse(raw) }
  } catch {
    return defaultPrefs()
  }
}
function defaultPrefs(): Prefs {
  const out = {} as Prefs
  EVENTS.forEach((e) => {
    out[e.key] = { inapp: true, email: false, push: false }
  })
  return out
}

export function NotificationsTab() {
  const [prefs, setPrefs] = useState<Prefs>(loadPrefs())

  const toggle = (event: EventKey, channel: ChannelKey) => {
    const next: Prefs = {
      ...prefs,
      [event]: { ...prefs[event], [channel]: !prefs[event][channel] },
    }
    setPrefs(next)
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(next))
  }

  return (
    <section className="bg-surface border border-ink-5/30 rounded-2xl shadow-card p-6 dark:bg-white/[0.04] dark:border-white/10">
      <h2 className="text-sm font-semibold text-ink-1 mb-1 dark:text-white">Notification preferences</h2>
      <p className="text-xs text-ink-4 mb-5 dark:text-white/50">
        Choose which events surface where. In-app delivery is live today; email and push are queued for a follow-up release.
      </p>

      <div className="overflow-x-auto">
        <table className="w-full text-xs">
          <thead>
            <tr className="text-left text-ink-3 border-b border-ink-5/20 dark:text-white/50 dark:border-white/10">
              <th className="font-medium py-2 pr-4">Event</th>
              {CHANNELS.map((c) => (
                <th key={c.key} className="font-medium py-2 px-3 text-center">
                  {c.label}
                  {c.soon && <span className="block text-[9px] uppercase text-ink-4 mt-0.5">soon</span>}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {EVENTS.map((e) => (
              <tr key={e.key} className="border-b border-ink-5/10 last:border-0 dark:border-white/5">
                <td className="py-3 pr-4">
                  <p className="font-medium text-ink-1 dark:text-white">{e.label}</p>
                  <p className="text-[11px] text-ink-4 dark:text-white/40">{e.description}</p>
                </td>
                {CHANNELS.map((c) => (
                  <td key={c.key} className="py-3 px-3 text-center">
                    <input
                      type="checkbox"
                      checked={prefs[e.key][c.key]}
                      onChange={() => toggle(e.key, c.key)}
                      disabled={!!c.soon}
                      className="h-4 w-4 rounded border-ink-5/40 text-brand-500 focus:ring-brand-400/40 disabled:opacity-30"
                    />
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <p className="text-[11px] text-ink-4 mt-4 dark:text-white/40">
        Preferences are stored locally for now. Once the backend `notification_preferences` table ships, these will sync to your account.
      </p>
    </section>
  )
}
