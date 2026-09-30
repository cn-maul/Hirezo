<script setup lang="ts">
import { inject, computed } from 'vue'
import { Check } from 'lucide-vue-next'
import { useClass } from '@/lib/utils'

// 由 Select.vue 提供
interface SelectContext {
  modelValue: string
  selectValue: (v: string) => void
}

const ctx = inject<SelectContext>('select-context')

const props = withDefaults(
  defineProps<{
    value: string
    disabled?: boolean
  }>(),
  { disabled: false },
)

const selected = computed(() => ctx?.modelValue === props.value)

const classes = useClass('select-item', selected.value && 'select-item--selected')

function onClick() {
  if (props.disabled) return
  ctx?.selectValue(props.value)
}
</script>

<template>
  <div
    role="option"
    :aria-selected="selected"
    :class="classes"
    @click="onClick"
  >
    <span class="select-item__label">
      <slot />
    </span>
    <span v-if="selected" class="select-item__check">
      <Check :size="18" />
    </span>
  </div>
</template>

<style scoped>
.select-item {
  position: relative;
  display: flex;
  width: 100%;
  align-items: center;
  gap: 0.5rem;
  border-radius: var(--r-chip);
  padding: 8px 32px 8px 10px;
  font-size: 14px;
  color: var(--text);
  cursor: pointer;
  user-select: none;
  transition:
    background-color 150ms var(--ease-out-quart),
    color 150ms var(--ease-out-quart);
}
.select-item:hover {
  background: var(--hover);
}
.select-item--selected {
  color: var(--text);
}

.select-item__label {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.select-item__check {
  position: absolute;
  right: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--accent);
}
.select-item__check :deep(svg) {
  width: 18px;
  height: 18px;
}
</style>