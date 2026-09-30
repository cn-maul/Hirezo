import { ref, watch } from 'vue'

// 主题状态：亮/暗切换，持久化到 localStorage，切换时在 <html> 上挂 .dark。
const dark = ref(false)

// 初始化：读取 localStorage
try {
  dark.value = localStorage.getItem('hirezo.dark') === '1'
} catch {
  dark.value = false
}

// 同步到 DOM 与 localStorage
watch(
  dark,
  (v) => {
    document.documentElement.classList.toggle('dark', v)
    try {
      localStorage.setItem('hirezo.dark', v ? '1' : '0')
    } catch {
      /* ignore */
    }
  },
  { immediate: true },
)

export function useTheme() {
  const toggle = () => {
    dark.value = !dark.value
  }
  return { dark, toggle }
}