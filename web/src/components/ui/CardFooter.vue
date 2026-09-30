<script setup lang="ts">
import { useClass } from '@/lib/utils'

const classes = useClass('card-footer')
</script>

<template>
  <div :class="classes">
    <slot />
  </div>
</template>

<style scoped>
.card-footer {
  display: flex;
  align-items: center;
  padding: 0 20px;
}
</style>