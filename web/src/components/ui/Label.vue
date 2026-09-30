<script setup lang="ts">
import { useClass } from '@/lib/utils'

withDefaults(defineProps<{ htmlFor?: string }>(), { htmlFor: '' })

const classes = useClass('label')
</script>

<template>
  <label :for="htmlFor" :class="classes">
    <slot />
  </label>
</template>

<style scoped>
.label {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 13px;
  line-height: 1;
  font-weight: 500;
  color: var(--text-2);
  user-select: none;
}
</style>