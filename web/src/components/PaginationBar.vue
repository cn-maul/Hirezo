<script setup lang="ts">
import { computed } from 'vue'
import Button from '@/components/ui/Button.vue'

const props = withDefaults(
  defineProps<{
    page: number
    pageSize: number
    total: number
  }>(),
  { page: 1, pageSize: 20, total: 0 },
)

const emit = defineEmits<{ change: [page: number] }>()

const totalPages = computed(() => Math.max(1, Math.ceil(props.total / props.pageSize)))
const disabled = computed(() => props.total === 0)
</script>

<template>
  <div class="pagination">
    <span class="pagination__total">共 {{ total }} 条</span>
    <div class="pagination__controls">
      <Button
        variant="outline"
        size="sm"
        :disabled="disabled || page <= 1"
        @click="emit('change', page - 1)"
      >
        上一页
      </Button>
      <span class="pagination__page">{{ page }} / {{ totalPages }}</span>
      <Button
        variant="outline"
        size="sm"
        :disabled="disabled || page >= totalPages"
        @click="emit('change', page + 1)"
      >
        下一页
      </Button>
    </div>
  </div>
</template>

<style scoped>
.pagination {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding-top: 16px;
  font-size: 13px;
  color: var(--text-2);
}
.pagination__total {
  font-variant-numeric: tabular-nums;
}
.pagination__controls {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}
.pagination__page {
  font-variant-numeric: tabular-nums;
}
</style>