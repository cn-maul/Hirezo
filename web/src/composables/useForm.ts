import { reactive, ref } from 'vue'
import type { Errors } from '@/lib/validation'

// 极简表单状态 composable（替代 react-hook-form / 原 useFormState Hook）。
// 提交时整体校验；出错字段在再次输入时清除错误。

export function useForm<F extends object>(initial: F, validate: (v: F) => Errors<F>) {
  const values = reactive<F>({ ...initial }) as F
  const errors = ref<Errors<F>>({})

  const set = <K extends keyof F>(key: K, value: F[K]) => {
    values[key] = value
    if (errors.value[key]) {
      const next = { ...errors.value }
      delete next[key]
      errors.value = next
    }
  }

  const getValues = (): F => ({ ...values })

  const reset = (next?: F) => {
    const base = next === undefined ? initial : next
    Object.keys(values).forEach((k) => {
      delete (values as Record<string, unknown>)[k]
    })
    Object.assign(values, base)
    errors.value = {}
  }

  // 包装 form onSubmit：阻止默认提交 → 整体校验 → 全部通过才调用 onValid
  const submit = (onValid: (v: F) => void | Promise<void>) => {
    return (e: Event) => {
      e.preventDefault()
      const errs = validate(getValues())
      errors.value = errs
      if (!Object.values(errs).some(Boolean)) void onValid(getValues())
    }
  }

  return { values, errors, set, getValues, reset, submit }
}