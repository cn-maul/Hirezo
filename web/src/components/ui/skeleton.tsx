import { cn } from '@/lib/utils'

function Skeleton({ className, ...props }: React.ComponentProps<'div'>) {
  return (
    <div
      data-slot="skeleton"
      className={cn('animate-pulse rounded-[var(--r-chip)] bg-[var(--track)]', className)}
      {...props}
    />
  )
}

export { Skeleton }
