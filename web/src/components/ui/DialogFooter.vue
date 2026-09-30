<script setup lang="ts">
import { useClass } from '@/lib/utils'

const classes = useClass('dialog-footer')
</script>

<template>
  <div :class="classes">
    <slot />
  </div>
</template>

<style scoped>
.dialog-footer {
  display: flex;
  flex-direction: column-reverse;
  gap: 0.5rem;
}
@media (min-width: 640px) {
  .dialog-footer {
    flex-direction: row;
    justify-content: flex-end;
  }
}
</style>