// Tiny i18n scaffold. Not a dependency-heavy lib (i18next/react-intl) on
// purpose — strings live in this file as a flat key→string map per locale,
// with English as the source-of-truth. The runtime just looks the key up
// and falls back to English when a translation is missing. Switching the
// active locale persists to localStorage and broadcasts to subscribers
// (used by useLocale below).
//
// To add a new locale: drop a new entry into `messages` keyed by ISO code,
// and copy/translate the en keys you want covered. Untranslated keys stay
// in English — never crash on missing keys.

import { useSyncExternalStore } from 'react'

export type Locale = 'en' | 'es' | 'fr' | 'de' | 'pt' | 'ja'

export const LOCALES: { code: Locale; label: string }[] = [
  { code: 'en', label: 'English' },
  { code: 'es', label: 'Español' },
  { code: 'fr', label: 'Français' },
  { code: 'de', label: 'Deutsch' },
  { code: 'pt', label: 'Português' },
  { code: 'ja', label: '日本語' },
]

// Source-of-truth keys live in `en`. Other locales need not be exhaustive —
// missing keys fall through. Keep keys hierarchical (settings.profile.title).
const messages: Record<Locale, Record<string, string>> = {
  en: {
    'settings.title':                'Settings',
    'settings.profile.tab':          'Profile',
    'settings.security.tab':         'Security',
    'settings.notifications.tab':    'Notifications',
    'settings.integrations.tab':     'Integrations',
    'settings.appearance.tab':       'Appearance',
    'settings.workspace.tab':        'Workspace',
    'settings.appearance.theme':     'Theme',
    'settings.appearance.language':  'Language',
    'theme.light':                   'Light',
    'theme.dark':                    'Dark',
    'theme.system':                  'System',
    'common.save':                   'Save changes',
    'common.cancel':                 'Cancel',
    'common.delete':                 'Delete',
    'common.create':                 'Create',
    'common.loading':                'Loading…',
    'common.saved':                  'Saved',
    'sessions.title':                'Active sessions',
    'sessions.signout_others':       'Sign out all other devices',
    'sessions.this_device':          'This device',
    'shortcuts.title':               'Keyboard shortcuts',
  },
  es: {
    'settings.title':             'Configuración',
    'settings.profile.tab':       'Perfil',
    'settings.security.tab':      'Seguridad',
    'settings.notifications.tab': 'Notificaciones',
    'settings.integrations.tab':  'Integraciones',
    'settings.appearance.tab':    'Apariencia',
    'theme.light': 'Claro', 'theme.dark': 'Oscuro', 'theme.system': 'Sistema',
    'common.save': 'Guardar', 'common.cancel': 'Cancelar',
  },
  fr: {
    'settings.title':             'Paramètres',
    'settings.profile.tab':       'Profil',
    'settings.security.tab':      'Sécurité',
    'theme.light': 'Clair', 'theme.dark': 'Sombre', 'theme.system': 'Système',
  },
  de: {
    'settings.title':             'Einstellungen',
    'theme.light': 'Hell', 'theme.dark': 'Dunkel', 'theme.system': 'System',
  },
  pt: {
    'settings.title':             'Configurações',
    'theme.light': 'Claro', 'theme.dark': 'Escuro', 'theme.system': 'Sistema',
  },
  ja: {
    'settings.title':             '設定',
    'theme.light': 'ライト', 'theme.dark': 'ダーク', 'theme.system': 'システム',
  },
}

const STORAGE_KEY = 'clickup-locale'
let current: Locale = (() => {
  if (typeof window === 'undefined') return 'en'
  const v = window.localStorage.getItem(STORAGE_KEY) as Locale | null
  return v && messages[v] ? v : 'en'
})()

const listeners = new Set<() => void>()
const subscribe = (cb: () => void) => { listeners.add(cb); return () => listeners.delete(cb) }

export function setLocale(next: Locale) {
  current = next
  if (typeof window !== 'undefined') window.localStorage.setItem(STORAGE_KEY, next)
  listeners.forEach((l) => l())
}

export function getLocale(): Locale { return current }

// t(key) returns the message for the current locale, falling back to en,
// then to the key itself so missing translations are visible without
// throwing. Optional params do printf-style {name} interpolation.
export function t(key: string, params?: Record<string, string | number>): string {
  const raw = messages[current]?.[key] ?? messages.en[key] ?? key
  if (!params) return raw
  // Use a global regex instead of String.prototype.replaceAll so the
  // module compiles under tsconfig lib=ES2020. Escape the key so a
  // literal `{` in the param name doesn't break the pattern.
  return Object.entries(params).reduce(
    (acc, [k, v]) => acc.replace(new RegExp(`\\{${escapeRe(k)}\\}`, 'g'), String(v)),
    raw,
  )
}

function escapeRe(s: string): string {
  return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

export function useLocale(): Locale {
  return useSyncExternalStore(subscribe, () => current, () => 'en')
}
