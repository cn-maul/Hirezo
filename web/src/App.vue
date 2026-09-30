<script setup lang="ts">
import { watch } from 'vue'
import { useQuery } from '@/composables/useQuery'
import { getSettings } from '@/api/settings'
import ToastHost from '@/components/ToastHost.vue'

// SiteTitle 全局同步浏览器标签页标题：跟随「设置-站点名称」，
// 覆盖登录页与所有管理页，改名后即时生效（设置页已失效该查询）。
const { data: settings } = useQuery(['settings'], getSettings)

watch(
  () => settings.value,
  (s) => {
    document.title = s?.site_name || 'Hirezo 教师管理'
  },
  { immediate: true },
)
</script>

<template>
  <RouterView />
  <ToastHost />
</template>