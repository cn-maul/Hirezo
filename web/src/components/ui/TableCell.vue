<script setup lang="ts">
import { useClass } from '@/lib/utils'

withDefaults(
  defineProps<{
    colSpan?: number
  }>(),
  { colSpan: undefined },
)

const classes = useClass('table-cell')
</script>

<template>
  <td :class="classes" :colspan="colSpan">
    <slot />
  </td>
</template>

<style scoped>
.table-cell {
  padding: 10px 12px;
  vertical-align: middle;
  white-space: nowrap;
}
</style>