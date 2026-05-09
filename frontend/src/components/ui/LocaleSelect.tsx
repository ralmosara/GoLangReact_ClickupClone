import { LOCALES, getLocale, setLocale, useLocale, type Locale } from '../../lib/i18n'

export function LocaleSelect() {
  useLocale() // re-render when locale flips
  return (
    <select
      value={getLocale()}
      onChange={(e) => setLocale(e.target.value as Locale)}
      className="h-8 rounded-md border border-ink-5/40 bg-surface px-2 text-xs font-medium text-ink-2 focus:outline-none focus:ring-2 focus:ring-brand-400/40 dark:border-white/10 dark:bg-white/5 dark:text-white/80"
      aria-label="Language"
    >
      {LOCALES.map((l) => (
        <option key={l.code} value={l.code}>{l.label}</option>
      ))}
    </select>
  )
}
