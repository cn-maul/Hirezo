import client from './client'
import type { Teacher, TeacherInput } from './teachers'

// 简历识别：上传 → 服务端解析与模型抽取 → 草稿 → 人工修正 → 入库。

export type ResumeFieldType = 'ok' | 'warn' | 'error'

export interface ResumeFieldMeta {
  evidence: string
  confidence: number
  level: ResumeFieldType
}

export interface ResumeWarning {
  field: string
  message: string
  level: 'warn' | 'error'
}

export interface ResumePayload {
  fields: TeacherInput
  meta: Record<string, ResumeFieldMeta>
  warnings: ResumeWarning[]
  summary: string
}

export interface ResumeDraft {
  id: number
  file_name: string
  file_type: 'docx' | 'pdf'
  file_size: number
  page_count: number
  created_at: string
  payload: ResumePayload
  raw_text?: string
}

export interface ResumeDraftList {
  items: ResumeDraft[]
  total: number
  page: number
  size: number
}

/** 简历类型 → 展示名 */
export const resumeTypeLabel: Record<string, string> = { docx: 'Word', pdf: 'PDF' }

/**
 * 上传并同步识别简历。模型调用最长 90s，这里单独放宽超时
 * （client.ts 默认 15s 只适合常规接口）。
 */
export async function parseResume(file: File): Promise<ResumeDraft> {
  const fd = new FormData()
  fd.append('file', file)
  const r = await client.post<{ data: ResumeDraft }>('/resume/parse', fd, { timeout: 130000 })
  return r.data.data
}

export async function fetchResumeDrafts(page = 1, size = 10): Promise<ResumeDraftList> {
  const r = await client.get<ResumeDraftList>('/resume/drafts', { params: { page, size } })
  return r.data
}

export async function fetchResumeDraft(id: number): Promise<ResumeDraft> {
  const r = await client.get<{ data: ResumeDraft }>(`/resume/drafts/${id}`)
  return r.data.data
}

export async function deleteResumeDraft(id: number): Promise<void> {
  await client.delete(`/resume/drafts/${id}`)
}

/** 用人工修正后的字段入库，成功后服务端自动删除对应草稿 */
export async function commitResumeDraft(id: number, fields: TeacherInput): Promise<Teacher> {
  const r = await client.post<{ data: Teacher }>(`/resume/drafts/${id}/commit`, { fields })
  return r.data.data
}

/** 本地预校验：与后端 accept 白名单一致，避免上传后才报错 */
export function isResumeFile(file: File): boolean {
  return /\.(docx|pdf)$/i.test(file.name)
}

/** 下载入库简历原件（blob 走 axios 以便 401/404 可被捕获提示） */
export async function downloadTeacherResume(id: number, name: string): Promise<void> {
  const r = await client.get<Blob>(`/teachers/${id}/resume`, { responseType: 'blob' })
  const ext = r.data.type === 'application/pdf' ? 'pdf' : 'docx'
  const url = URL.createObjectURL(r.data)
  const a = document.createElement('a')
  a.href = url
  a.download = `${name}-简历.${ext}`
  a.click()
  URL.revokeObjectURL(url)
}

export const RESUME_MAX_BYTES = 10 << 20