<script setup lang="ts">
// 字段下方只放校验错误；识别依据与置信度不再逐项展示（核对时看左侧原件即可）。
defineProps<{ error?: string }>()
</script>

<template>
  <p v-if="error" class="teacher-error">{{ error }}</p>
</template>

<style scoped>
.teacher-error {
  font-size: 13px;
  color: var(--danger);
}
</style>
