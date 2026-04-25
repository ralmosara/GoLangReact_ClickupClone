import { type ButtonHTMLAttributes, forwardRef } from 'react'
import { cn } from '../../lib/utils'

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'primary' | 'secondary' | 'ghost' | 'danger'
  size?: 'xs' | 'sm' | 'md' | 'lg'
  loading?: boolean
}

const variants: Record<string, string> = {
  primary: [
    'bg-brand-gradient text-white shadow-button',
    'hover:opacity-90 hover:shadow-lg',
    'active:scale-[0.98] active:opacity-100',
    'disabled:opacity-40 disabled:shadow-none disabled:cursor-not-allowed',
  ].join(' '),
  secondary: [
    'bg-white text-ink-1 border border-ink-5/60 shadow-card',
    'hover:border-ink-4/60 hover:bg-surface',
    'active:scale-[0.98]',
    'disabled:opacity-40 disabled:cursor-not-allowed',
  ].join(' '),
  ghost: [
    'text-ink-2',
    'hover:bg-ink-1/5 hover:text-ink-1',
    'active:scale-[0.98]',
    'disabled:opacity-40 disabled:cursor-not-allowed',
  ].join(' '),
  danger: [
    'bg-red-50 text-red-600 border border-red-200',
    'hover:bg-red-100 hover:border-red-300',
    'active:scale-[0.98]',
    'disabled:opacity-40 disabled:cursor-not-allowed',
  ].join(' '),
}

const sizes: Record<string, string> = {
  xs: 'h-6  px-2   text-[11px] font-medium rounded-sm gap-1',
  sm: 'h-8  px-3   text-xs     font-medium rounded    gap-1.5',
  md: 'h-9  px-4   text-sm     font-medium rounded    gap-2',
  lg: 'h-11 px-5   text-base   font-semibold rounded-lg gap-2',
}

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(
  ({ variant = 'primary', size = 'md', loading, className, children, disabled, ...props }, ref) => (
    <button
      ref={ref}
      disabled={disabled || loading}
      className={cn(
        'inline-flex items-center justify-center transition-all duration-150',
        'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-500/50 focus-visible:ring-offset-1',
        variants[variant],
        sizes[size],
        className,
      )}
      {...props}
    >
      {loading ? (
        <span className="inline-block w-3.5 h-3.5 border-2 border-current border-t-transparent rounded-full animate-spin" />
      ) : children}
    </button>
  ),
)
Button.displayName = 'Button'
