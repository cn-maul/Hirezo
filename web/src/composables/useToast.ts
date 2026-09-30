import { ref } from 'vue'

// 轻量 toast 系统：替代 sonner。全局单例，由 <ToastHost /> 渲染。

export type ToastType = 'success' | 'error' | 'info'

export interface ToastItem {
  id: number
  type: ToastType
  message: string
  duration: number
}

const toasts = ref<ToastItem[]>([])

let nextId = 1

function push(type: ToastType, message: string, duration: number) {
  const id = nextId++
  toasts.value.push({ id, type, message, duration })
  if (duration > 0) {
    setTimeout(() => dismiss(id), duration)
  }
  return id
}

export function dismiss(id: number) {
  toasts.value = toasts.value.filter((t) => t.id !== id)
}

export function useToast() {
  const success = (message: string, duration = 3000) => push('success', message, duration)
  const error = (message: string, duration = 4000) => push('error', message, duration)
  const info = (message: string, duration = 3000) => push('info', message, duration)
  return { success, error, info, dismiss }
}

// 供 ToastHost 读取
export function useToastState() {
  return toasts
}