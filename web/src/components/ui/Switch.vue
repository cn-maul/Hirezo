<script setup lang="ts">
import { useClass } from '@/lib/utils'

const props = withDefaults(
  defineProps<{
    modelValue?: boolean
    id?: string
    disabled?: boolean
  }>(),
  {
    modelValue: false,
    id: '',
    disabled: false,
  },
)

const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()

const classes = useClass('switch')

function onClick() {
  if (props.disabled) return
  emit('update:modelValue', !props.modelValue)
}
</script>

<template>
  <button
    :id="id"
    type="button"
    role="switch"
    :aria-checked="modelValue"
    :class="classes"
    :disabled="disabled"
    @click="onClick"
  >
    <span class="switch__thumb" />
  </button>
</template>

<style scoped>
.switch {
  position: relative;
  display: inline-flex;
  width: 51px;
  height: 31px;
  flex-shrink: 0;
  align-items: center;
  border-radius: 999px;
  background: var(--track);
  outline: none;
  cursor: pointer;
  transition: background-color 300ms var(--ease-out-quart);
  -webkit-tap-highlight-color: transparent;
}
.switch::before {
  content: '';
  position: absolute;
  inset: -6.5px 0;
}
.switch[aria-checked='true'] {
  background: var(--live);
}
.switch:disabled {
  cursor: not-allowed;
  opacity: 0.4;
}
.switch:focus-visible {
  box-shadow: 0 0 0 4px rgba(0, 113, 227, 0.25);
}
.dark .switch:focus-visible {
  box-shadow: 0 0 0 4px rgba(10, 132, 255, 0.35);
}

.switch__thumb {
  display: block;
  width: 27px;
  height: 27px;
  border-radius: 999px;
  background: #fff;
  box-shadow:
    0 2px 6px rgba(0, 0, 0, 0.18),
    0 0 0 0.5px rgba(0, 0, 0, 0.04);
  transition:
    transform 300ms var(--ease-spring),
    background-color 200ms var(--ease-smooth);
}
.switch[aria-checked='true'] .switch__thumb {
  transform: translateX(22px);
}
.switch[aria-checked='false'] .switch__thumb {
  transform: translateX(2px);
}
</style>