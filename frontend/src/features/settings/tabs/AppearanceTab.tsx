import { ThemeToggle } from '../../../components/ui/ThemeToggle'
import { LocaleSelect } from '../../../components/ui/LocaleSelect'
import { t } from '../../../lib/i18n'

export function AppearanceTab() {
  return (
    <section className="bg-surface border border-ink-5/30 rounded-2xl shadow-card p-6 space-y-6 dark:bg-white/[0.04] dark:border-white/10">
      <div>
        <h2 className="text-sm font-semibold text-ink-1 mb-1 dark:text-white">{t('settings.appearance.theme')}</h2>
        <p className="text-xs text-ink-4 mb-3 dark:text-white/50">
          Pick a fixed scheme or follow your operating system.
        </p>
        <ThemeToggle />
      </div>

      <div className="border-t border-ink-5/20 dark:border-white/10" />

      <div>
        <h2 className="text-sm font-semibold text-ink-1 mb-1 dark:text-white">{t('settings.appearance.language')}</h2>
        <p className="text-xs text-ink-4 mb-3 dark:text-white/50">
          UI language. Most strings still fall back to English — the i18n catalog is being built out.
        </p>
        <LocaleSelect />
      </div>
    </section>
  )
}
