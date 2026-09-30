import client from './client'

export type DictKind = 'subject' | 'education'

export interface Dictionary {
  id: number
  kind: DictKind
  name: string
  color: string
  sort: number
  enabled: number // 1 启用 0 停用
}

/** 拉取某类字典的全部条目（按 sort,id 排序，含停用项） */
export async function fetchDictionaries(kind: DictKind): Promise<Dictionary[]> {
  const r = await client.get<{ data: Dictionary[] }>(`/dictionaries/${kind}`)
  return r.data.data ?? []
}

export async function createDictionary(
  kind: DictKind,
  p: { name: string; color?: string; sort?: number; enabled?: number },
): Promise<Dictionary> {
  const r = await client.post<{ data: Dictionary }>(`/dictionaries/${kind}`, p)
  return r.data.data
}

export async function updateDictionary(
  kind: DictKind,
  id: number,
  p: { name?: string; color?: string; sort?: number; enabled?: number },
): Promise<Dictionary> {
  const r = await client.put<{ data: Dictionary }>(`/dictionaries/${kind}/${id}`, p)
  return r.data.data
}

/** 删除；被教师引用时后端返回 409 并携带原因 */
export async function deleteDictionary(kind: DictKind, id: number): Promise<void> {
  await client.delete(`/dictionaries/${kind}/${id}`)
}