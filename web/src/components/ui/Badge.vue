<script setup lang="ts">
import { useClass } from '@/lib/utils'

const props = withDefaults(
  defineProps<{
    variant?: 'default' | 'secondary' | 'destructive' | 'outline' | 'success' | 'warning'
  }>(),
  { variant: 'default' },
)

const classes = useClass('badge', `badge--${props.variant}`)
</script>

<template>
  <span :class="classes">
    <slot />
  </span>
</template>

<style scoped>
.badge {
  display: inline-flex;
  width: fit-content;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  gap: 0.25rem;
  white-space: nowrap;
  border-radius: 999px;
  border: 1px solid transparent;
  padding: 2px 10px;
  font-size: 12px;
  font-weight: 500;
  letter-spacing: -0.005em;
}
.badge :deep(svg) {
  width: 12px;
  height: 12px;
  pointer-events: none;
}
.badge--default {
  background: rgba(0, 113, 227, 0.1);
  color: var(--accent-link);
}
.badge--secondary {
  background: var(--track);
  color: var(--text-2);
}
.badge--destructive {
  background: rgba(215, 0, 21, 0.1);
  color: #d70015;
}
.badge--outline {
  border-color: var(--hairline);
  background: var(--surface);
  color: var(--text-2);
}
.badge--success {
  background: var(--success-soft);
  color: var(--success-soft-fg);
}
.badge--warning {
  background: var(--warning-soft);
  color: var(--warning-soft-fg);
}
</style>