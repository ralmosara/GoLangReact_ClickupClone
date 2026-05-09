// Lightweight theme system: persists user choice in localStorage, listens to
// the OS `prefers-color-scheme` when set to "system", and toggles a `dark`
// class + `data-theme` attribute on <html> so Tailwind's `dark:` variants
// activate. Kept dependency-free (no context provider) to avoid touching
// every page — components import the hook only where needed.

import { useEffect, useSyncExternalStore } from 'react'

export type ThemeMode = 'light' | 'dark' | 'system'
const STORAGE_KEY = 'clickup-theme'

let current: ThemeMode = (() => {
  if (typeof window === 'undefined') return 'system'
  const v = window.localStorage.getItem(STORAGE_KEY)
  return v === 'light' || v === 'dark' || v === 'system' ? v : 'system'
})()

const listeners = new Set<() => void>()
const subscribe = (cb: () => void) => {
  listeners.add(cb)
  return () => listeners.delete(cb)
}
const getSnapshot = () => current

function resolveDark(mode: ThemeMode): boolean {
  if (mode === 'dark') return true
  if (mode === 'light') return false
  return window.matchMedia('(prefers-color-scheme: dark)').matches
}

function applyTheme(mode: ThemeMode) {
  if (typeof document === 'undefined') return
  const isDark = resolveDark(mode)
  const root = document.documentElement
  root.classList.toggle('dark', isDark)
  root.setAttribute('data-theme', isDark ? 'dark' : 'light')
}

export function setTheme(next: ThemeMode) {
  current = next
  if (typeof window !== 'undefined') {
    window.localStorage.setItem(STORAGE_KEY, next)
  }
  applyTheme(next)
  listeners.forEach((l) => l())
}

export function useTheme() {
  const mode = useSyncExternalStore(subscribe, getSnapshot, () => 'system' as ThemeMode)
  return mode
}

// Mount once near the root. Applies the persisted theme on boot and re-applies
// when the OS scheme flips while in "system" mode.
export function ThemeBootstrap() {
  useEffect(() => {
    applyTheme(current)
    const mq = window.matchMedia('(prefers-color-scheme: dark)')
    const onChange = () => {
      if (current === 'system') applyTheme('system')
    }
    mq.addEventListener('change', onChange)
    return () => mq.removeEventListener('change', onChange)
  }, [])
  return null
}
