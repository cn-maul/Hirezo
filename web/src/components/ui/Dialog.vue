<script setup lang="ts">
import { computed } from 'vue'
import X from 'lucide-vue-next/dist/esm/icons/x.js'
import { useClass } from '@/lib/utils'

const props = withDefaults(
  defineProps<{
    open?: boolean
  }>(),
  { open: false },
)

const emit = defineEmits<{ 'update:open': [value: boolean] }>()

const isOpen = computed({
  get: () => props.open,
  set: (v: boolean) => emit('update:open', v),
})

const classes = useClass('dialog-content')

function close() {
  isOpen.value = false
}

function onOverlayClick(e: MouseEvent) {
  if (e.target === e.currentTarget) close()
}
</script>

<template>
  <Teleport to="body">
    <Transition name="overlay">
      <div
        v-if="open"
        class="dialog-overlay"
        @click="onOverlayClick"
      >
        <Transition name="dialog" appear>
          <div
            :class="classes"
            role="dialog"
            aria-modal="true"
          >
            <slot />
            <button
              type="button"
              class="dialog-close"
              aria-label="关闭"
              @click="close"
            >
              <X :size="18" />
            </button>
          </div>
        </Transition>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.dialog-overlay {
  position: fixed;
  inset: 0;
  z-index: 50;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1rem;
  background: rgba(0, 0, 0, 0.25);
  backdrop-filter: blur(2px);
  -webkit-backdrop-filter: blur(2px);
}
.dark .dialog-overlay {
  background: rgba(0, 0, 0, 0.55);
}

.dialog-content {
  position: relative;
  width: 100%;
  max-width: 32rem;
  max-height: 88vh;
  overflow-y: auto;
  border-radius: var(--r-hero);
  background: var(--surface);
  color: var(--text);
  padding: 24px;
  box-shadow: var(--sh-overlay);
}

.dialog-close {
  position: absolute;
  top: 16px;
  right: 16px;
  display: grid;
  width: 32px;
  height: 32px;
  place-items: center;
  border-radius: 999px;
  color: var(--text-2);
  transition:
    background-color 200ms var(--ease-out-quart),
    color 200ms var(--ease-out-quart),
    transform 200ms var(--ease-out-quart);
}
.dialog-close:hover {
  background: var(--hover);
  color: var(--text);
}
.dialog-close:active {
  transform: scale(0.9);
}
.dialog-close :deep(svg) {
  width: 18px;
  height: 18px;
  pointer-events: none;
}
</style>