import client from './client'

export async function getSettings(): Promise<Record<string, string>> {
  const r = await client.get('/settings')
  return r.data?.data ?? {}
}

export async function updateSettings(settings: Record<string, string>): Promise<void> {
  await client.put('/settings', settings)
}

// ---------- 简历识别用的大模型配置（管理员，密钥永不回显原文） ----------

export interface LLMConfig {
  endpoint: string
  protocol: string
  model: string
  api_key_set: boolean
  api_key_hint: string
  configured: boolean
}

export interface LLMConfigInput {
  endpoint: string
  protocol: string
  model: string
  /** 留空表示不修改已保存的密钥 */
  api_key?: string
  /** true 时清除已保存的密钥 */
  clear_api_key?: boolean
}

export const LLM_PROTOCOLS = [
  { value: 'openai-chat', label: 'OpenAI Chat（DeepSeek / Qwen / Moonshot 等兼容）' },
  { value: 'openai-responses', label: 'OpenAI Responses' },
  { value: 'anthropic', label: 'Anthropic Messages（Claude）' },
] as const

export async function getLLMConfig(): Promise<LLMConfig> {
  const r = await client.get<{ data: LLMConfig }>('/settings/llm')
  return r.data.data
}

export async function updateLLMConfig(p: LLMConfigInput): Promise<LLMConfig> {
  const r = await client.put<{ data: LLMConfig }>('/settings/llm', p)
  return r.data.data
}