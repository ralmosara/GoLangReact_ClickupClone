import { useState } from 'react'
import { useLogin } from '../../hooks/useAuth'
import { Button, Input } from '../../components/ui'

export function LoginPage() {
  // Self-registration was removed in favour of admin-driven user creation.
  // The only path to a new account is a workspace admin inviting the user
  // via Members → Invite. This page is therefore login-only.
  const [email, setEmail]       = useState('')
  const [password, setPassword] = useState('')

  const login  = useLogin()
  const errMsg = login.error ? 'Invalid email or password. Please try again.' : null

  return (
    <div className="min-h-screen flex bg-canvas-gradient">
      {/* Left decorative panel */}
      <div className="hidden lg:flex w-[44%] flex-col justify-between bg-sidebar p-12 relative overflow-hidden">
        <div className="absolute top-[-80px] left-[-80px] w-[360px] h-[360px] bg-brand-500/10 rounded-full blur-3xl pointer-events-none" />
        <div className="absolute bottom-[-40px] right-[-60px] w-[280px] h-[280px] bg-purple-500/10 rounded-full blur-3xl pointer-events-none" />

        <div className="relative">
          <div className="flex items-center gap-2 mb-16">
            <div className="w-7 h-7 rounded-lg bg-brand-gradient flex items-center justify-center">
              <svg className="w-4 h-4 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2.5} d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
              </svg>
            </div>
            <span className="text-white font-semibold text-sm tracking-wide">ClickUp</span>
          </div>

          <h1 className="text-3xl font-bold text-white leading-tight mb-3">
            Ship more.<br />
            <span className="text-gradient">Track better.</span>
          </h1>
          <p className="text-sidebar-text text-sm leading-relaxed max-w-xs">
            A modern project management experience built for teams who move fast.
          </p>
        </div>

        {/* Feature list */}
        <div className="relative space-y-4">
          {[
            { icon: '⚡', label: 'Real-time collaboration via WebSocket' },
            { icon: '📋', label: 'Kanban boards with drag-free flow' },
            { icon: '🔔', label: 'Smart notifications, zero noise' },
          ].map((f) => (
            <div key={f.label} className="flex items-center gap-3">
              <span className="text-lg">{f.icon}</span>
              <span className="text-sidebar-text text-sm">{f.label}</span>
            </div>
          ))}
        </div>
      </div>

      {/* Right auth panel */}
      <div className="flex-1 flex items-center justify-center p-8">
        <div className="w-full max-w-[380px] animate-slide-up">
          {/* Logo for small screens */}
          <div className="flex items-center gap-2 mb-10 lg:hidden">
            <div className="w-7 h-7 rounded-lg bg-brand-gradient flex items-center justify-center">
              <svg className="w-4 h-4 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2.5} d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
              </svg>
            </div>
            <span className="text-ink-1 font-semibold text-sm">ClickUp</span>
          </div>

          <h2 className="text-2xl font-bold text-ink-1 mb-1">Welcome back</h2>
          <p className="text-sm text-ink-3 mb-7">Sign in to your workspace</p>

          <form
            className="space-y-4"
            onSubmit={(e) => { e.preventDefault(); login.mutate({ email, password }) }}
          >
            <Input
              label="Email"
              type="email"
              placeholder="you@company.com"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              autoFocus
            />
            <Input
              label="Password"
              type="password"
              placeholder="••••••••"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
            />

            {errMsg && (
              <div className="flex items-center gap-2 text-red-600 bg-red-50 border border-red-200 rounded-lg px-3 py-2.5 text-sm animate-fade-in">
                <svg className="w-4 h-4 shrink-0" fill="currentColor" viewBox="0 0 20 20">
                  <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
                </svg>
                {errMsg}
              </div>
            )}

            <Button
              type="submit"
              size="lg"
              loading={login.isPending}
              disabled={!email || !password}
              className="w-full mt-2"
            >
              Sign in
            </Button>
          </form>

          <p className="text-center text-xs text-ink-4 mt-8">
            Need an account? Ask your workspace admin to invite you.
          </p>
        </div>
      </div>
    </div>
  )
}
