import { ref, watch, type Ref } from 'vue'

// 轻量数据获取：替代 @tanstack/react-query。
// 提供按 key 缓存、失效、refetch、loading/error 状态。
// 不做窗口聚焦重取、不做 GC；够用且零依赖。

export type QueryKeyPart = string | number | boolean | object | undefined | null
export type QueryKey = QueryKeyPart | QueryKeyPart[]

interface CacheEntry<T> {
  data: T
  version: number
}

const cache = new Map<string, CacheEntry<unknown>>()
const invalidated = new Set<string>()

function keyString(key: QueryKey): string {
  return Array.isArray(key) ? key.map((k) => JSON.stringify(k)).join('|') : JSON.stringify(key)
}

// 使某个 key 前缀失效（下次 useQuery 重新拉取）
export function invalidateQueries(key: QueryKey) {
  invalidated.add(keyString(key))
}

// 直接写入缓存（用于详情页保存后同步）
export function setQueryData<T>(key: QueryKey, data: T) {
  const k = keyString(key)
  cache.set(k, { data, version: (cache.get(k)?.version ?? 0) + 1 })
}

function isInvalidated(key: string): boolean {
  for (const prefix of invalidated) {
    if (key === prefix || key.startsWith(prefix + '|')) return true
  }
  return false
}

export interface UseQueryOptions {
  enabled?: Ref<boolean> | boolean
  staleTime?: number
}

/**
 * 拉取数据并缓存。key 变化时自动重取；invalidateQueries 使缓存失效后重取。
 * 返回 { data, isLoading, error, refetch }。
 */
export function useQuery<T>(
  key: QueryKey,
  fetcher: () => Promise<T>,
  options: UseQueryOptions = {},
) {
  const data = ref<T | undefined>(undefined)
  const isLoading = ref(true)
  const error = ref<Error | null>(null)
  const enabled = typeof options.enabled === 'boolean' ? ref(options.enabled) : (options.enabled ?? ref(true))

  const keyStr = keyString(key)

  const load = async () => {
    if (!enabled.value) {
      isLoading.value = false
      return
    }
    isLoading.value = true
    error.value = null
    try {
      const cached = cache.get(keyStr)
      if (cached && !isInvalidated(keyStr)) {
        data.value = cached.data as T
      } else {
        const result = await fetcher()
        cache.set(keyStr, { data: result, version: (cache.get(keyStr)?.version ?? 0) + 1 })
        invalidated.delete(keyStr)
        data.value = result
      }
    } catch (e) {
      error.value = e instanceof Error ? e : new Error(String(e))
    } finally {
      isLoading.value = false
    }
  }

  // key 变化时重取
  watch(
    () => keyStr,
    () => void load(),
    { immediate: true },
  )

  // enabled 从 false → true 时重取
  watch(enabled, (v) => {
    if (v) void load()
  })

  const refetch = () => load()

  return { data, isLoading, error, refetch }
}

/** 简单 mutation：执行异步操作并管理 pending/error。 */
export function useMutation<TArgs extends unknown[], TResult>(
  fn: (...args: TArgs) => Promise<TResult>,
) {
  const isPending = ref(false)
  const error = ref<Error | null>(null)

  const mutate = async (...args: TArgs): Promise<TResult> => {
    isPending.value = true
    error.value = null
    try {
      return await fn(...args)
    } catch (e) {
      error.value = e instanceof Error ? e : new Error(String(e))
      throw e
    } finally {
      isPending.value = false
    }
  }

  return { isPending, error, mutate }
}