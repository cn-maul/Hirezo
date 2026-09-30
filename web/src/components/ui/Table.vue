<script setup lang="ts">
import { useClass } from '@/lib/utils'

const classes = useClass('table')
</script>

<template>
  <div class="table-container">
    <table :class="classes">
      <slot />
    </table>
  </div>
</template>

<style scoped>
.table-container {
  position: relative;
  width: 100%;
  overflow-x: auto;
}
.table {
  width: 100%;
  border-collapse: collapse;
  font-size: 14px;
  caption-side: bottom;
}
</style>