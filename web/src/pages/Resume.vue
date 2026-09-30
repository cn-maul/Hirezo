<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { FileText, RefreshCw, Trash2, UploadCloud } from 'lucide-vue-next'
import PageHeader from '@/components/PageHeader.vue'
import Button from '@/components/ui/Button.vue'
import DeleteConfirm from '@/components/DeleteConfirm.vue'
import TeacherFormDialog from '@/components/teachers/TeacherFormDialog.vue'
import { errMsg } from '@/lib/utils'
import { fetchDictionaries } from '@/api/dicts'
import { getLLMConfig } from '@/api/settings'
import type { Teacher, TeacherInput } from '@/api/teachers'
import {
  RESUME_MAX_BYTES,
  commitResumeDraft,
  deleteResumeDraft,
  fetchResumeDraft,
  fetchResumeDrafts,
  isResumeFile,
  parseResume,
  resumeTypeLabel,
  type ResumeDraft,
} from '@/api/resume'
import { useQuery, useMutation, invalidateQueries } from '@/composables/useQuery'
import { useToast } from '@/composables/useToast'

const DRAFT_PAGE_SIZE = 10

const router = useRouter()
const { success, error } = useToast()

const inputRef = ref<HTMLInputElement | null>(null)
const dragOver = ref(false)
const busy = ref(false)
const current = ref<ResumeDraft | null>(null)
const open = ref(false)

const { data: llm } = useQuery(['llm'], getLLMConfig)
const { data: subjects } = useQuery<import('@/api/dicts').Dictionary[]>(['dicts', 'subject'], () =>
  fetchDictionaries('subject'),
)
const { data: educations } = useQuery<import('@/api/dicts').Dictionary[]>(
  ['dicts', 'education'],
  () => fetchDictionaries('education'),
)
const { data: drafts } = useQuery(['resume-drafts'], () => fetchResumeDrafts(1, DRAFT_PAGE_SIZE))

const commitMutation = useMutation(({ id, fields }: { id: number; fields: TeacherInput }) =>
  commitResumeDraft(id, fields),
)
const commitPending = commitMutation.isPending
const deleteMutation = useMutation((id: number) => deleteResumeDraft(id))

const startRecognize = async (file: File) => {
  if (!isResumeFile(file)) {
    error('仅支持 .docx 与 .pdf 文件')
    return
  }
  if (file.size > RESUME_MAX_BYTES) {
    error('文件超过 10MB 上限')
    return
  }
  busy.value = true
  try {
    const draft = await parseResume(file)
    current.value = draft
    open.value = true
    invalidateQueries(['resume-drafts'])
  } catch (e) {
    error(errMsg(e, '识别失败，请稍后重试'))
  } finally {
    busy.value = false
  }
}

const onPick = (files: FileList | null) => {
  const f = files?.[0]
  if (f) void startRecognize(f)
}

const reopen = async (id: number) => {
  busy.value = true
  try {
    const draft = await fetchResumeDraft(id)
    current.value = draft
    open.value = true
  } catch (e) {
    error(errMsg(e, '读取识别记录失败'))
  } finally {
    busy.value = false
  }
}

const notConfigured = computed(() => llm.value !== undefined && !llm.value.configured)

const onCommit = async (fields: TeacherInput) => {
  if (!current.value) return
  try {
    const t = await commitMutation.mutate({ id: current.value.id, fields })
    success(`已入库：${t.name}`)
    open.value = false
    current.value = null
    invalidateQueries(['teachers'])
    invalidateQueries(['resume-drafts'])
  } catch (e) {
    error(errMsg(e, '入库失败'))
  }
}

const onDeleteDraft = async (id: number) => {
  try {
    await deleteMutation.mutate(id)
    success('已丢弃识别记录')
    invalidateQueries(['resume-drafts'])
  } catch (e) {
    error(errMsg(e, '删除失败'))
  }
}

function formatSize(n: number) {
  if (n <= 0) return '-'
  return n < 1024 * 1024 ? `${(n / 1024).toFixed(0)} KB` : `${(n / 1024 / 1024).toFixed(1)} MB`
}

/** 草稿 payload.fields → 表单可用的 Teacher 形状（性别识别失败时给空串，交由表单拦截） */
function toTeacher(d: ResumeDraft): Teacher {
  const f = d.payload?.fields
  return {
    id: d.id,
    name: f?.name ?? '',
    gender: (f?.gender === 'female' ? 'female' : f?.gender === 'male' ? 'male' : '') as Teacher['gender'],
    age: f?.age ?? 0,
    subject: f?.subject ?? '',
    has_cert: f?.has_cert === 1 ? 1 : 0,
    phone: f?.phone ?? '',
    education: f?.education ?? '',
    university: f?.university ?? '',
    major: f?.major ?? '',
    remark: f?.remark ?? '',
    created_at: d.created_at,
    updated_at: d.created_at,
  }
}
</script>

<template>
  <div class="resume">
    <PageHeader
      title="简历识别"
      description="上传 .docx / .pdf 简历，自动抽取人员信息并预填表单，核对后一键入库。"
    />

    <div v-if="notConfigured" class="resume__not-configured">
      <p class="resume__not-configured-text">尚未配置大模型，简历识别不可用。</p>
      <Button size="sm" @click="router.push('/settings/llm')">去配置模型</Button>
    </div>

    <div
      class="resume__dropzone"
      :class="dragOver ? 'resume__dropzone--over' : ''"
      @dragover.prevent="dragOver = true"
      @dragleave="dragOver = false"
      @drop.prevent="
        dragOver = false;
        if (!busy) onPick(($event as DragEvent).dataTransfer?.files ?? null)
      "
    >
      <div class="resume__dropzone-icon">
        <UploadCloud :size="24" :stroke-width="1.75" />
      </div>
      <div class="resume__dropzone-text">
        <p class="resume__dropzone-title">拖拽简历到此处，或点击选择文件</p>
        <p class="resume__dropzone-hint">
          支持 .docx / .pdf · 单个文件不超过 10MB · 扫描件 PDF 需要服务器安装 poppler-utils
        </p>
      </div>
      <input
        ref="inputRef"
        type="file"
        accept=".docx,.pdf"
        class="resume__file-input"
        :disabled="busy || notConfigured"
        @change="
          onPick(($event.target as HTMLInputElement).files);
          ($event.target as HTMLInputElement).value = ''
        "
      />
      <Button
        :loading="busy"
        :disabled="notConfigured"
        @click="inputRef?.click()"
      >
        {{ busy ? '正在识别…（约 10-60 秒）' : '选择简历文件' }}
      </Button>
    </div>

    <section class="resume__section">
      <div class="resume__section-header">
        <h2 class="resume__section-title">识别记录</h2>
        <span class="resume__section-count">共 {{ drafts?.total ?? 0 }} 条，入库后自动清除</span>
      </div>

      <div v-if="!drafts?.items.length" class="resume__empty">
        暂无识别记录
      </div>

      <ul v-else class="resume__list">
        <li v-for="d in drafts.items" :key="d.id" class="resume__item">
          <div class="resume__item-main">
            <span class="resume__item-icon">
              <FileText :size="18" :stroke-width="1.75" />
            </span>
            <div class="resume__item-info">
              <p class="resume__item-name">{{ d.file_name }}</p>
              <p class="resume__item-meta">
                {{ resumeTypeLabel[d.file_type] ?? d.file_type }} · {{ formatSize(d.file_size) }}
                <template v-if="d.page_count > 0"> · {{ d.page_count }} 页</template>
                · {{ d.created_at }}
                <template v-if="(d.payload?.warnings?.length ?? 0) > 0">
                  · {{ d.payload.warnings.length }} 处待核对
                </template>
              </p>
            </div>
          </div>
          <div class="resume__item-actions">
            <Button variant="outline" size="sm" :disabled="busy" @click="reopen(d.id)">
              <RefreshCw :size="16" :stroke-width="1.75" />
              继续录入
            </Button>
            <DeleteConfirm
              title="丢弃识别记录"
              :description="`确定丢弃「${d.file_name}」的识别结果吗？原始文件不会被保留。`"
              @confirm="onDeleteDraft(d.id)"
            >
              <Button variant="ghost" size="icon" aria-label="丢弃">
                <Trash2 :size="16" :stroke-width="1.75" />
              </Button>
            </DeleteConfirm>
          </div>
        </li>
      </ul>
    </section>

    <TeacherFormDialog
      v-model:open="open"
      :draft="current ? toTeacher(current) : null"
      :subjects="subjects ?? []"
      :educations="educations ?? []"
      :submitting="commitPending"
      :field-meta="current?.payload?.meta"
      :warnings="current?.payload?.warnings"
      @submit="onCommit"
      @update:open="(v) => { if (!v) current = null }"
    >
      <template #extra>
        <details v-if="current?.raw_text" class="resume__raw">
          <summary class="resume__raw-summary">识别原文（用于核对）</summary>
          <pre class="resume__raw-text">{{ current.raw_text }}</pre>
        </details>
      </template>
    </TeacherFormDialog>
  </div>
</template>

<style scoped>
.resume {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.resume__not-configured {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  border-radius: var(--r-thumb);
  border: 1px solid rgba(255, 159, 10, 0.5);
  background: rgba(255, 159, 10, 0.1);
  padding: 12px 16px;
}
.resume__not-configured-text {
  font-size: 14px;
  color: var(--text);
}

.resume__dropzone {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.75rem;
  border-radius: var(--r-thumb);
  border: 2px dashed var(--hairline);
  background: var(--surface);
  box-shadow: var(--sh-card);
  outline: 1px solid var(--hairline);
  outline-offset: -1px;
  padding: 48px 24px;
  text-align: center;
  transition:
    border-color 200ms var(--ease-out-quart),
    background-color 200ms var(--ease-out-quart);
}
.resume__dropzone--over {
  border-color: var(--accent);
  background: rgba(0, 113, 227, 0.06);
}
.resume__dropzone-icon {
  display: grid;
  width: 48px;
  height: 48px;
  place-items: center;
  border-radius: 999px;
  background: var(--track);
}
.resume__dropzone-icon :deep(svg) {
  color: var(--text-2);
}
.resume__dropzone-text {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.resume__dropzone-title {
  font-size: 15px;
  font-weight: 500;
  color: var(--text);
}
.resume__dropzone-hint {
  font-size: 13px;
  color: var(--text-3);
}
.resume__file-input {
  display: none;
}

.resume__section {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}
.resume__section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.resume__section-title {
  font-size: 17px;
  font-weight: 600;
  letter-spacing: -0.02em;
}
.resume__section-count {
  font-size: 13px;
  color: var(--text-3);
}

.resume__empty {
  border-radius: var(--r-thumb);
  border: 1px solid var(--hairline);
  padding: 32px 16px;
  text-align: center;
  font-size: 14px;
  color: var(--text-3);
}

.resume__list {
  display: flex;
  flex-direction: column;
  border-radius: var(--r-thumb);
  border: 1px solid var(--hairline);
  overflow: hidden;
}
.resume__item {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  padding: 12px 16px;
  border-bottom: 1px solid var(--hairline);
}
.resume__item:last-child {
  border-bottom: 0;
}
.resume__item-main {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 0.75rem;
}
.resume__item-icon {
  display: grid;
  width: 36px;
  height: 36px;
  flex-shrink: 0;
  place-items: center;
  border-radius: var(--r-chip);
  background: var(--track);
}
.resume__item-icon :deep(svg) {
  color: var(--text-2);
}
.resume__item-info {
  min-width: 0;
}
.resume__item-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 14px;
  font-weight: 500;
  color: var(--text);
}
.resume__item-meta {
  font-size: 12px;
  color: var(--text-3);
}
.resume__item-actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.resume__raw {
  border-radius: var(--r-thumb);
  background: var(--track);
  padding: 8px 12px;
  font-size: 13px;
  color: var(--text-2);
}
.resume__raw-summary {
  cursor: pointer;
  user-select: none;
}
.resume__raw-text {
  margin-top: 8px;
  max-height: 224px;
  overflow: auto;
  white-space: pre-wrap;
  font-family: var(--font);
}
</style>