import { useEffect, type ReactNode } from 'react'
import { cn } from '../../lib/utils'

interface ModalProps {
  open: boolean
  onClose: () => void
  title?: string
  description?: string
  children: ReactNode
  width?: string
}

export function Modal({ open, onClose, title, description, children, width = 'max-w-md' }: ModalProps) {
  useEffect(() => {
    if (!open) return
    const handler = (e: KeyboardEvent) => { if (e.key === 'Escape') onClose() }
    document.body.style.overflow = 'hidden'
    window.addEventListener('keydown', handler)
    return () => {
      window.removeEventListener('keydown', handler)
      document.body.style.overflow = ''
    }
  }, [open, onClose])

  if (!open) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 animate-fade-in">
      <div className="absolute inset-0 bg-ink-1/30 backdrop-blur-sm" onClick={onClose} />
      <div className={cn('relative bg-surface w-full rounded-xl shadow-modal border border-ink-5/30 animate-scale-in', width)}>
        {(title || description) && (
          <div className="px-6 pt-6 pb-4 border-b border-ink-5/40">
            <div className="flex items-start justify-between gap-4">
              <div>
                {title && <h2 className="text-base font-semibold text-ink-1">{title}</h2>}
                {description && <p className="text-xs text-ink-3 mt-0.5">{description}</p>}
              </div>
              <button
                onClick={onClose}
                className="text-ink-4 hover:text-ink-2 hover:bg-ink-1/5 rounded p-1 transition-colors -mr-1 -mt-1"
              >
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>
          </div>
        )}
        <div className="p-6">{children}</div>
      </div>
    </div>
  )
}
