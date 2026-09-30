<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import Users from 'lucide-vue-next/dist/esm/icons/users.js'
import Settings from 'lucide-vue-next/dist/esm/icons/settings.js'
import Sun from 'lucide-vue-next/dist/esm/icons/sun.js'
import Moon from 'lucide-vue-next/dist/esm/icons/moon.js'
import LogOut from 'lucide-vue-next/dist/esm/icons/log-out.js'
import PanelLeft from 'lucide-vue-next/dist/esm/icons/layout-panel-left.js'
import GraduationCap from 'lucide-vue-next/dist/esm/icons/graduation-cap.js'
import FileSearch from 'lucide-vue-next/dist/esm/icons/file-search.js'
import Button from '@/components/ui/Button.vue'
import Tooltip from '@/components/ui/Tooltip.vue'
import AccountDialog from '@/components/AccountDialog.vue'
import { cn } from '@/lib/utils'
import { useTheme } from '@/composables/useTheme'
import { useAuth } from '@/composables/useAuth'
import { useQuery } from '@/composables/useQuery'
import { getSettings } from '@/api/settings'

const DEFAULT_SITE_NAME = 'Hirezo 教师管理'

const route = useRoute()
const router = useRouter()
const { dark, toggle } = useTheme()
const { logout, user } = useAuth()
const isAdmin = computed(() => user.value?.role === 'admin')

const { data: settings } = useQuery(['settings'], getSettings)
const siteName = computed(() => settings.value?.site_name || DEFAULT_SITE_NAME)

const collapsed = ref(false)
const accountOpen = ref(false)
const scrolled = ref(false)
const scrollRef = ref<HTMLElement | null>(null)

// Below the tablet breakpoint a 224px sidebar eats the viewport, so force it
// to icon-only. The user's own toggle still wins once they're above it.
const narrow = ref(
  typeof window !== 'undefined' && window.matchMedia('(max-width: 900px)').matches,
)

let mq: MediaQueryList | null = null
function onMqChange(e: MediaQueryListEvent) {
  narrow.value = e.matches
}

onMounted(() => {
  mq = window.matchMedia('(max-width: 900px)')
  mq.addEventListener('change', onMqChange)
  const el = scrollRef.value
  if (el) {
    const onScroll = () => {
      scrolled.value = el.scrollTop > 4
    }
    onScroll()
    el.addEventListener('scroll', onScroll, { passive: true })
  }
})

onBeforeUnmount(() => {
  mq?.removeEventListener('change', onMqChange)
})

const railCollapsed = computed(() => collapsed.value || narrow.value)

// Route changes reset the scroll position; keep the edge state honest.
watch(
  () => route.path,
  () => {
    scrollRef.value?.scrollTo({ top: 0 })
    scrolled.value = false
  },
)

const onLogout = async () => {
  await logout()
  void router.replace('/login')
}

const displayName = computed(() => user.value?.display_name || user.value?.username || '管')
const initial = computed(() => displayName.value.charAt(0))

const navItems = computed(() => {
  const items = [
    { to: '/teachers', icon: Users, label: '人员管理', end: true },
    { to: '/resume', icon: FileSearch, label: '简历识别', end: false },
  ]
  if (isAdmin.value) {
    items.push({ to: '/settings', icon: Settings, label: '设置', end: false })
  }
  return items
})
</script>

<template>
  <div class="layout">
    <aside :class="cn('layout__aside', railCollapsed && 'layout__aside--collapsed')">
      <div class="layout__brand">
        <div :class="cn('layout__logo', railCollapsed && 'layout__logo--center')">
          <GraduationCap :size="18" :stroke-width="1.75" />
        </div>
        <span v-if="!railCollapsed" class="layout__site-name">{{ siteName }}</span>
      </div>

      <nav class="layout__nav">
        <RouterLink
          v-for="item in navItems"
          :key="item.to"
          :to="item.to"
          :class="cn(
            'layout__nav-item',
            railCollapsed && 'layout__nav-item--collapsed',
            route.path === item.to || (item.to !== '/teachers' && route.path.startsWith(item.to))
              ? 'layout__nav-item--active'
              : '',
          )"
        >
          <component :is="item.icon" :size="18" :stroke-width="1.75" class="layout__nav-icon" />
          <span v-if="!railCollapsed">{{ item.label }}</span>
        </RouterLink>
      </nav>

      <div class="layout__aside-footer">
        <Button
          variant="ghost"
          size="icon"
          aria-label="收起侧边栏"
          :class="railCollapsed && 'layout__collapse-btn--center'"
          @click="collapsed = !collapsed"
        >
          <PanelLeft :stroke-width="1.75" />
        </Button>
      </div>
    </aside>

    <div class="layout__main">
      <header :class="cn('layout__topbar glass-nav', scrolled && 'layout__topbar--scrolled')">
        <Tooltip :content="dark ? '切换到亮色模式' : '切换到暗色模式'">
          <Button variant="ghost" size="icon" aria-label="切换主题" @click="toggle">
            <Sun v-if="dark" :stroke-width="1.75" />
            <Moon v-else :stroke-width="1.75" />
          </Button>
        </Tooltip>
        <Tooltip content="退出登录">
          <Button variant="ghost" size="icon" aria-label="退出登录" @click="onLogout">
            <LogOut :stroke-width="1.75" />
          </Button>
        </Tooltip>
        <Tooltip :content="displayName">
          <button
            type="button"
            class="layout__avatar"
            aria-label="账号设置"
            @click="accountOpen = true"
          >
            {{ initial }}
          </button>
        </Tooltip>
      </header>

      <main ref="scrollRef" class="layout__content">
        <div class="layout__content-inner">
          <RouterView />
        </div>
      </main>
    </div>

    <AccountDialog v-model:open="accountOpen" />
  </div>
</template>

<style scoped>
.layout {
  display: flex;
  height: 100vh;
  overflow: hidden;
  background: var(--bg);
}

/* ── Sidebar ── */
.layout__aside {
  display: flex;
  flex-shrink: 0;
  flex-direction: column;
  background: var(--bg);
  width: 224px;
  transition: width 300ms var(--ease-out-quart);
}
.layout__aside--collapsed {
  width: 64px;
}

.layout__brand {
  display: flex;
  height: 64px;
  flex-shrink: 0;
  align-items: center;
  gap: 10px;
  padding: 0 12px;
}
.layout__logo {
  display: grid;
  width: 32px;
  height: 32px;
  flex-shrink: 0;
  place-items: center;
  border-radius: var(--r-chip);
  background: var(--accent);
  color: #fff;
}
.layout__logo--center {
  margin: 0 auto;
}
.layout__site-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 15px;
  font-weight: 600;
  letter-spacing: -0.015em;
  color: var(--text);
}

.layout__nav {
  flex: 1;
  overflow-y: auto;
  padding: 0 8px 16px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.layout__nav-item {
  display: flex;
  align-items: center;
  gap: 12px;
  border-radius: var(--r-thumb);
  padding: 8px 12px;
  font-size: 14px;
  font-weight: 500;
  color: var(--text-2);
  transition:
    background-color 200ms var(--ease-out-quart),
    color 200ms var(--ease-out-quart);
}
.layout__nav-item--collapsed {
  justify-content: center;
  padding: 8px;
}
.layout__nav-item:hover {
  background: var(--hover);
  color: var(--text);
}
.layout__nav-item--active {
  background: rgba(0, 113, 227, 0.1);
  color: var(--accent-link);
}
.layout__nav-icon {
  flex-shrink: 0;
}

.layout__aside-footer {
  padding: 8px;
}
.layout__collapse-btn--center {
  margin: 0 auto;
}

/* ── Main column ── */
.layout__main {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
}

.layout__topbar {
  z-index: 20;
  display: flex;
  height: 64px;
  flex-shrink: 0;
  align-items: center;
  justify-content: flex-end;
  gap: 4px;
  padding: 0 12px;
  transition: box-shadow 300ms var(--ease-out-quart);
}
@media (min-width: 640px) {
  .layout__topbar {
    padding: 0 16px;
  }
}
.layout__topbar--scrolled {
  border-bottom: 1px solid var(--hairline);
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.03);
}

.layout__avatar {
  display: grid;
  width: 36px;
  height: 36px;
  margin-left: 4px;
  place-items: center;
  border-radius: 999px;
  background: var(--track);
  font-size: 13px;
  font-weight: 600;
  color: var(--text-2);
  cursor: pointer;
  transition:
    background-color 200ms var(--ease-out-quart),
    color 200ms var(--ease-out-quart),
    transform 200ms var(--ease-out-quart);
}
.layout__avatar:hover {
  background: var(--faint);
  color: var(--text);
}
.layout__avatar:active {
  transform: scale(0.95);
}

.layout__content {
  flex: 1;
  overflow-y: auto;
}
.layout__content-inner {
  margin: 0 auto;
  width: 100%;
  max-width: 1600px;
  padding: 24px 20px 32px;
}
@media (min-width: 768px) {
  .layout__content-inner {
    padding: 32px;
  }
}
</style>