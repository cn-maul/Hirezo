<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import GraduationCap from 'lucide-vue-next/dist/esm/icons/graduation-cap.js'
import User from 'lucide-vue-next/dist/esm/icons/user.js'
import Lock from 'lucide-vue-next/dist/esm/icons/lock.js'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import Label from '@/components/ui/Label.vue'
import Card from '@/components/ui/Card.vue'
import CardHeader from '@/components/ui/CardHeader.vue'
import CardTitle from '@/components/ui/CardTitle.vue'
import CardContent from '@/components/ui/CardContent.vue'
import { useAuth } from '@/composables/useAuth'
import { useToast } from '@/composables/useToast'
import { useForm } from '@/composables/useForm'
import { useQuery } from '@/composables/useQuery'
import { getSettings } from '@/api/settings'
import { validateLogin, type LoginFormValues } from '@/lib/validation'
import { errMsg } from '@/lib/utils'

const { login } = useAuth()
const { error } = useToast()
const route = useRoute()
const router = useRouter()
const loading = ref(false)

const { data: settings } = useQuery(['settings'], getSettings, { staleTime: 60_000 })
const siteName = computed(() => settings.value?.site_name || 'Hirezo 教师管理')

const { values, errors, submit } = useForm<LoginFormValues>(
  { username: '', password: '' },
  validateLogin,
)

const onFinish = async ({ username, password }: LoginFormValues) => {
  loading.value = true
  try {
    await login(username, password)
    // 防止开放重定向：解码后必须以/开头，不能是/login，不能以//开头（协议相对URL）。
    let safeNext = '/'
    const next = route.query.next
    if (typeof next === 'string') {
      try {
        const decoded = decodeURIComponent(next)
        if (decoded.startsWith('/') && !decoded.startsWith('//') && !decoded.startsWith('/login')) {
          safeNext = decoded
        }
      } catch {
        // 非法百分号编码：忽略，回首页
      }
    }
    void router.replace(safeNext)
  } catch (e) {
    error(errMsg(e, '登录失败'))
  } finally {
    loading.value = false
  }
}

// 预绑定 submit 处理器（避免 @submit="submit(onFinish)" 被 Vue 编译器误判为 withModifiers）
const onSubmit = submit(onFinish)
</script>

<template>
  <div class="login">
    <Card class="login__card">
      <CardHeader class="login__header">
        <div class="login__logo">
          <GraduationCap :size="24" :stroke-width="1.75" />
        </div>
        <CardTitle class="login__title">{{ siteName }}</CardTitle>
      </CardHeader>
      <CardContent>
        <form class="login__form" @submit="onSubmit">
          <div class="login__field">
            <Label html-for="username">用户名</Label>
            <div class="login__input-wrap">
              <User :size="18" :stroke-width="1.75" class="login__input-icon" />
              <Input
                id="username"
                class="login__input"
                placeholder="用户名"
                autofocus
                autocomplete="username"
                v-model="values.username"
              />
            </div>
            <p v-if="errors.username" class="login__error">{{ errors.username }}</p>
          </div>
          <div class="login__field">
            <Label html-for="password">密码</Label>
            <div class="login__input-wrap">
              <Lock :size="18" :stroke-width="1.75" class="login__input-icon" />
              <Input
                id="password"
                type="password"
                class="login__input"
                placeholder="密码"
                autocomplete="current-password"
                v-model="values.password"
              />
            </div>
            <p v-if="errors.password" class="login__error">{{ errors.password }}</p>
          </div>
          <Button type="submit" class="login__submit" size="lg" :loading="loading">
            登录
          </Button>
        </form>
      </CardContent>
    </Card>
  </div>
</template>

<style scoped>
.login {
  display: flex;
  min-height: 100vh;
  align-items: center;
  justify-content: center;
  background: var(--bg);
  padding: 20px;
}

.login__card {
  width: 100%;
  max-width: 380px;
  gap: 1.5rem;
  border-radius: var(--r-hero);
  padding: 32px 4px;
  box-shadow: var(--sh-panel);
}

.login__header {
  align-items: center;
  text-align: center;
}

.login__logo {
  display: grid;
  width: 52px;
  height: 52px;
  margin-bottom: 8px;
  place-items: center;
  border-radius: var(--r-thumb);
  background: var(--accent);
  color: #fff;
}

.login__title {
  font-size: 21px;
  letter-spacing: -0.02em;
}

.login__form {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.login__field {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.login__input-wrap {
  position: relative;
}

.login__input-icon {
  position: absolute;
  top: 50%;
  left: 12px;
  transform: translateY(-50%);
  color: var(--text-3);
  pointer-events: none;
}

.login__input {
  padding-left: 40px;
}

.login__error {
  font-size: 13px;
  color: var(--danger);
}

.login__submit {
  margin-top: 8px;
  width: 100%;
}
</style>