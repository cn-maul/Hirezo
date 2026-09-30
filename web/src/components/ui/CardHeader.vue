<script setup lang="ts">
import { useClass } from '@/lib/utils'

const classes = useClass('card-header')
</script>

<template>
  <div :class="classes">
    <slot />
  </div>
</template>

<style scoped>
.card-header {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  padding: 0 20px;
}
</style>