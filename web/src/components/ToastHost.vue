<script setup lang="ts">
import { CheckCircle2, AlertCircle, Info } from 'lucide-vue-next'
import { useToastState, dismiss } from '@/composables/useToast'

const toasts = useToastState()

const icons = {
  success: CheckCircle2,
  error: AlertCircle,
  info: Info,
}
</script>

<template>
  <Teleport to="body">
    <div class="toast-host" aria-live="polite">
      <TransitionGroup name="toast">
        <div
          v-for="t in toasts"
          :key="t.id"
          class="toast"
          :class="`toast--${t.type}`"
          role="status"
          @click="dismiss(t.id)"
        >
          <component :is="icons[t.type]" :size="18" class="toast__icon" />
          <span class="toast__msg">{{ t.message }}</span>
        </div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>

<style scoped>
.toast-host {
  position: fixed;
  top: 16px;
  left: 50%;
  transform: translateX(-50%);
  z-index: 100;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  pointer-events: none;
}

.toast {
  display: flex;
  align-items: center;
  gap: 8px;
  max-width: 420px;
  border-radius: var(--r-thumb);
  background: var(--surface);
  color: var(--text);
  padding: 10px 16px;
  font-size: 14px;
  box-shadow: var(--sh-overlay);
  outline: 1px solid var(--hairline);
  outline-offset: -1px;
  cursor: pointer;
  pointer-events: auto;
}

.toast__icon {
  flex-shrink: 0;
}
.toast--success .toast__icon {
  color: var(--success);
}
.toast--error .toast__icon {
  color: var(--danger);
}
.toast--info .toast__icon {
  color: var(--accent);
}

.toast__msg {
  line-height: 1.4;
}
</style>