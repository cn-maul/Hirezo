import * as React from 'react'
import { cn } from '@/lib/utils'

function Textarea({ className, ...props }: React.ComponentProps<'textarea'>) {
  return (
    <textarea
      data-slot="textarea"
      className={cn(
        'flex field-sizing-content min-h-20 w-full rounded-[var(--r-thumb)] border border-[var(--hairline)] bg-surface px-3 py-2.5 text-[14px] leading-relaxed text-foreground transition-[border-color,box-shadow,background-color] duration-200 ease-[var(--ease-out-quart)] outline-none',
        'placeholder:text-[var(--text-4)]',
        'hover:bg-hover',
        'focus-visible:border-[var(--accent)] focus-visible:bg-surface focus-visible:shadow-[0_0_0_4px_rgba(0,113,227,0.18)] dark:focus-visible:shadow-[0_0_0_4px_rgba(10,132,255,0.28)]',
        'aria-invalid:border-destructive aria-invalid:shadow-[0_0_0_4px_rgba(215,0,21,0.14)]',
        'resize-none disabled:cursor-not-allowed disabled:opacity-40',
        className,
      )}
      {...props}
    />
  )
}

export { Textarea }
