<script setup lang="ts">
import { useClass } from '@/lib/utils'

const props = withDefaults(
  defineProps<{
    modelValue?: string
    placeholder?: string
    id?: string
    rows?: number
    disabled?: boolean
    invalid?: boolean
  }>(),
  {
    modelValue: '',
    placeholder: '',
    id: '',
    rows: 3,
    disabled: false,
    invalid: false,
  },
)

const emit = defineEmits<{ 'update:modelValue': [value: string] }>()

const classes = useClass('textarea', props.invalid && 'textarea--invalid')

function onInput(e: Event) {
  emit('update:modelValue', (e.target as HTMLTextAreaElement).value)
}
</script>

<template>
  <textarea
    :id="id"
    :placeholder="placeholder"
    :rows="rows"
    :disabled="disabled"
    :value="modelValue"
    :class="classes"
    :aria-invalid="invalid || undefined"
    @input="onInput"
  />
</template>

<style scoped>
.textarea {
  display: flex;
  width: 100%;
  min-height: 80px;
  border-radius: var(--r-thumb);
  border: 1px solid var(--hairline);
  background: var(--surface);
  padding: 10px 12px;
  font-size: 14px;
  line-height: 1.6;
  color: var(--text);
  outline: none;
  resize: none;
  transition:
    border-color 200ms var(--ease-out-quart),
    box-shadow 200ms var(--ease-out-quart),
    background-color 200ms var(--ease-out-quart);
}
.textarea::placeholder {
  color: var(--text-4);
}
.textarea:hover:not(:focus):not(:disabled) {
  background: var(--hover);
}
.textarea:focus {
  border-color: var(--accent);
  background: var(--surface);
  box-shadow: 0 0 0 4px rgba(0, 113, 227, 0.18);
}
.dark .textarea:focus {
  box-shadow: 0 0 0 4px rgba(10, 132, 255, 0.28);
}
.textarea--invalid {
  border-color: var(--danger);
  box-shadow: 0 0 0 4px rgba(215, 0, 21, 0.14);
}
.textarea:disabled {
  cursor: not-allowed;
  opacity: 0.4;
}
</style>