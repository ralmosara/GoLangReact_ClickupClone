import { useState } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import { Check, Key, ShieldCheck, ShieldOff } from 'lucide-react'

import { api } from '../../../lib/api'
import { queryClient } from '../../../lib/queryClient'
import { Button, Input } from '../../../components/ui'
import { ActiveSessionsPanel } from '../components/ActiveSessionsPanel'

interface MFAStatus {
  enrolled: boolean
  remaining_recovery_codes: number
}

export function SecurityTab() {
  return (
    <div className="space-y-4">
      <ChangePasswordPanel />
      <MFAPanel />
      <ActiveSessionsPanel />
    </div>
  )
}

function ChangePasswordPanel() {
  const [current, setCurrent] = useState('')
  const [next, setNext] = useState('')
  const [confirm, setConfirm] = useState('')
  const [done, setDone] = useState(false)

  const change = useMutation({
    mutationFn: () =>
      api.post('me/password', {
        json: { current_password: current, new_password: next },
      }),
    onSuccess: () => {
      setDone(true)
      setCurrent(''); setNext(''); setConfirm('')
      setTimeout(() => setDone(false), 3000)
    },
  })

  const mismatch = next.length > 0 && confirm.length > 0 && next !== confirm
  const tooShort = next.length > 0 && next.length < 8
  const canSubmit = !!current && !!next && !!confirm && !mismatch && !tooShort && !change.isPending

  return (
    <section className="bg-surface border border-ink-5/30 rounded-2xl shadow-card p-6 dark:bg-white/[0.04] dark:border-white/10">
      <h2 className="text-sm font-semibold text-ink-1 flex items-center gap-2 mb-1 dark:text-white">
        <Key className="w-4 h-4 text-brand-500" />
        Change password
      </h2>
      <p className="text-xs text-ink-4 mb-5 dark:text-white/50">
        Pick something at least 8 characters. Other devices stay signed in unless you also revoke them below.
      </p>
      <div className="space-y-3 max-w-sm">
        <Input
          type="password"
          label="Current password"
          value={current}
          onChange={(e) => setCurrent(e.target.value)}
          autoComplete="current-password"
        />
        <Input
          type="password"
          label="New password"
          value={next}
          onChange={(e) => setNext(e.target.value)}
          autoComplete="new-password"
          hint={tooShort ? 'Must be at least 8 characters.' : undefined}
        />
        <Input
          type="password"
          label="Confirm new password"
          value={confirm}
          onChange={(e) => setConfirm(e.target.value)}
          autoComplete="new-password"
          hint={mismatch ? 'Passwords do not match.' : undefined}
        />
      </div>
      {change.error && (
        <p className="text-xs text-red-500 mt-3">{String(change.error)}</p>
      )}
      <div className="mt-5 flex items-center gap-3">
        <Button onClick={() => change.mutate()} disabled={!canSubmit} loading={change.isPending}>
          Update password
        </Button>
        {done && (
          <span className="text-sm text-green-600 inline-flex items-center gap-1.5 dark:text-green-400">
            <Check className="w-4 h-4" /> Password updated
          </span>
        )}
      </div>
    </section>
  )
}

function MFAPanel() {
  const { data: status, isLoading } = useQuery({
    queryKey: ['mfa-status'],
    queryFn: () => api.get('me/mfa/status').json<MFAStatus>(),
  })

  const [enrolling, setEnrolling] = useState<{ uri: string; secret: string } | null>(null)
  const [code, setCode] = useState('')
  const [recoveryCodes, setRecoveryCodes] = useState<string[] | null>(null)

  const enroll = useMutation({
    mutationFn: () => api.post('me/mfa/enroll').json<{ uri: string; secret: string }>(),
    onSuccess: (res) => setEnrolling(res),
  })

  const verify = useMutation({
    mutationFn: () => api.post('me/mfa/verify-enroll', { json: { code } }).json<{ recovery_codes: string[] }>(),
    onSuccess: (res) => {
      setRecoveryCodes(res.recovery_codes)
      setEnrolling(null)
      setCode('')
      queryClient.invalidateQueries({ queryKey: ['mfa-status'] })
    },
  })

  const disable = useMutation({
    mutationFn: () => api.delete('me/mfa'),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['mfa-status'] }),
  })

  return (
    <section className="bg-surface border border-ink-5/30 rounded-2xl shadow-card p-6 dark:bg-white/[0.04] dark:border-white/10">
      <h2 className="text-sm font-semibold text-ink-1 flex items-center gap-2 mb-1 dark:text-white">
        {status?.enrolled
          ? <ShieldCheck className="w-4 h-4 text-green-500" />
          : <ShieldOff className="w-4 h-4 text-ink-3 dark:text-white/40" />}
        Two-factor authentication
      </h2>
      <p className="text-xs text-ink-4 mb-5 dark:text-white/50">
        {status?.enrolled
          ? `Enabled. ${status.remaining_recovery_codes} recovery codes remaining.`
          : 'Add a TOTP app (Google Authenticator, 1Password, Authy) so a leaked password isn\'t enough to sign in.'}
      </p>

      {isLoading && <p className="text-xs text-ink-4">Loading…</p>}

      {!isLoading && !status?.enrolled && !enrolling && (
        <Button size="sm" onClick={() => enroll.mutate()} loading={enroll.isPending}>
          Set up 2FA
        </Button>
      )}

      {enrolling && (
        <div className="border border-brand-200 bg-brand-50 rounded-xl p-4 space-y-3 dark:bg-brand-500/10 dark:border-brand-500/30">
          <p className="text-xs text-ink-2 dark:text-white/80">
            Scan this URI in your authenticator app, or enter the secret manually:
          </p>
          <pre className="font-mono text-[10px] bg-white p-2 rounded border border-ink-5/30 break-all whitespace-pre-wrap dark:bg-white/5 dark:border-white/10 dark:text-white/80">
            {enrolling.uri}
          </pre>
          <p className="text-[11px] text-ink-3 dark:text-white/50">Secret: <span className="font-mono">{enrolling.secret}</span></p>
          <div className="flex items-end gap-2">
            <Input
              label="Confirm with the 6-digit code"
              value={code}
              onChange={(e) => setCode(e.target.value)}
              placeholder="123456"
              className="flex-1"
            />
            <Button onClick={() => verify.mutate()} disabled={code.length < 6 || verify.isPending} loading={verify.isPending}>
              Verify
            </Button>
          </div>
          {verify.error && <p className="text-xs text-red-500">{String(verify.error)}</p>}
        </div>
      )}

      {recoveryCodes && (
        <div className="border border-amber-200 bg-amber-50 rounded-xl p-4 mt-4 dark:bg-amber-500/10 dark:border-amber-500/30">
          <p className="text-xs font-semibold text-amber-800 mb-2 dark:text-amber-200">
            Save these recovery codes — they're shown only once.
          </p>
          <div className="grid grid-cols-2 gap-1 font-mono text-[11px] text-ink-2 dark:text-white/80">
            {recoveryCodes.map((c) => <span key={c}>{c}</span>)}
          </div>
          <button
            onClick={() => setRecoveryCodes(null)}
            className="mt-2 text-[11px] text-amber-700 hover:text-amber-900 underline dark:text-amber-200"
          >
            I've saved them
          </button>
        </div>
      )}

      {status?.enrolled && !enrolling && !recoveryCodes && (
        <Button
          variant="danger"
          size="sm"
          onClick={() => {
            if (confirm('Disable two-factor authentication? Your account will accept passwords alone again.')) {
              disable.mutate()
            }
          }}
          loading={disable.isPending}
        >
          Disable 2FA
        </Button>
      )}
    </section>
  )
}
