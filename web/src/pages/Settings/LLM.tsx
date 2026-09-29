import { useEffect, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  LLM_PROTOCOLS,
  getLLMConfig,
  updateLLMConfig,
  type LLMConfigInput,
} from '@/api/settings'
import { errMsg } from '@/lib/utils'

const EMPTY: LLMConfigInput = {
  endpoint: '',
  protocol: 'openai-chat',
  model: '',
  api_key: '',
}

export default function LLM() {
  const queryClient = useQueryClient()
  const { data } = useQuery({ queryKey: ['llm'], queryFn: getLLMConfig })

  const [form, setForm] = useState<LLMConfigInput>(EMPTY)
  const [hint, setHint] = useState('')

  useEffect(() => {
    if (!data) return
    setForm({
      endpoint: data.endpoint,
      protocol: data.protocol || 'openai-chat',
      model: data.model,
      api_key: '',
    })
    setHint(data.api_key_set ? `已保存密钥：${data.api_key_hint}` : '')
  }, [data])

  const mutation = useMutation({
    mutationFn: (body: LLMConfigInput) => updateLLMConfig(body),
    onSuccess: (next) => {
      toast.success('模型配置已保存')
      queryClient.invalidateQueries({ queryKey: ['llm'] })
      setForm((f) => ({ ...f, api_key: '' }))
      setHint(next.api_key_set ? `已保存密钥：${next.api_key_hint}` : '')
    },
    onError: (e) => toast.error(errMsg(e, '保存失败')),
  })

  const save = () => {
    const body: LLMConfigInput = {
      endpoint: form.endpoint.trim(),
      protocol: form.protocol,
      model: form.model.trim(),
    }
    const key = (form.api_key ?? '').trim()
    if (key) body.api_key = key
    mutation.mutate(body)
  }

  const clearKey = () => {
    mutation.mutate({
      endpoint: form.endpoint.trim(),
      protocol: form.protocol,
      model: form.model.trim(),
      clear_api_key: true,
    })
  }

  const dirty =
    !!data &&
    (form.endpoint.trim() !== data.endpoint ||
      form.protocol !== (data.protocol || 'openai-chat') ||
      form.model.trim() !== data.model ||
      !!(form.api_key ?? '').trim())

  return (
    <div className="space-y-3">
      <h2 className="text-[19px] leading-tight font-semibold tracking-[-0.02em]">模型配置</h2>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            大模型接入
            <span
              className={
                'rounded-full px-2 py-0.5 text-[12px] font-medium ' +
                (data?.configured
                  ? 'bg-emerald-500/15 text-emerald-600 dark:text-emerald-400'
                  : 'bg-[var(--track)] text-[var(--text-2)]')
              }
            >
              {data?.configured ? '已配置' : '未配置'}
            </span>
          </CardTitle>
          <CardDescription>
            用于简历识别的字段抽取。支持 OpenAI Chat、OpenAI Responses 与 Anthropic Messages 三种协议，
            以及 DeepSeek、Qwen、Moonshot、Ollama、vLLM 等兼容服务。
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="llm-endpoint">API 端点</Label>
            <Input
              id="llm-endpoint"
              value={form.endpoint}
              onChange={(e) => setForm((f) => ({ ...f, endpoint: e.target.value }))}
              placeholder="https://api.deepseek.com/v1"
              spellCheck={false}
            />
            <p className="text-[12px] text-[var(--text-3)]">
              填基础地址（含 /v1 版本路径），不要带 ?key= 之类的查询参数。
            </p>
          </div>

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label>协议</Label>
              <Select
                value={form.protocol}
                onValueChange={(v) => setForm((f) => ({ ...f, protocol: v }))}
              >
                <SelectTrigger id="llm-protocol">
                  <SelectValue placeholder="选择协议" />
                </SelectTrigger>
                <SelectContent>
                  {LLM_PROTOCOLS.map((p) => (
                    <SelectItem key={p.value} value={p.value}>
                      {p.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label htmlFor="llm-model">模型名称</Label>
              <Input
                id="llm-model"
                value={form.model}
                onChange={(e) => setForm((f) => ({ ...f, model: e.target.value }))}
                placeholder="deepseek-chat / qwen-vl-max / claude-sonnet-4-5"
                spellCheck={false}
              />
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="llm-key">API Key</Label>
            <div className="flex gap-2">
              <Input
                id="llm-key"
                type="password"
                autoComplete="off"
                value={form.api_key ?? ''}
                onChange={(e) => setForm((f) => ({ ...f, api_key: e.target.value }))}
                placeholder={hint ? `${hint}，留空表示不修改` : '本地服务可留空'}
                spellCheck={false}
              />
              {data?.api_key_set && (
                <Button variant="outline" onClick={clearKey} disabled={mutation.isPending}>
                  清除
                </Button>
              )}
            </div>
            <p className="text-[12px] text-[var(--text-3)]">
              密钥以 AES-256-GCM 加密后保存在本地数据库，任何接口都不会回显原文；
              备份数据时请连同数据库同目录的 <code className="rounded-[var(--r-chip)] bg-background px-1 py-0.5 font-mono">.hirezo-secret</code> 密钥文件一起备份。
            </p>
          </div>

          <div className="flex items-center gap-3">
            <Button onClick={save} loading={mutation.isPending} disabled={!dirty}>
              保存
            </Button>
            {dirty && (
              <Button
                variant="ghost"
                onClick={() => {
                  setForm(
                    data
                      ? {
                          endpoint: data.endpoint,
                          protocol: data.protocol || 'openai-chat',
                          model: data.model,
                          api_key: '',
                        }
                      : EMPTY,
                  )
                  setHint(data?.api_key_set ? `已保存密钥：${data.api_key_hint}` : '')
                }}
              >
                放弃修改
              </Button>
            )}
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>扫描件支持</CardTitle>
          <CardDescription>
            没有文本层的 PDF（扫描件 / 图片型）需要服务器安装 poppler-utils 以提供 pdftoppm 命令；
            未安装时会明确报错，文本层 PDF 与 .docx 不受影响。
          </CardDescription>
        </CardHeader>
      </Card>
    </div>
  )
}
