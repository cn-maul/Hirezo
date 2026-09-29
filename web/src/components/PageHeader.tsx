import type { ReactNode } from 'react'
import { ArrowLeft } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

export default function PageHeader({
  title,
  description,
  action,
  onBack,
  className,
}: {
  title: ReactNode
  description?: ReactNode
  action?: ReactNode
  onBack?: () => void
  className?: string
}) {
  return (
    <div className={cn('flex flex-wrap items-start justify-between gap-3', className)}>
      <div className="min-w-0">
        <div className="flex flex-wrap items-center gap-2">
          {onBack && (
            <Button variant="ghost" size="icon" onClick={onBack} aria-label="返回">
              <ArrowLeft />
            </Button>
          )}
          <h1 className="text-[26px] leading-tight font-semibold tracking-[-0.025em] text-foreground">
            {title}
          </h1>
        </div>
        {description && (
          <p className={cn('mt-1.5 text-[14px] text-[var(--text-2)]', onBack && 'pl-12')}>
            {description}
          </p>
        )}
      </div>
      {action && <div className="flex flex-wrap items-center gap-2">{action}</div>}
    </div>
  )
}
