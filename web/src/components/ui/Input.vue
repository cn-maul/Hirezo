<script setup lang="ts">
import { useClass } from '@/lib/utils'

const props = withDefaults(
  defineProps<{
    modelValue?: string | number
    type?: string
    placeholder?: string
    id?: string
    disabled?: boolean
    invalid?: boolean
    maxLength?: number
    min?: number
    max?: number
    autoFocus?: boolean
    autoComplete?: string
    spellCheck?: boolean
  }>(),
  {
    modelValue: '',
    type: 'text',
    placeholder: '',
    id: '',
    disabled: false,
    invalid: false,
    maxLength: undefined,
    min: undefined,
    max: undefined,
    autoFocus: false,
    autoComplete: undefined,
    spellCheck: undefined,
  },
)

const emit = defineEmits<{ 'update:modelValue': [value: string] }>()

const classes = useClass('input', props.invalid && 'input--invalid')

function onInput(e: Event) {
  emit('update:modelValue', (e.target as HTMLInputElement).value)
}
</script>

<template>
  <input
    :id="id"
    :type="type"
    :placeholder="placeholder"
    :disabled="disabled"
    :maxlength="maxLength"
    :min="min"
    :max="max"
    :autofocus="autoFocus"
    :autocomplete="autoComplete"
    :spellcheck="spellCheck"
    :value="modelValue"
    :class="classes"
    :aria-invalid="invalid || undefined"
    @input="onInput"
  />
</template>

<style scoped>
.input {
  display: flex;
  height: 40px;
  width: 100%;
  min-width: 0;
  border-radius: var(--r-thumb);
  border: 1px solid var(--hairline);
  background: var(--surface);
  padding: 0 12px;
  font-size: 14px;
  color: var(--text);
  outline: none;
  transition:
    border-color 200ms var(--ease-out-quart),
    box-shadow 200ms var(--ease-out-quart),
    background-color 200ms var(--ease-out-quart);
}
.input::placeholder {
  color: var(--text-4);
}
.input::selection {
  background: rgba(0, 113, 227, 0.18);
}
.input:hover:not(:focus):not(:disabled) {
  background: var(--hover);
}
.input:focus {
  border-color: var(--accent);
  background: var(--surface);
  box-shadow: 0 0 0 4px rgba(0, 113, 227, 0.18);
}
.dark .input:focus {
  box-shadow: 0 0 0 4px rgba(10, 132, 255, 0.28);
}
.input--invalid {
  border-color: var(--danger);
  box-shadow: 0 0 0 4px rgba(215, 0, 21, 0.14);
}
.input:disabled {
  pointer-events: none;
  cursor: not-allowed;
  opacity: 0.4;
}
.input[type='file'] {
  display: inline-flex;
  height: 28px;
  border: 0;
  background: transparent;
  font-size: 13px;
  font-weight: 500;
  color: var(--text);
}
</style>