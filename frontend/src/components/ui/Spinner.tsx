import { cn } from '../../lib/utils'

export function Spinner({ className }: { className?: string }) {
  return (
    <div className={cn('inline-block rounded-full border-2 border-current border-t-transparent animate-spin', className ?? 'w-4 h-4')} />
  )
}

export function PageSpinner() {
  return (
    <div className="flex items-center justify-center h-full min-h-[200px]">
      <Spinner className="w-6 h-6 text-brand-500" />
    </div>
  )
}
