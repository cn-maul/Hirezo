<script setup lang="ts">
import ArrowLeft from 'lucide-vue-next/dist/esm/icons/arrow-left.js'
import Button from '@/components/ui/Button.vue'
import { useClass } from '@/lib/utils'

withDefaults(
  defineProps<{
    title: string
    description?: string
    onBack?: () => void
  }>(),
  { description: '', onBack: undefined },
)

const classes = useClass('page-header')
</script>

<template>
  <div :class="classes">
    <div class="page-header__main">
      <div class="page-header__title-row">
        <Button
          v-if="onBack"
          variant="ghost"
          size="icon"
          aria-label="返回"
          @click="onBack"
        >
          <ArrowLeft :size="18" />
        </Button>
        <h1 class="page-header__title">{{ title }}</h1>
      </div>
      <p v-if="description" class="page-header__desc" :class="{ 'page-header__desc--back': onBack }">
        {{ description }}
      </p>
    </div>
    <div v-if="$slots.action" class="page-header__action">
      <slot name="action" />
    </div>
  </div>
</template>

<style scoped>
.page-header {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  justify-content: space-between;
  gap: 0.75rem;
}

.page-header__main {
  min-width: 0;
}

.page-header__title-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.5rem;
}

.page-header__title {
  font-size: 26px;
  line-height: 1.2;
  font-weight: 600;
  letter-spacing: -0.025em;
  color: var(--text);
}

.page-header__desc {
  margin-top: 6px;
  font-size: 14px;
  color: var(--text-2);
}
.page-header__desc--back {
  padding-left: 48px;
}

.page-header__action {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.5rem;
}
</style>