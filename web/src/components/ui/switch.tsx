import * as React from 'react'
import * as SwitchPrimitive from '@radix-ui/react-switch'
import { cn } from '@/lib/utils'

function Switch({
  className,
  ...props
}: React.ComponentProps<typeof SwitchPrimitive.Root>) {
  return (
    <SwitchPrimitive.Root
      data-slot="switch"
      className={cn(
        'peer relative inline-flex h-[31px] w-[51px] shrink-0 items-center rounded-full transition-colors duration-300 ease-[var(--ease-out-quart)] outline-none',
        'data-[state=checked]:bg-[var(--live)] data-[state=unchecked]:bg-[var(--track)]',
        'disabled:cursor-not-allowed disabled:opacity-40',
        // 44px hit area without a visual 44px box
        'before:absolute before:-inset-y-[6.5px] before:-inset-x-0 before:content-[""]',
        className,
      )}
      {...props}
    >
      <SwitchPrimitive.Thumb
        data-slot="switch-thumb"
        className={cn(
          'pointer-events-none block size-[27px] rounded-full bg-white shadow-[0_2px_6px_rgba(0,0,0,0.18),0_0_0_0.5px_rgba(0,0,0,0.04)] transition-transform duration-300 ease-[var(--ease-spring)]',
          'data-[state=checked]:translate-x-[22px] data-[state=unchecked]:translate-x-[2px]',
        )}
      />
    </SwitchPrimitive.Root>
  )
}

export { Switch }
