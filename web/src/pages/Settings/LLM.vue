<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import Card from '@/components/ui/Card.vue'
import CardHeader from '@/components/ui/CardHeader.vue'
import CardTitle from '@/components/ui/CardTitle.vue'
import CardDescription from '@/components/ui/CardDescription.vue'
import CardContent from '@/components/ui/CardContent.vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import Label from '@/components/ui/Label.vue'
import Select from '@/components/ui/Select.vue'
import SelectItem from '@/components/ui/SelectItem.vue'
import {
  LLM_PROTOCOLS,
  getLLMConfig,
  updateLLMConfig,
  type LLMConfigInput,
} from '@/api/settings'
import { useQuery, useMutation, invalidateQueries } from '@/composables/useQuery'
import { useToast } from '@/composables/useToast'
import { errMsg } from '@/lib/utils'

const EMPTY: LLMConfigInput = {
  endpoint: '',
  protocol: 'openai-chat',
  model: '',
  api_key: '',
}

const { success, error } = useToast()
const { data } = useQuery(['llm'], getLLMConfig)

const form = ref<LLMConfigInput>({ ...EMPTY })
const hint = ref('')

watch(
  () => data.value,
  (d) => {
    if (!d) return
    form.value = {
      endpoint: d.endpoint,
      protocol: d.protocol || 'openai-chat',
      model: d.model,
      api_key: '',
    }
    hint.value = d.api_key_set ? `已保存密钥：${d.api_key_hint}` : ''
  },
  { immediate: true },
)

const mutation = useMutation((body: LLMConfigInput) => updateLLMConfig(body))
const mutationPending = mutation.isPending

const save = async () => {
  const body: LLMConfigInput = {
    endpoint: form.value.endpoint.trim(),
    protocol: form.value.protocol,
    model: form.value.model.trim(),
  }
  const key = (form.value.api_key ?? '').trim()
  if (key) body.api_key = key
  try {
    const next = await mutation.mutate(body)
    success('模型配置已保存')
    invalidateQueries(['llm'])
    form.value.api_key = ''
    hint.value = next.api_key_set ? `已保存密钥：${next.api_key_hint}` : ''
  } catch (e) {
    error(errMsg(e, '保存失败'))
  }
}

const clearKey = async () => {
  try {
    await mutation.mutate({
      endpoint: form.value.endpoint.trim(),
      protocol: form.value.protocol,
      model: form.value.model.trim(),
      clear_api_key: true,
    })
    success('模型配置已保存')
    invalidateQueries(['llm'])
  } catch (e) {
    error(errMsg(e, '保存失败'))
  }
}

const dirty = computed(
  () =>
    !!data.value &&
    (form.value.endpoint.trim() !== data.value.endpoint ||
      form.value.protocol !== (data.value.protocol || 'openai-chat') ||
      form.value.model.trim() !== data.value.model ||
      !!(form.value.api_key ?? '').trim()),
)

const discard = () => {
  if (data.value) {
    form.value = {
      endpoint: data.value.endpoint,
      protocol: data.value.protocol || 'openai-chat',
      model: data.value.model,
      api_key: '',
    }
    hint.value = data.value.api_key_set ? `已保存密钥：${data.value.api_key_hint}` : ''
  } else {
    form.value = { ...EMPTY }
    hint.value = ''
  }
}
</script>

<template>
  <div class="llm">
    <h2 class="llm__title">模型配置</h2>

    <Card>
      <CardHeader>
        <CardTitle class="llm__card-title">
          大模型接入
          <span
            :class="
              data?.configured
                ? 'llm__status llm__status--configured'
                : 'llm__status'
            "
          >
            {{ data?.configured ? '已配置' : '未配置' }}
          </span>
        </CardTitle>
        <CardDescription>
          用于简历识别的字段抽取。支持 OpenAI Chat、OpenAI Responses 与 Anthropic Messages 三种协议，
          以及 DeepSeek、Qwen、Moonshot、Ollama、vLLM 等兼容服务。
        </CardDescription>
      </CardHeader>
      <CardContent class="llm__content">
        <div class="llm__field">
          <Label html-for="llm-endpoint">API 端点</Label>
          <Input
            id="llm-endpoint"
            v-model="form.endpoint"
            placeholder="https://api.deepseek.com/v1"
            :spell-check="false"
          />
          <p class="llm__hint">填基础地址（含 /v1 版本路径），不要带 ?key= 之类的查询参数。</p>
        </div>

        <div class="llm__grid">
          <div class="llm__field">
            <Label>协议</Label>
            <Select v-model="form.protocol" id="llm-protocol">
              <template #value="{ selected }">
                {{ LLM_PROTOCOLS.find((p) => p.value === selected)?.label ?? '选择协议' }}
              </template>
              <SelectItem v-for="p in LLM_PROTOCOLS" :key="p.value" :value="p.value">
                {{ p.label }}
              </SelectItem>
            </Select>
          </div>

          <div class="llm__field">
            <Label html-for="llm-model">模型名称</Label>
            <Input
              id="llm-model"
              v-model="form.model"
              placeholder="deepseek-chat / qwen-vl-max / claude-sonnet-4-5"
              :spell-check="false"
            />
          </div>
        </div>

        <div class="llm__field">
          <Label html-for="llm-key">API Key</Label>
          <div class="llm__key-row">
            <Input
              id="llm-key"
              type="password"
              autocomplete="off"
              v-model="form.api_key"
              :placeholder="hint ? `${hint}，留空表示不修改` : '本地服务可留空'"
              :spell-check="false"
            />
            <Button v-if="data?.api_key_set" variant="outline" :disabled="mutationPending" @click="clearKey">
              清除
            </Button>
          </div>
          <p class="llm__hint">
            密钥以 AES-256-GCM 加密后保存在本地数据库，任何接口都不会回显原文；
            备份数据时请连同数据库同目录的
            <code class="llm__code">.hirezo-secret</code> 密钥文件一起备份。
          </p>
        </div>

        <div class="llm__actions">
          <Button :loading="mutationPending" :disabled="!dirty" @click="save">保存</Button>
          <Button v-if="dirty" variant="ghost" @click="discard">放弃修改</Button>
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
</template>

<style scoped>
.llm {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}
.llm__title {
  font-size: 19px;
  line-height: 1.2;
  font-weight: 600;
  letter-spacing: -0.02em;
}
.llm__card-title {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}
.llm__status {
  border-radius: 999px;
  padding: 2px 8px;
  font-size: 12px;
  font-weight: 500;
  background: var(--track);
  color: var(--text-2);
}
.llm__status--configured {
  background: var(--success-soft);
  color: var(--success-soft-fg);
}

.llm__content {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}
.llm__field {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}
.llm__grid {
  display: grid;
  grid-template-columns: 1fr;
  gap: 1rem;
}
@media (min-width: 640px) {
  .llm__grid {
    grid-template-columns: 1fr 1fr;
  }
}
.llm__key-row {
  display: flex;
  gap: 0.5rem;
}
.llm__hint {
  font-size: 12px;
  color: var(--text-3);
}
.llm__code {
  border-radius: var(--r-chip);
  background: var(--bg);
  padding: 2px 4px;
  font-family: var(--mono);
}
.llm__actions {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}
</style>