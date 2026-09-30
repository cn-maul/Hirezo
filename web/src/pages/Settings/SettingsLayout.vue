<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import Settings2 from 'lucide-vue-next/dist/esm/icons/settings-2.js'
import Tags from 'lucide-vue-next/dist/esm/icons/tags.js'
import Sparkles from 'lucide-vue-next/dist/esm/icons/sparkles.js'
import { cn } from '@/lib/utils'
import { APP_VERSION_LABEL } from '@/lib/version'
import { useAuth } from '@/composables/useAuth'

const route = useRoute()
const router = useRouter()
const { authed, user } = useAuth()

const tabs = [
  { to: '/settings/general', icon: Settings2, label: '通用' },
  { to: '/settings/dicts', icon: Tags, label: '字典管理' },
  { to: '/settings/llm', icon: Sparkles, label: '模型配置' },
]

const showSpinner = computed(() => authed.value === null)

// 非管理员跳回首页
if (authed.value !== null && (authed.value === false || user.value?.role !== 'admin')) {
  void router.replace('/')
}
</script>

<template>
  <div v-if="showSpinner" class="settings-layout__spinner">
    <span class="spinner settings-layout__ring" />
  </div>

  <div v-else class="settings-layout">
    <div class="settings-layout__header">
      <h1 class="settings-layout__title">设置</h1>
      <span class="settings-layout__version">{{ APP_VERSION_LABEL }}</span>
    </div>

    <!-- Apple 分段控件：横向滚动只在窄屏发生，灰轨 + 滑动白 pill -->
    <div class="settings-layout__tabs-scroll">
      <div class="settings-layout__tabs">
        <RouterLink
          v-for="t in tabs"
          :key="t.to"
          :to="t.to"
          :class="cn(
            'settings-layout__tab',
            route.path === t.to ? 'settings-layout__tab--active' : '',
          )"
        >
          <component :is="t.icon" :size="18" :stroke-width="1.75" />
          {{ t.label }}
        </RouterLink>
      </div>
    </div>

    <RouterView />
  </div>
</template>

<style scoped>
.settings-layout {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}
.settings-layout__spinner {
  display: flex;
  height: 256px;
  align-items: center;
  justify-content: center;
}
.settings-layout__ring {
  width: 28px;
  height: 28px;
  border-width: 2px;
  border-color: var(--track);
  border-top-color: var(--accent);
}

.settings-layout__header {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-end;
  justify-content: space-between;
  gap: 0.5rem;
}
.settings-layout__title {
  font-size: 26px;
  line-height: 1.2;
  font-weight: 600;
  letter-spacing: -0.025em;
}
.settings-layout__version {
  border-radius: 999px;
  background: var(--track);
  padding: 4px 12px;
  font-family: var(--mono);
  font-size: 12px;
  color: var(--text-2);
}

.settings-layout__tabs-scroll {
  max-width: 100%;
  overflow-x: auto;
}
.settings-layout__tabs {
  display: flex;
  width: max-content;
  align-items: center;
  gap: 2px;
  border-radius: var(--r-pill);
  background: var(--track);
  padding: 4px;
}
.settings-layout__tab {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  border-radius: var(--r-pill);
  padding: 8px 16px;
  font-size: 14px;
  font-weight: 500;
  white-space: nowrap;
  color: var(--text-2);
  transition:
    background-color 200ms var(--ease-out-quart),
    color 200ms var(--ease-out-quart),
    box-shadow 200ms var(--ease-out-quart);
}
.settings-layout__tab:hover {
  color: var(--text);
}
.settings-layout__tab--active {
  background: var(--surface);
  color: var(--text);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08), 0 2px 8px rgba(0, 0, 0, 0.06);
}
</style>