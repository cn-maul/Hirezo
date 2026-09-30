<script setup lang="ts">
import { computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuth } from '@/composables/useAuth'

const { authed } = useAuth()
const route = useRoute()
const router = useRouter()

const showSpinner = computed(() => authed.value === null)

// 未登录时跳转登录页（带 next 参数）。
// 用 watch 监听，避免只在初始化时判断一次导致登录后不生效。
watch(
  authed,
  (v) => {
    if (v === false) {
      const next = encodeURIComponent(route.fullPath)
      void router.replace(`/login?next=${next}`)
    }
  },
  { immediate: true },
)
</script>

<template>
  <div v-if="showSpinner" class="require-auth">
    <span class="spinner require-auth__ring" />
  </div>
  <RouterView v-else />
</template>

<style scoped>
.require-auth {
  display: flex;
  min-height: 100vh;
  align-items: center;
  justify-content: center;
}
.require-auth__ring {
  width: 32px;
  height: 32px;
  border-width: 2px;
  border-color: var(--track);
  border-top-color: var(--accent);
}
</style>