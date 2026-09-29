import * as React from 'react'
import { Slot } from '@radix-ui/react-slot'
import { cva, type VariantProps } from 'class-variance-authority'
import { cn } from '@/lib/utils'

const buttonVariants = cva(
  "inline-flex shrink-0 items-center justify-center gap-2 whitespace-nowrap rounded-[var(--r-pill)] text-[14px] font-medium tracking-[-0.008em] outline-none select-none transition-[transform,background-color,box-shadow,color,border-color] duration-200 ease-[var(--ease-out-quart)] disabled:pointer-events-none disabled:opacity-40 active:scale-[0.97] [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-[18px] focus-visible:shadow-[0_0_0_4px_rgba(0,113,227,0.25)] dark:focus-visible:shadow-[0_0_0_4px_rgba(10,132,255,0.35)]",
  {
    variants: {
      variant: {
        default:
          'bg-primary text-primary-foreground shadow-[var(--sh-card)] hover:brightness-105',
        destructive: 'bg-destructive text-white shadow-[var(--sh-card)] hover:brightness-105',
        outline:
          'border border-[var(--hairline)] bg-surface text-foreground hover:bg-hover',
        secondary: 'bg-track text-foreground hover:brightness-[0.97]',
        ghost: 'text-muted-foreground hover:bg-hover hover:text-foreground',
        link: 'text-primary underline-offset-4 hover:underline',
      },
      size: {
        default: 'h-10 px-4 py-2 has-[>svg]:px-3.5',
        sm: 'h-9 gap-1.5 px-3.5 has-[>svg]:px-3',
        lg: 'h-11 px-6 has-[>svg]:px-4.5',
        icon: 'size-10',
      },
    },
    defaultVariants: {
      variant: 'default',
      size: 'default',
    },
  },
)

function Button({
  className,
  variant,
  size,
  asChild = false,
  loading = false,
  children,
  ...props
}: React.ComponentProps<'button'> &
  VariantProps<typeof buttonVariants> & {
    asChild?: boolean
    loading?: boolean
  }) {
  const Comp = asChild ? Slot : 'button'
  return (
    <Comp
      data-slot="button"
      className={cn(buttonVariants({ variant, size, className }))}
      disabled={loading}
      {...props}
    >
      {loading && (
        <span className="size-4 shrink-0 animate-spin rounded-full border-2 border-current border-t-transparent" />
      )}
      {children}
    </Comp>
  )
}

export { Button, buttonVariants }
