import { useSearchParams } from 'react-router-dom'
import { Bell, Building2, Palette, Shield, User as UserIcon } from 'lucide-react'

import { cn } from '../../lib/utils'
import { t } from '../../lib/i18n'
import { ProfileTab } from './tabs/ProfileTab'
import { SecurityTab } from './tabs/SecurityTab'
import { AppearanceTab } from './tabs/AppearanceTab'
import { NotificationsTab } from './tabs/NotificationsTab'
import { WorkspaceTab } from './tabs/WorkspaceTab'

type TabKey = 'profile' | 'security' | 'notifications' | 'appearance' | 'workspace'

const TABS: { key: TabKey; labelKey: string; Icon: typeof UserIcon }[] = [
  { key: 'profile',       labelKey: 'settings.profile.tab',       Icon: UserIcon },
  { key: 'security',      labelKey: 'settings.security.tab',      Icon: Shield },
  { key: 'notifications', labelKey: 'settings.notifications.tab', Icon: Bell },
  { key: 'appearance',    labelKey: 'settings.appearance.tab',    Icon: Palette },
  { key: 'workspace',     labelKey: 'settings.workspace.tab',     Icon: Building2 },
]

/**
 * SettingsPage is now a tabbed hub. The active tab persists in the URL
 * (?tab=security) so a user can deep-link to a specific section, the
 * back button works as expected, and reloading lands them where they
 * were. Falls back to "profile" for legacy/missing values.
 */
export function SettingsPage() {
  const [params, setParams] = useSearchParams()
  const tab = (params.get('tab') as TabKey) || 'profile'
  const active: TabKey = TABS.some((t) => t.key === tab) ? tab : 'profile'

  return (
    <div className="max-w-4xl mx-auto px-6 py-8 animate-slide-up">
      <header className="mb-6">
        <h1 className="text-xl font-bold text-ink-1 dark:text-white">{t('settings.title')}</h1>
        <p className="text-sm text-ink-4 mt-1 dark:text-white/50">
          Manage your account, security, appearance, and workspace preferences.
        </p>
      </header>

      <div className="flex flex-col md:flex-row gap-6">
        {/* Vertical tab nav on desktop, scrollable horizontal pills on mobile */}
        <nav
          aria-label="Settings sections"
          className="md:w-48 shrink-0 flex md:flex-col gap-1 overflow-x-auto md:overflow-visible -mx-6 px-6 md:mx-0 md:px-0 pb-2 md:pb-0"
        >
          {TABS.map(({ key, labelKey, Icon }) => {
            const isActive = key === active
            return (
              <button
                key={key}
                onClick={() => setParams({ tab: key }, { replace: true })}
                aria-current={isActive ? 'page' : undefined}
                className={cn(
                  'inline-flex items-center gap-2 h-9 px-3 rounded-lg text-sm font-medium transition-colors shrink-0',
                  isActive
                    ? 'bg-brand-50 text-brand-700 dark:bg-brand-500/15 dark:text-brand-200'
                    : 'text-ink-2 hover:bg-ink-1/[0.04] hover:text-ink-1 dark:text-white/70 dark:hover:bg-white/5 dark:hover:text-white',
                )}
              >
                <Icon className="w-4 h-4" />
                {t(labelKey)}
              </button>
            )
          })}
        </nav>

        <div className="flex-1 min-w-0">
          {active === 'profile'       && <ProfileTab />}
          {active === 'security'      && <SecurityTab />}
          {active === 'notifications' && <NotificationsTab />}
          {active === 'appearance'    && <AppearanceTab />}
          {active === 'workspace'     && <WorkspaceTab />}
        </div>
      </div>
    </div>
  )
}
