<script setup lang="ts">
import { useClass } from '@/lib/utils'

const classes = useClass('skeleton')
</script>

<template>
  <div :class="classes" />
</template>