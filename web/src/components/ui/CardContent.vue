<script setup lang="ts">
import { useClass } from '@/lib/utils'

const classes = useClass('card-content')
</script>

<template>
  <div :class="classes">
    <slot />
  </div>
</template>

<style scoped>
.card-content {
  padding: 0 20px;
}
</style>