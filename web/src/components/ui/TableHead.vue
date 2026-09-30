<script setup lang="ts">
import { useClass } from '@/lib/utils'

withDefaults(
  defineProps<{
    style?: Record<string, string>
  }>(),
  { style: undefined },
)

const classes = useClass('table-head')
</script>

<template>
  <th :class="classes" :style="style">
    <slot />
  </th>
</template>

<style scoped>
.table-head {
  height: 40px;
  padding: 0 12px;
  text-align: left;
  vertical-align: middle;
  font-size: 12px;
  font-weight: 500;
  letter-spacing: 0.01em;
  white-space: nowrap;
  color: var(--text-2);
}
</style>