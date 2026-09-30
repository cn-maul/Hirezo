<script setup lang="ts">
import { useClass } from '@/lib/utils'

const props = withDefaults(
  defineProps<{
    variant?: 'default' | 'destructive' | 'outline' | 'secondary' | 'ghost' | 'link'
    size?: 'default' | 'sm' | 'lg' | 'icon'
    loading?: boolean
    type?: 'button' | 'submit' | 'reset'
    disabled?: boolean
  }>(),
  {
    variant: 'default',
    size: 'default',
    loading: false,
    type: 'button',
    disabled: false,
  },
)

const emit = defineEmits<{ click: [event: MouseEvent] }>()

const classes = useClass(
  'btn',
  `btn--${props.variant}`,
  `btn--${props.size}`,
  props.loading && 'btn--loading',
)

function onClick(e: MouseEvent) {
  if (props.disabled || props.loading) return
  emit('click', e)
}
</script>

<template>
  <button
    :type="type"
    :class="classes"
    :disabled="disabled || loading"
    @click="onClick"
  >
    <span v-if="loading" class="btn__spinner" aria-hidden="true" />
    <slot />
  </button>
</template>

<style scoped>
.btn {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  flex-shrink: 0;
  white-space: nowrap;
  border-radius: var(--r-pill);
  font-size: 14px;
  font-weight: 500;
  letter-spacing: -0.008em;
  outline: none;
  user-select: none;
  cursor: pointer;
  transition:
    transform 200ms var(--ease-out-quart),
    background-color 150ms var(--ease-smooth),
    box-shadow 150ms var(--ease-smooth),
    color 150ms var(--ease-smooth),
    border-color 150ms var(--ease-smooth);
  -webkit-tap-highlight-color: transparent;
}
.btn:active:not(:disabled) {
  transform: scale(0.97);
}
.btn:disabled {
  pointer-events: none;
  opacity: 0.4;
}
.btn:focus-visible {
  box-shadow: 0 0 0 4px rgba(0, 113, 227, 0.25);
}
.dark .btn:focus-visible {
  box-shadow: 0 0 0 4px rgba(10, 132, 255, 0.35);
}

/* Sizes */
.btn--default {
  height: 40px;
  padding: 0 16px;
}
.btn--sm {
  height: 36px;
  gap: 0.375rem;
  padding: 0 14px;
}
.btn--lg {
  height: 44px;
  padding: 0 24px;
}
.btn--icon {
  width: 40px;
  height: 40px;
  padding: 0;
}

/* Variants */
.btn--default {
  background: var(--accent);
  color: #fff;
  box-shadow: var(--sh-card);
}
.btn--default:hover:not(:disabled) {
  filter: brightness(1.05);
}
.btn--destructive {
  background: var(--danger);
  color: #fff;
  box-shadow: var(--sh-card);
}
.btn--destructive:hover:not(:disabled) {
  filter: brightness(1.05);
}
.btn--outline {
  border: 1px solid var(--hairline);
  background: var(--surface);
  color: var(--text);
}
.btn--outline:hover:not(:disabled) {
  background: var(--hover);
}
.btn--secondary {
  background: var(--track);
  color: var(--text);
}
.btn--secondary:hover:not(:disabled) {
  filter: brightness(0.97);
}
.btn--ghost {
  color: var(--text-2);
}
.btn--ghost:hover:not(:disabled) {
  background: var(--hover);
  color: var(--text);
}
.btn--link {
  color: var(--accent);
  text-underline-offset: 4px;
}
.btn--link:hover:not(:disabled) {
  text-decoration: underline;
}

/* Loading spinner */
.btn__spinner {
  width: 16px;
  height: 16px;
  flex-shrink: 0;
  border-radius: 999px;
  border: 2px solid currentColor;
  border-top-color: transparent;
  animation: btn-spin 0.7s linear infinite;
}
@keyframes btn-spin {
  to { transform: rotate(360deg); }
}

/* Icon sizing */
.btn :deep(svg) {
  width: 18px;
  height: 18px;
  flex-shrink: 0;
  pointer-events: none;
}
</style>