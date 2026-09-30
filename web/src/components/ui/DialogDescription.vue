<script setup lang="ts">
import { useClass } from '@/lib/utils'

const classes = useClass('dialog-desc')
</script>

<template>
  <p :class="classes">
    <slot />
  </p>
</template>

<style scoped>
.dialog-desc {
  font-size: 13px;
  color: var(--text-2);
}
</style>