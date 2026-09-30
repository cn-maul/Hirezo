<script setup lang="ts">
import { useClass } from '@/lib/utils'

const classes = useClass('table-body')
</script>

<template>
  <tbody :class="classes">
    <slot />
  </tbody>
</template>

<style scoped>
.table-body :deep(tr:last-child) {
  border-bottom: 0;
}
</style>