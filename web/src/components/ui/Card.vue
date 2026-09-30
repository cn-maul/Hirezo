<script setup lang="ts">
import { useClass } from '@/lib/utils'

const classes = useClass('card')
</script>

<template>
  <div :class="classes">
    <slot />
  </div>
</template>

<style scoped>
.card {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
  border-radius: var(--r-card);
  padding: 20px 0;
  background: var(--surface);
  box-shadow: var(--sh-card);
  outline: 1px solid var(--hairline);
  outline-offset: -1px;
}
</style>