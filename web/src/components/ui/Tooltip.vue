<script setup lang="ts">
import { ref, onBeforeUnmount } from 'vue'

withDefaults(
  defineProps<{
    content?: string
    side?: 'top' | 'bottom' | 'left' | 'right'
  }>(),
  { content: '', side: 'top' },
)

const visible = ref(false)
let timer: ReturnType<typeof setTimeout> | null = null

function show() {
  if (timer) clearTimeout(timer)
  timer = setTimeout(() => {
    visible.value = true
  }, 200)
}

function hide() {
  if (timer) clearTimeout(timer)
  visible.value = false
}

onBeforeUnmount(() => {
  if (timer) clearTimeout(timer)
})
</script>

<template>
  <span
    class="tooltip-wrap"
    @mouseenter="show"
    @mouseleave="hide"
    @focusin="show"
    @focusout="hide"
  >
    <slot />
    <Transition name="tip">
      <span
        v-if="visible && content"
        class="tooltip"
        :class="`tooltip--${side}`"
        role="tooltip"
      >
        {{ content }}
      </span>
    </Transition>
  </span>
</template>

<style scoped>
.tooltip-wrap {
  position: relative;
  display: inline-flex;
}

.tooltip {
  position: absolute;
  z-index: 50;
  width: max-content;
  max-width: 240px;
  border-radius: var(--r-chip);
  background: rgba(29, 29, 31, 0.92);
  padding: 6px 10px;
  font-size: 12px;
  font-weight: 500;
  letter-spacing: -0.005em;
  color: #fff;
  text-align: center;
  box-shadow: var(--sh-card);
  backdrop-filter: blur(8px);
  -webkit-backdrop-filter: blur(8px);
  pointer-events: none;
}
.dark .tooltip {
  background: rgba(245, 245, 247, 0.92);
  color: #1d1d1f;
}

.tooltip--top {
  bottom: calc(100% + 6px);
  left: 50%;
  transform: translateX(-50%);
}
.tooltip--bottom {
  top: calc(100% + 6px);
  left: 50%;
  transform: translateX(-50%);
}
.tooltip--left {
  right: calc(100% + 6px);
  top: 50%;
  transform: translateY(-50%);
}
.tooltip--right {
  left: calc(100% + 6px);
  top: 50%;
  transform: translateY(-50%);
}

.tip-enter-active,
.tip-leave-active {
  transition:
    opacity 200ms var(--ease-out-quart),
    transform 200ms var(--ease-out-quart);
}
.tip-enter-from,
.tip-leave-to {
  opacity: 0;
  transform: translateX(-50%) scale(0.96);
}
</style>