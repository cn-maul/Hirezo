<script setup lang="ts">
import { useClass } from '@/lib/utils'

const classes = useClass('card-desc')
</script>

<template>
  <div :class="classes">
    <slot />
  </div>
</template>

<style scoped>
.card-desc {
  font-size: 13px;
  color: var(--text-2);
}
</style>