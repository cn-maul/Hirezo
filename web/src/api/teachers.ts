import client from './client'

export interface Teacher {
  id: number
  name: string
  gender: 'male' | 'female'
  age: number // 0 = 未填
  subject: string
  has_cert: number // 1 有 0 无
  phone: string
  education: string
  university: string
  major: string
  remark: string
  resume_file?: string // 入库简历的归档文件名，空 = 手工录入无简历
  created_at: string
  updated_at: string
}

export type TeacherInput = Omit<Teacher, 'id' | 'created_at' | 'updated_at'>

export interface ListParams {
  keyword?: string
  subject?: string
  gender?: 'male' | 'female' | ''
  education?: string
  hasCert?: 0 | 1 | ''
  page?: number
  size?: number
  order?: 'asc' | 'desc'
}

export interface ListResult {
  items: Teacher[]
  total: number
  page: number
  size: number
}

export const emptyTeacher: TeacherInput = {
  name: '',
  gender: 'male',
  age: 0,
  subject: '',
  has_cert: 0,
  phone: '',
  education: '',
  university: '',
  major: '',
  remark: '',
}

function toParams(p: ListParams): Record<string, any> {
  const params: Record<string, any> = {}
  if (p.keyword) params.keyword = p.keyword
  if (p.subject) params.subject = p.subject
  if (p.gender) params.gender = p.gender
  if (p.education) params.education = p.education
  if (p.hasCert !== undefined && p.hasCert !== '') params.hasCert = p.hasCert
  if (p.page) params.page = p.page
  if (p.size) params.size = p.size
  if (p.order) params.order = p.order
  return params
}

export async function fetchTeachers(p: ListParams = {}): Promise<ListResult> {
  const r = await client.get<ListResult>('/teachers', { params: toParams(p) })
  return r.data
}

export async function fetchTeacher(id: number): Promise<Teacher> {
  const r = await client.get<{ data: Teacher }>(`/teachers/${id}`)
  return r.data.data
}

export async function createTeacher(p: TeacherInput): Promise<Teacher> {
  const r = await client.post<{ data: Teacher }>('/teachers', p)
  return r.data.data
}

export async function updateTeacher(id: number, p: TeacherInput): Promise<Teacher> {
  const r = await client.put<{ data: Teacher }>(`/teachers/${id}`, p)
  return r.data.data
}

export async function deleteTeacher(id: number): Promise<void> {
  await client.delete(`/teachers/${id}`)
}

/** 批量硬删除，返回实际删除条数 */
export async function batchDeleteTeachers(ids: number[]): Promise<number> {
  const r = await client.post<{ data: { deleted: number } }>('/teachers/batch-delete', { ids })
  return r.data.data.deleted
}

/** 按当前筛选导出 xlsx 并触发下载（走 axios 以便错误可被捕获提示） */
export async function exportXlsx(p: ListParams = {}): Promise<void> {
  const r = await client.get('/teachers/export', { params: toParams(p), responseType: 'blob' })
  const url = URL.createObjectURL(r.data)
  const a = document.createElement('a')
  a.href = url
  a.download = `teachers-${new Date().toISOString().slice(0, 10)}.xlsx`
  a.click()
  URL.revokeObjectURL(url)
}

export const genderLabel: Record<string, string> = { male: '男', female: '女' }

/** 年龄 0 表示未填，展示为 “-” */
export function ageOrDash(age: number): string {
  return age > 0 ? String(age) : '-'
}