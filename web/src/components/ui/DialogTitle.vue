<script setup lang="ts">
import { useClass } from '@/lib/utils'

const classes = useClass('dialog-title')
</script>

<template>
  <h2 :class="classes">
    <slot />
  </h2>
</template>

<style scoped>
.dialog-title {
  font-size: 19px;
  line-height: 1.2;
  font-weight: 600;
  letter-spacing: -0.02em;
  color: var(--text);
}
</style>