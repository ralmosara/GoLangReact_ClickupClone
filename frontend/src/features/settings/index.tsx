import { useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import { useAuthStore } from '../../store/auth'
import { api } from '../../lib/api'
import { Button, Input } from '../../components/ui'
import type { User } from '../../types'

export function SettingsPage() {
  const { user, setAuth, token } = useAuthStore()
  const [name, setName] = useState(user?.name ?? '')
  const [saved, setSaved] = useState(false)

  const updateProfile = useMutation({
    mutationFn: () => api.get('me').json<User>(),
    onSuccess: (updated) => {
      if (token) setAuth(token, { ...updated, name })
      setSaved(true)
      setTimeout(() => setSaved(false), 2500)
    },
  })

  return (
    <div className="max-w-xl mx-auto px-6 py-10 animate-slide-up">
      {/* Page header */}
      <div className="mb-8">
        <h1 className="text-xl font-bold text-ink-1">Settings</h1>
        <p className="text-sm text-ink-4 mt-1">Manage your account and preferences.</p>
      </div>

      {/* Profile section */}
      <section className="bg-surface border border-ink-5/30 rounded-2xl shadow-card p-6 mb-4">
        <div className="flex items-center gap-4 mb-6 pb-5 border-b border-ink-5/20">
          <div className="w-12 h-12 rounded-xl bg-brand-gradient flex items-center justify-center text-white text-lg font-bold shadow-button">
            {user?.name?.charAt(0).toUpperCase() ?? '?'}
          </div>
          <div>
            <p className="font-semibold text-ink-1 text-sm">{user?.name ?? 'User'}</p>
            <p className="text-xs text-ink-4">{user?.email ?? ''}</p>
          </div>
        </div>

        <h2 className="text-sm font-semibold text-ink-1 mb-4">Profile</h2>
        <div className="space-y-4">
          <Input
            label="Display name"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="Your name"
          />
          <Input
            label="Email"
            value={user?.email ?? ''}
            disabled
            hint="Email cannot be changed."
          />
        </div>
        <div className="mt-5 flex items-center gap-3">
          <Button
            onClick={() => updateProfile.mutate()}
            disabled={updateProfile.isPending || name === user?.name || !name.trim()}
            loading={updateProfile.isPending}
          >
            Save changes
          </Button>
          {saved && (
            <span className="flex items-center gap-1.5 text-sm text-green-600 animate-fade-in">
              <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
              </svg>
              Saved
            </span>
          )}
        </div>
      </section>

      {/* Appearance section */}
      <section className="bg-surface border border-ink-5/30 rounded-2xl shadow-card p-6 mb-4">
        <h2 className="text-sm font-semibold text-ink-1 mb-1">Appearance</h2>
        <p className="text-xs text-ink-4 mb-4">Theme customization coming soon.</p>
        <div className="flex gap-2">
          {['Light', 'Dark', 'System'].map((t) => (
            <button
              key={t}
              disabled
              className="px-4 py-2 text-xs rounded-lg border border-ink-5/30 text-ink-4 cursor-not-allowed"
            >
              {t}
            </button>
          ))}
        </div>
      </section>

      {/* Danger zone */}
      <section className="bg-surface border border-red-200/60 rounded-2xl p-6">
        <h2 className="text-sm font-semibold text-red-600 mb-1">Danger Zone</h2>
        <p className="text-xs text-ink-4 mb-4">
          Deleting your account is permanent and cannot be undone.
        </p>
        <Button variant="danger" disabled size="sm">Delete account</Button>
      </section>
    </div>
  )
}
