<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import RefreshCw from 'lucide-vue-next/dist/esm/icons/refresh-cw.js'
import UploadCloud from 'lucide-vue-next/dist/esm/icons/cloud-upload.js'
import Button from '@/components/ui/Button.vue'
import TeacherForm from '@/components/teachers/TeacherForm.vue'
import { errMsg } from '@/lib/utils'
import { fetchDictionaries } from '@/api/dicts'
import { getLLMConfig } from '@/api/settings'
import type { Teacher, TeacherInput } from '@/api/teachers'
import {
  RESUME_MAX_BYTES,
  commitResumeDraft,
  isResumeFile,
  parseResume,
  resumeTypeLabel,
  type ResumeDraft,
} from '@/api/resume'
import { useQuery, useMutation, invalidateQueries } from '@/composables/useQuery'
import { useToast } from '@/composables/useToast'

const router = useRouter()
const { success, error } = useToast()

const inputRef = ref<HTMLInputElement | null>(null)
const dragOver = ref(false)
const current = ref<ResumeDraft | null>(null)

// ---------- 识别会话：左原件预览，右识别状态；识别完成后右侧就地变成核对表单 ----------

type SessionStatus = 'parsing' | 'error' | 'done'

interface ResumeSession {
  file: File | null
  url: string
  isPdf: boolean
  status: SessionStatus
  message: string
}

const session = reactive<ResumeSession>({
  file: null,
  url: '',
  isPdf: false,
  status: 'parsing',
  message: '',
})

const docxContainer = ref<HTMLElement | null>(null)
const elapsed = ref(0)
let timer: ReturnType<typeof setInterval> | null = null
let blobUrl: string | null = null

const inSession = computed(() => session.file !== null)

const parsing = computed(() => inSession.value && session.status === 'parsing')
const failed = computed(() => inSession.value && session.status === 'error')

const formDraft = computed(() => (current.value ? toTeacher(current.value) : null))
const formMeta = computed(() => current.value?.payload?.meta)
const formWarnings = computed(() => current.value?.payload?.warnings)

function stopTimer() {
  if (timer) clearInterval(timer)
  timer = null
}

function startTimer() {
  stopTimer()
  elapsed.value = 0
  timer = setInterval(() => (elapsed.value += 1), 1000)
}

onBeforeUnmount(() => {
  stopTimer()
  if (blobUrl) URL.revokeObjectURL(blobUrl)
})

/** 渲染原件：PDF 用浏览器原生查看器；docx 懒加载渲染库（分块仅识别页加载） */
async function renderPreview() {
  if (!inSession.value || session.isPdf || !docxContainer.value) return
  try {
    const { renderAsync } = await import('docx-preview')
    if (docxContainer.value && session.file) {
      docxContainer.value.innerHTML = ''
      await renderAsync(session.file, docxContainer.value, undefined, {
        inWrapper: true,
        ignoreLastRenderedPageBreak: true,
      })
    }
  } catch (e) {
    error(errMsg(e, '简历预览渲染失败'))
  }
}

watch(
  () => session.file,
  async (f) => {
    if (!f) return
    await nextTick()
    void renderPreview()
  },
)

function clearSession() {
  stopTimer()
  if (blobUrl) URL.revokeObjectURL(blobUrl)
  blobUrl = null
  if (docxContainer.value) docxContainer.value.innerHTML = ''
  session.file = null
  session.url = ''
  session.status = 'parsing'
  session.message = ''
}

/** 结束一次核对：清掉结果与预览，回到上传区 */
function resetWorkbench() {
  current.value = null
  clearSession()
}

const startRecognize = async (file: File) => {
  if (!isResumeFile(file)) {
    error('仅支持 .docx 与 .pdf 文件')
    return
  }
  if (file.size > RESUME_MAX_BYTES) {
    error('文件超过 10MB 上限')
    return
  }
  if (blobUrl) URL.revokeObjectURL(blobUrl)
  blobUrl = URL.createObjectURL(file)
  current.value = null
  session.file = file
  session.url = blobUrl
  session.isPdf = /\.pdf$/i.test(file.name)
  session.status = 'parsing'
  session.message = ''
  startTimer()
  try {
    const draft = await parseResume(file)
    stopTimer()
    current.value = draft
    session.status = 'done'
  } catch (e) {
    stopTimer()
    session.status = 'error'
    session.message = errMsg(e, '识别失败，请稍后重试')
  }
}

const retryRecognize = () => {
  if (session.file) void startRecognize(session.file)
}

const onPick = (files: FileList | null) => {
  const f = files?.[0]
  if (f) void startRecognize(f)
}

const onCommit = async (fields: TeacherInput) => {
  if (!current.value) return
  try {
    const t = await commitMutation.mutate({ id: current.value.id, fields })
    success(`已入库：${t.name}`)
    resetWorkbench()
    invalidateQueries(['teachers'])
  } catch (e) {
    error(errMsg(e, '入库失败'))
  }
}

const { data: llm } = useQuery(['llm'], getLLMConfig)
const { data: subjects } = useQuery<import('@/api/dicts').Dictionary[]>(['dicts', 'subject'], () =>
  fetchDictionaries('subject'),
)
const { data: educations } = useQuery<import('@/api/dicts').Dictionary[]>(
  ['dicts', 'education'],
  () => fetchDictionaries('education'),
)
const commitMutation = useMutation(({ id, fields }: { id: number; fields: TeacherInput }) =>
  commitResumeDraft(id, fields),
)
const commitPending = commitMutation.isPending

function formatSize(n: number) {
  if (n <= 0) return '-'
  return n < 1024 * 1024 ? `${(n / 1024).toFixed(0)} KB` : `${(n / 1024 / 1024).toFixed(1)} MB`
}

const notConfigured = computed(() => llm.value !== undefined && !llm.value.configured)

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
    <div v-if="notConfigured" class="resume__not-configured">
      <p class="resume__not-configured-text">尚未配置大模型，简历识别不可用。</p>
      <Button size="sm" @click="router.push('/settings/llm')">去配置模型</Button>
    </div>

    <!-- 识别中 / 识别失败：左原件预览，右状态；识别完成后右侧就地变成核对表单 -->
    <div v-if="inSession" class="resume__split">
      <div class="resume__preview" aria-label="简历原件预览">
        <div class="resume__preview-body">
          <iframe
            v-if="session.isPdf"
            :src="session.url"
            class="resume__preview-frame"
            title="简历预览"
          ></iframe>
          <div v-else ref="docxContainer" class="resume__preview-docx"></div>
        </div>
      </div>

      <section class="resume__panel">
        <div v-if="parsing" class="resume__status">
          <RefreshCw :size="28" :stroke-width="1.75" class="resume__status-spin" />
          <p class="resume__status-title">AI 正在识别…</p>
          <p class="resume__status-hint">通常 10~60 秒完成，识别完成后右侧会直接显示核对表单</p>
          <p class="resume__status-meta">
            {{ resumeTypeLabel[session.isPdf ? 'pdf' : 'docx'] }} ·
            {{ formatSize(session.file?.size ?? 0) }} · 已用时 {{ elapsed }}s
          </p>
        </div>

        <div v-else-if="failed" class="resume__status resume__status--error">
          <p class="resume__status-title">识别失败</p>
          <p class="resume__status-hint">{{ session.message }}</p>
          <div class="resume__status-actions">
            <Button size="sm" @click="retryRecognize">
              <RefreshCw :size="16" :stroke-width="1.75" />
              重试
            </Button>
            <Button variant="outline" size="sm" @click="resetWorkbench">放弃</Button>
          </div>
        </div>

        <TeacherForm
          v-else-if="current"
          :draft="formDraft"
          :subjects="subjects ?? []"
          :educations="educations ?? []"
          :submitting="commitPending"
          :field-meta="formMeta"
          :warnings="formWarnings"
          submit-label="确认入库"
          cancel-label="返回上传"
          @submit="onCommit"
          @cancel="resetWorkbench"
        />
      </section>
    </div>

    <div
      v-else
      class="resume__dropzone"
      :class="dragOver ? 'resume__dropzone--over' : ''"
      @dragover.prevent="dragOver = true"
      @dragleave="dragOver = false"
      @drop.prevent="
        dragOver = false;
        onPick(($event as DragEvent).dataTransfer?.files ?? null)
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
        :disabled="notConfigured"
        @change="
          onPick(($event.target as HTMLInputElement).files);
          ($event.target as HTMLInputElement).value = ''
        "
      />
      <Button :disabled="notConfigured" @click="inputRef?.click()">选择简历文件</Button>
    </div>
  </div>
</template>

<style scoped>
.resume {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
  /* 整页上移 16px：内容区默认 32px 上留白对识别页太奢侈，让位给双栏高度 */
  margin-top: -16px;
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

/* ---------- 识别会话双栏 ---------- */

/* 两栏等高，占满一屏：100vh 扣掉顶栏 64 + 上移后的留白 16 + 下留白 32；底部留白即两栏与页面下缘的距离。
   右栏用 clamp 保证至少 420px，好让表单排得下两列；余量都留给原件预览。 */
.resume__split {
  display: grid;
  grid-template-columns: minmax(0, 1fr) clamp(420px, 42%, 560px);
  grid-template-rows: minmax(0, 1fr);
  gap: 1rem;
  height: calc(100vh - 112px);
  min-height: 420px;
}
@media (max-width: 900px) {
  .resume__split {
    grid-template-columns: 1fr;
    grid-template-rows: auto;
    height: auto;
  }
  .resume__preview {
    height: 60vh;
  }
  .resume__panel {
    overflow: visible;
  }
}

.resume__preview {
  display: flex;
  min-width: 0;
  min-height: 0;
  flex-direction: column;
  border-radius: var(--r-thumb);
  border: 1px solid var(--hairline);
  background: var(--surface);
  box-shadow: var(--sh-card);
  overflow: hidden;
}
.resume__preview-body {
  flex: 1;
  min-height: 0;
  overflow: auto;
  background: var(--track);
}
.resume__preview-frame {
  display: block;
  width: 100%;
  height: 100%;
  border: 0;
}
.resume__preview-docx {
  padding: 8px;
}
.resume__preview-docx :deep(.docx-wrapper) {
  padding: 12px 0;
  background: transparent;
}
.resume__preview-docx :deep(.docx-wrapper > section.docx) {
  margin-bottom: 12px;
  box-shadow: var(--sh-card);
}

.resume__panel {
  display: flex;
  min-width: 0;
  min-height: 0;
  flex-direction: column;
  gap: 0.75rem;
  overflow: auto;
  overscroll-behavior: contain;
  border-radius: var(--r-thumb);
  border: 1px solid var(--hairline);
  background: var(--surface);
  box-shadow: var(--sh-card);
  padding: 16px;
}
.resume__status {
  display: flex;
  flex: 1;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  min-height: 320px;
  padding: 32px 16px;
  text-align: center;
}
.resume__status--error {
  color: var(--danger);
}
.resume__status-spin {
  color: var(--accent);
  animation: resume-spin 1.2s linear infinite;
}
@keyframes resume-spin {
  to {
    transform: rotate(360deg);
  }
}
.resume__status-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--text);
}
.resume__status-hint {
  font-size: 13px;
  color: var(--text-3);
}
.resume__status-meta {
  font-size: 12px;
  color: var(--text-3);
  font-variant-numeric: tabular-nums;
}
.resume__status-actions {
  display: flex;
  gap: 8px;
  margin-top: 8px;
}

/* 右栏表单刻意排两列（姓名/性别、年龄/电话、学历/院校、学科/专业），备注与教资整幅 */
.resume__panel :deep(.teacher-grid) {
  grid-template-columns: 1fr 1fr;
}
@media (max-width: 560px) {
  .resume__panel :deep(.teacher-grid) {
    grid-template-columns: 1fr;
  }
}
</style>