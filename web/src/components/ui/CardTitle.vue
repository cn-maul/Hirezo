<script setup lang="ts">
import { useClass } from '@/lib/utils'

const classes = useClass('card-title')
</script>

<template>
  <div :class="classes">
    <slot />
  </div>
</template>

<style scoped>
.card-title {
  font-size: 15px;
  line-height: 1.2;
  font-weight: 600;
  letter-spacing: -0.015em;
  color: var(--text);
}
</style>