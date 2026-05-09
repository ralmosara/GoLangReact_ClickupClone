import { useEffect, useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import { Check } from 'lucide-react'

import { api } from '../../../lib/api'
import { useAuthStore } from '../../../store/auth'
import { Button, Input } from '../../../components/ui'
import { t } from '../../../lib/i18n'
import type { User } from '../../../types'

export function ProfileTab() {
  const { user, setAuth, token } = useAuthStore()
  const [name, setName] = useState(user?.name ?? '')
  const [saved, setSaved] = useState(false)

  // If the auth store rehydrates after first paint, re-seed the local
  // editable copy. Avoids the empty-name flash when the user navigates
  // straight into Settings.
  useEffect(() => { setName(user?.name ?? '') }, [user?.name])

  const update = useMutation({
    mutationFn: () => api.patch('me', { json: { name } }).json<User>(),
    onSuccess: (updated) => {
      if (token) setAuth(token, updated)
      setSaved(true)
      setTimeout(() => setSaved(false), 2500)
    },
  })

  const dirty = name.trim() !== (user?.name ?? '') && name.trim().length > 0

  return (
    <section className="bg-surface border border-ink-5/30 rounded-2xl shadow-card p-6 dark:bg-white/[0.04] dark:border-white/10">
      <div className="flex items-center gap-4 mb-6 pb-5 border-b border-ink-5/20 dark:border-white/10">
        <div className="w-12 h-12 rounded-xl bg-brand-gradient flex items-center justify-center text-white text-lg font-bold shadow-button">
          {user?.name?.charAt(0).toUpperCase() ?? '?'}
        </div>
        <div>
          <p className="font-semibold text-ink-1 text-sm dark:text-white">{user?.name ?? 'User'}</p>
          <p className="text-xs text-ink-4 dark:text-white/50">{user?.email ?? ''}</p>
        </div>
      </div>

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
          hint="Email cannot be changed yet — re-verification flow is on the roadmap."
        />
      </div>

      {update.error && (
        <p className="text-xs text-red-500 mt-3">{String(update.error)}</p>
      )}

      <div className="mt-5 flex items-center gap-3">
        <Button
          onClick={() => update.mutate()}
          disabled={!dirty || update.isPending}
          loading={update.isPending}
        >
          {t('common.save')}
        </Button>
        {saved && (
          <span className="flex items-center gap-1.5 text-sm text-green-600 animate-fade-in dark:text-green-400">
            <Check className="w-4 h-4" /> {t('common.saved')}
          </span>
        )}
      </div>
    </section>
  )
}
