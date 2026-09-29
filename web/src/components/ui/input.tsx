import * as React from 'react'
import { cn } from '@/lib/utils'

function Input({ className, type, ...props }: React.ComponentProps<'input'>) {
  return (
    <input
      type={type}
      data-slot="input"
      className={cn(
        'flex h-10 w-full min-w-0 rounded-[var(--r-thumb)] border border-[var(--hairline)] bg-surface px-3 py-1 text-[14px] text-foreground transition-[border-color,box-shadow,background-color] duration-200 ease-[var(--ease-out-quart)] outline-none',
        'placeholder:text-[var(--text-4)] selection:bg-[rgba(0,113,227,0.18)]',
        'hover:bg-hover',
        'focus-visible:border-[var(--accent)] focus-visible:bg-surface focus-visible:shadow-[0_0_0_4px_rgba(0,113,227,0.18)] dark:focus-visible:shadow-[0_0_0_4px_rgba(10,132,255,0.28)]',
        'aria-invalid:border-destructive aria-invalid:shadow-[0_0_0_4px_rgba(215,0,21,0.14)]',
        'file:inline-flex file:h-7 file:border-0 file:bg-transparent file:text-[13px] file:font-medium file:text-foreground',
        'disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-40',
        className,
      )}
      {...props}
    />
  )
}

export { Input }
