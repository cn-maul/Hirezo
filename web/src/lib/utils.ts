import type { CSSProperties } from 'react'
import { clsx, type ClassValue } from 'clsx'
import { twMerge } from 'tailwind-merge'

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

/**
 * 统一错误信息提取：axios 拦截器已把错误转换成 Error（消息为后端 error.message），
 * 其他来源（fetch / 运行时异常）返回 fallback。避免到处写 catch(e:any) 丢失类型信息。
 */
export function errMsg(e: unknown, fallback = '请求失败，请稍后再试'): string {
  return e instanceof Error && e.message ? e.message : fallback
}

/**
 * 分类色胶囊的配对色：淡底 + 同色相文字。
 *
 * 颜色来自后端配置（用户数据），不能铺满整块，但也不该缩成一个小点——
 * 6px 的点在灰胶囊里几乎看不见，等于没标。折中是让颜色上底、文字取同色相，
 * 既保住辨识度，又不到「满色块」那种吵闹。
 *
 * 只把原始色交给 CSS 变量 `--cat`，混色比例由 index.css 统一控制
 * （深浅模式各一套），组件不重复一遍深浅判断。
 *
 * 非法的颜色值返回 null，调用方回退到中性灰胶囊。
 */
export function categoryTint(color?: string): CSSProperties | null {
  const hex = (color ?? '').trim()
  if (!/^#[0-9a-fA-F]{6}$/.test(hex)) return null
  return { '--cat': hex } as CSSProperties
}
