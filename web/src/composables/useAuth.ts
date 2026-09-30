import { ref } from 'vue'
import { fetchAuthStatus, login as apiLogin, logout as apiLogout, type User } from '@/api/auth'

// 全局单例 auth 状态
const authed = ref<boolean | null>(null)
const user = ref<User | null>(null)

// Session检查间隔（5分钟）
const SESSION_CHECK_INTERVAL = 5 * 60 * 1000

let intervalId: ReturnType<typeof setInterval> | null = null

// 标记是否已由用户显式登录/登出（用于避免初始 checkStatus 的竞态覆盖）
let userActionTaken = false

async function checkStatus() {
  const { ok, user: u } = await fetchAuthStatus()
  // 若用户已显式登录/登出，忽略这次初始检查的迟到结果，避免竞态覆盖
  if (userActionTaken) return ok
  authed.value = ok
  user.value = u ?? null
  return ok
}

// 初始检查
void checkStatus()

// 定期检查 session 是否过期
function startInterval() {
  if (intervalId) return
  intervalId = setInterval(async () => {
    try {
      const ok = await checkStatus()
      if (!ok) stopInterval()
    } catch {
      // 网络错误时不处理，下次检查时再判断
    }
  }, SESSION_CHECK_INTERVAL)
}

function stopInterval() {
  if (intervalId) {
    clearInterval(intervalId)
    intervalId = null
  }
}

export function useAuth() {
  const login = async (username: string, password: string) => {
    const { user: u } = await apiLogin(username, password)
    userActionTaken = true
    authed.value = true
    user.value = u
    startInterval()
  }

  const logout = async () => {
    await apiLogout()
    userActionTaken = true
    authed.value = false
    user.value = null
    stopInterval()
  }

  return { authed, user, login, logout }
}