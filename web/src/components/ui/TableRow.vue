<script setup lang="ts">
import { useClass } from '@/lib/utils'

const classes = useClass('table-row')
</script>

<template>
  <tr :class="classes">
    <slot />
  </tr>
</template>

<style scoped>
.table-row {
  border-bottom: 1px solid var(--hairline);
  transition:
    background-color 150ms var(--ease-out-quart),
    color 150ms var(--ease-out-quart);
}
.table-row:hover {
  background: var(--hover);
}
</style>