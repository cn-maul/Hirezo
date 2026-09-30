<script setup lang="ts">
import { onBeforeUnmount, onMounted, provide, ref } from 'vue'
import ChevronDown from 'lucide-vue-next/dist/esm/icons/chevron-down.js'
import { useClass } from '@/lib/utils'

const props = withDefaults(
  defineProps<{
    modelValue?: string
    placeholder?: string
    id?: string
    size?: 'sm' | 'default'
    disabled?: boolean
  }>(),
  {
    modelValue: '',
    placeholder: '',
    id: '',
    size: 'default',
    disabled: false,
  },
)

const emit = defineEmits<{ 'update:modelValue': [value: string] }>()

const open = ref(false)
const triggerRef = ref<HTMLButtonElement | null>(null)
const contentRef = ref<HTMLDivElement | null>(null)

const classes = useClass('select-trigger', `select-trigger--${props.size}`)

function toggle() {
  if (props.disabled) return
  open.value = !open.value
}

function selectValue(v: string) {
  emit('update:modelValue', v)
  open.value = false
}

provide('select-context', {
  modelValue: props.modelValue,
  selectValue,
})

function onDocumentClick(e: MouseEvent) {
  const target = e.target as Node
  if (
    open.value &&
    triggerRef.value &&
    !triggerRef.value.contains(target) &&
    contentRef.value &&
    !contentRef.value.contains(target)
  ) {
    open.value = false
  }
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') open.value = false
}

onMounted(() => {
  document.addEventListener('click', onDocumentClick)
  document.addEventListener('keydown', onKeydown)
})
onBeforeUnmount(() => {
  document.removeEventListener('click', onDocumentClick)
  document.removeEventListener('keydown', onKeydown)
})
</script>

<template>
  <div class="select">
    <button
      :id="id"
      ref="triggerRef"
      type="button"
      :class="classes"
      :disabled="disabled"
      :aria-expanded="open"
      @click="toggle"
    >
      <span class="select-value" :class="{ 'select-value--placeholder': !modelValue }">
        <slot name="value" :selected="modelValue">
          {{ modelValue || placeholder }}
        </slot>
      </span>
      <ChevronDown :size="18" class="select-chevron" :class="{ 'select-chevron--open': open }" />
    </button>

    <Transition name="pop">
      <div
        v-if="open"
        ref="contentRef"
        class="select-content"
        role="listbox"
      >
        <slot :close="selectValue" />
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.select {
  position: relative;
  display: inline-block;
}

.select-trigger {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  border-radius: var(--r-thumb);
  border: 1px solid var(--hairline);
  background: var(--surface);
  padding: 0 12px;
  font-size: 14px;
  white-space: nowrap;
  color: var(--text);
  outline: none;
  user-select: none;
  cursor: pointer;
  transition:
    border-color 200ms var(--ease-out-quart),
    box-shadow 200ms var(--ease-out-quart),
    background-color 200ms var(--ease-out-quart);
}
.select-trigger--default {
  height: 40px;
}
.select-trigger--sm {
  height: 36px;
}
.select-trigger:hover:not(:disabled) {
  background: var(--hover);
}
.select-trigger:focus-visible {
  border-color: var(--accent);
  box-shadow: 0 0 0 4px rgba(0, 113, 227, 0.18);
}
.select-trigger:disabled {
  cursor: not-allowed;
  opacity: 0.4;
}

.select-value {
  flex: 1;
  text-align: left;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.select-value--placeholder {
  color: var(--text-4);
}

.select-chevron {
  flex-shrink: 0;
  color: var(--text-3);
  opacity: 0.7;
  transition: transform 200ms var(--ease-out-quart);
}
.select-chevron--open {
  transform: rotate(180deg);
}

.select-content {
  position: absolute;
  top: calc(100% + 4px);
  left: 0;
  z-index: 60;
  min-width: 100%;
  max-height: 300px;
  overflow-y: auto;
  border-radius: var(--r-thumb);
  background: var(--surface);
  padding: 4px;
  box-shadow: var(--sh-overlay);
}
</style>