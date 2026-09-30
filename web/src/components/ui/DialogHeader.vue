<script setup lang="ts">
import { useClass } from '@/lib/utils'

const classes = useClass('dialog-header')
</script>

<template>
  <div :class="classes">
    <slot />
  </div>
</template>

<style scoped>
.dialog-header {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
  text-align: center;
}
@media (min-width: 640px) {
  .dialog-header {
    text-align: left;
  }
}
</style>