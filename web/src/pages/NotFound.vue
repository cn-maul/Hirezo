<script setup lang="ts">
import { useRouter } from 'vue-router'
import FileQuestion from 'lucide-vue-next/dist/esm/icons/file-question.js'
import Button from '@/components/ui/Button.vue'

const router = useRouter()
</script>

<template>
  <div class="not-found">
    <FileQuestion :size="44" :stroke-width="1.5" class="not-found__icon" />
    <div>
      <h1 class="not-found__title">页面不存在</h1>
      <p class="not-found__desc">您访问的页面不存在或已被移除</p>
    </div>
    <Button @click="router.push('/')">返回首页</Button>
  </div>
</template>

<style scoped>
.not-found {
  display: flex;
  min-height: 100vh;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 1.25rem;
  background: var(--bg);
  padding: 32px;
  text-align: center;
}
.not-found__icon {
  color: var(--faint);
}
.not-found__title {
  font-size: 21px;
  font-weight: 600;
  letter-spacing: -0.02em;
}
.not-found__desc {
  margin-top: 6px;
  font-size: 14px;
  color: var(--text-2);
}
</style>