import { type InputHTMLAttributes, forwardRef } from 'react'
import { cn } from '../../lib/utils'

interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
  label?: string
  hint?: string
  error?: string
  prefixNode?: React.ReactNode
}

export const Input = forwardRef<HTMLInputElement, InputProps>(
  ({ label, hint, error, prefixNode, className, id, ...props }, ref) => {
    const inputId = id ?? label?.toLowerCase().replace(/\s+/g, '-')
    return (
      <div className="w-full">
        {label && (
          <label htmlFor={inputId} className="block text-xs font-semibold text-ink-2 mb-1.5 tracking-wide">
            {label}
          </label>
        )}
        <div className="relative flex items-center">
          {prefixNode && (
            <span className="absolute left-3 text-ink-4 pointer-events-none">{prefixNode}</span>
          )}
          <input
            id={inputId}
            ref={ref}
            className={cn(
              'w-full h-9 bg-white border rounded text-sm text-ink-1 transition-all duration-150',
              'placeholder:text-ink-4/70',
              'focus:outline-none focus:ring-2 focus:ring-brand-500/20 focus:border-brand-500',
              error
                ? 'border-red-400 focus:ring-red-400/20 focus:border-red-400'
                : 'border-ink-5/80 hover:border-ink-4/60',
              prefixNode ? 'pl-9 pr-3' : 'px-3',
              className,
            )}
            {...props}
          />
        </div>
        {(hint || error) && (
          <p className={cn('text-xs mt-1.5', error ? 'text-red-500' : 'text-ink-4')}>
            {error ?? hint}
          </p>
        )}
      </div>
    )
  },
)
Input.displayName = 'Input'
