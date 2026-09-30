<script setup lang="ts">
import { useClass } from '@/lib/utils'

const classes = useClass('table-header')
</script>

<template>
  <thead :class="classes">
    <slot />
  </thead>
</template>

<style scoped>
.table-header {
  border-bottom: 1px solid var(--hairline);
}
</style>