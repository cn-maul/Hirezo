<script setup lang="ts">
withDefaults(defineProps<{ label?: string }>(), { label: '加载中…' })
</script>

<template>
  <div class="page-spinner" role="status" :aria-label="label">
    <span class="spinner page-spinner__ring" />
    <span class="page-spinner__label">{{ label }}</span>
  </div>
</template>

<style scoped>
.page-spinner {
  display: flex;
  height: 256px;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.75rem;
}

.page-spinner__ring {
  width: 28px;
  height: 28px;
  border-width: 2px;
  border-color: var(--track);
  border-top-color: var(--accent);
}

.page-spinner__label {
  font-size: 13px;
  color: var(--text-2);
}
</style>