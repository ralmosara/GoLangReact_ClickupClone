import { useState } from 'react'
import { useLogin, useRegister } from '../../hooks/useAuth'
import { Button, Input } from '../../components/ui'
import { cn } from '../../lib/utils'

type Mode = 'login' | 'register'

export function LoginPage() {
  const [mode, setMode] = useState<Mode>('login')
  const [email, setEmail]       = useState('')
  const [password, setPassword] = useState('')
  const [name, setName]         = useState('')

  const login    = useLogin()
  const register = useRegister()
  const mut      = mode === 'login' ? login : register
  const errMsg   = mut.error ? 'Invalid email or password. Please try again.' : null

  const submit = () =>
    mode === 'login'
      ? login.mutate({ email, password })
      : register.mutate({ email, password, name })

  return (
    <div className="min-h-screen flex bg-canvas-gradient">
      {/* Left decorative panel */}
      <div className="hidden lg:flex w-[44%] flex-col justify-between bg-sidebar p-12 relative overflow-hidden">
        {/* Gradient orbs */}
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

          {/* Mode tabs */}
          <div className="inline-flex bg-ink-5/40 rounded-lg p-0.5 mb-8">
            {(['login', 'register'] as Mode[]).map((m) => (
              <button
                key={m}
                onClick={() => setMode(m)}
                className={cn(
                  'px-5 py-1.5 text-sm font-medium rounded-md transition-all',
                  mode === m
                    ? 'bg-white text-ink-1 shadow-card'
                    : 'text-ink-3 hover:text-ink-2',
                )}
              >
                {m === 'login' ? 'Sign in' : 'Create account'}
              </button>
            ))}
          </div>

          <h2 className="text-2xl font-bold text-ink-1 mb-1">
            {mode === 'login' ? 'Welcome back' : 'Get started free'}
          </h2>
          <p className="text-sm text-ink-3 mb-7">
            {mode === 'login'
              ? 'Sign in to your workspace'
              : 'Create your account in seconds'}
          </p>

          <form
            className="space-y-4"
            onSubmit={(e) => { e.preventDefault(); submit() }}
          >
            {mode === 'register' && (
              <Input
                label="Full name"
                type="text"
                placeholder="Jane Smith"
                value={name}
                onChange={(e) => setName(e.target.value)}
                autoFocus={mode === 'register'}
              />
            )}
            <Input
              label="Email"
              type="email"
              placeholder="you@company.com"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              autoFocus={mode === 'login'}
            />
            <Input
              label="Password"
              type="password"
              placeholder={mode === 'register' ? 'Minimum 8 characters' : '••••••••'}
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
              loading={mut.isPending}
              disabled={!email || !password || (mode === 'register' && !name)}
              className="w-full mt-2"
            >
              {mode === 'login' ? 'Sign in' : 'Create account'}
            </Button>
          </form>

          <p className="text-center text-xs text-ink-4 mt-8">
            {mode === 'login' ? "Don't have an account?" : 'Already have an account?'}{' '}
            <button
              onClick={() => setMode(mode === 'login' ? 'register' : 'login')}
              className="text-brand-600 font-medium hover:text-brand-700 transition-colors"
            >
              {mode === 'login' ? 'Sign up' : 'Sign in'}
            </button>
          </p>
        </div>
      </div>
    </div>
  )
}
