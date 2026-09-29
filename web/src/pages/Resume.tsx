import { useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { FileText, RefreshCw, Trash2, UploadCloud } from 'lucide-react'
import PageHeader from '@/components/PageHeader'
import { Button } from '@/components/ui/button'
import DeleteConfirm from '@/components/DeleteConfirm'
import TeacherFormDialog from '@/components/teachers/TeacherFormDialog'
import { cn, errMsg } from '@/lib/utils'
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

const DRAFT_PAGE_SIZE = 10

function formatSize(n: number) {
  if (n <= 0) return '-'
  return n < 1024 * 1024 ? `${(n / 1024).toFixed(0)} KB` : `${(n / 1024 / 1024).toFixed(1)} MB`
}

export default function Resume() {
  const queryClient = useQueryClient()
  const inputRef = useRef<HTMLInputElement>(null)
  const [dragOver, setDragOver] = useState(false)
  const [busy, setBusy] = useState(false)
  const [current, setCurrent] = useState<ResumeDraft | null>(null)
  const [open, setOpen] = useState(false)

  const { data: llm } = useQuery({ queryKey: ['llm'], queryFn: getLLMConfig })
  const { data: subjects } = useQuery({
    queryKey: ['dicts', 'subject'],
    queryFn: () => fetchDictionaries('subject'),
  })
  const { data: educations } = useQuery({
    queryKey: ['dicts', 'education'],
    queryFn: () => fetchDictionaries('education'),
  })
  const { data: drafts } = useQuery({
    queryKey: ['resume-drafts'],
    queryFn: () => fetchResumeDrafts(1, DRAFT_PAGE_SIZE),
  })

  const commitMutation = useMutation({
    mutationFn: ({ id, fields }: { id: number; fields: TeacherInput }) =>
      commitResumeDraft(id, fields),
    onSuccess: (t) => {
      toast.success(`已入库：${t.name}`)
      setOpen(false)
      setCurrent(null)
      queryClient.invalidateQueries({ queryKey: ['teachers'] })
      queryClient.invalidateQueries({ queryKey: ['resume-drafts'] })
    },
    onError: (e) => toast.error(errMsg(e, '入库失败')),
  })

  const deleteMutation = useMutation({
    mutationFn: (id: number) => deleteResumeDraft(id),
    onSuccess: () => {
      toast.success('已丢弃识别记录')
      queryClient.invalidateQueries({ queryKey: ['resume-drafts'] })
    },
    onError: (e) => toast.error(errMsg(e, '删除失败')),
  })

  const startRecognize = async (file: File) => {
    if (!isResumeFile(file)) {
      toast.error('仅支持 .docx 与 .pdf 文件')
      return
    }
    if (file.size > RESUME_MAX_BYTES) {
      toast.error('文件超过 10MB 上限')
      return
    }
    setBusy(true)
    try {
      const draft = await parseResume(file)
      setCurrent(draft)
      setOpen(true)
      queryClient.invalidateQueries({ queryKey: ['resume-drafts'] })
    } catch (e) {
      toast.error(errMsg(e, '识别失败，请稍后重试'))
    } finally {
      setBusy(false)
    }
  }

  const onPick = (files: FileList | null) => {
    const f = files?.[0]
    if (f) void startRecognize(f)
  }

  const reopen = async (id: number) => {
    setBusy(true)
    try {
      const draft = await fetchResumeDraft(id)
      setCurrent(draft)
      setOpen(true)
    } catch (e) {
      toast.error(errMsg(e, '读取识别记录失败'))
    } finally {
      setBusy(false)
    }
  }

  const notConfigured = llm !== undefined && !llm.configured

  return (
    <div className="space-y-6">
      <PageHeader
        title="简历识别"
        description="上传 .docx / .pdf 简历，自动抽取人员信息并预填表单，核对后一键入库。"
      />

      {notConfigured && (
        <div className="flex flex-wrap items-center justify-between gap-3 rounded-[var(--r-thumb)] border border-amber-500/50 bg-amber-500/10 px-4 py-3">
          <p className="text-[14px] text-foreground">
            尚未配置大模型，简历识别不可用。
          </p>
          <Button size="sm" asChild>
            <Link to="/settings/llm">去配置模型</Link>
          </Button>
        </div>
      )}

      <div
        onDragOver={(e) => {
          e.preventDefault()
          setDragOver(true)
        }}
        onDragLeave={() => setDragOver(false)}
        onDrop={(e) => {
          e.preventDefault()
          setDragOver(false)
          if (busy) return
          void onPick(e.dataTransfer.files)
        }}
        className={cn(
          'surface-card flex flex-col items-center justify-center gap-3 rounded-[var(--r-thumb)] border-2 border-dashed px-6 py-12 text-center transition-colors duration-200',
          dragOver ? 'border-[var(--accent)] bg-[rgba(0,113,227,0.06)]' : 'border-[var(--hairline)]',
        )}
      >
        <div className="grid size-12 place-items-center rounded-full bg-[var(--track)]">
          <UploadCloud className="size-6 text-[var(--text-2)]" strokeWidth={1.75} />
        </div>
        <div className="space-y-1">
          <p className="text-[15px] font-medium text-foreground">拖拽简历到此处，或点击选择文件</p>
          <p className="text-[13px] text-[var(--text-3)]">
            支持 .docx / .pdf · 单个文件不超过 10MB · 扫描件 PDF 需要服务器安装 poppler-utils
          </p>
        </div>
        <input
          ref={inputRef}
          type="file"
          accept=".docx,.pdf"
          className="hidden"
          disabled={busy || notConfigured}
          onChange={(e) => {
            void onPick(e.target.files)
            e.target.value = ''
          }}
        />
        <Button
          loading={busy}
          disabled={notConfigured}
          onClick={() => inputRef.current?.click()}
        >
          {busy ? '正在识别…（约 10-60 秒）' : '选择简历文件'}
        </Button>
      </div>

      <section className="space-y-3">
        <div className="flex items-center justify-between">
          <h2 className="text-[17px] font-semibold tracking-[-0.02em]">识别记录</h2>
          <span className="text-[13px] text-[var(--text-3)]">
            共 {drafts?.total ?? 0} 条，入库后自动清除
          </span>
        </div>

        {!drafts?.items.length ? (
          <div className="rounded-[var(--r-thumb)] border border-[var(--hairline)] px-4 py-8 text-center text-[14px] text-[var(--text-3)]">
            暂无识别记录
          </div>
        ) : (
          <ul className="divide-y divide-[var(--hairline)] rounded-[var(--r-thumb)] border border-[var(--hairline)]">
            {drafts.items.map((d) => {
              const warnCount = d.payload?.warnings?.length ?? 0
              return (
                <li
                  key={d.id}
                  className="flex flex-wrap items-center justify-between gap-3 px-4 py-3"
                >
                  <div className="flex min-w-0 items-center gap-3">
                    <span className="grid size-9 shrink-0 place-items-center rounded-[var(--r-chip)] bg-[var(--track)]">
                      <FileText className="size-[18px] text-[var(--text-2)]" strokeWidth={1.75} />
                    </span>
                    <div className="min-w-0">
                      <p className="truncate text-[14px] font-medium text-foreground">
                        {d.file_name}
                      </p>
                      <p className="text-[12px] text-[var(--text-3)]">
                        {resumeTypeLabel[d.file_type] ?? d.file_type} · {formatSize(d.file_size)}
                        {d.page_count > 0 && ` · ${d.page_count} 页`} · {d.created_at}
                        {warnCount > 0 && ` · ${warnCount} 处待核对`}
                      </p>
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    <Button
                      variant="outline"
                      size="sm"
                      disabled={busy}
                      onClick={() => void reopen(d.id)}
                    >
                      <RefreshCw className="size-4" strokeWidth={1.75} />
                      继续录入
                    </Button>
                    <DeleteConfirm
                      title="丢弃识别记录"
                      description={`确定丢弃「${d.file_name}」的识别结果吗？原始文件不会被保留。`}
                      trigger={
                        <Button variant="ghost" size="icon" aria-label="丢弃">
                          <Trash2 className="size-4" strokeWidth={1.75} />
                        </Button>
                      }
                      onConfirm={async () => {
                        await deleteMutation.mutateAsync(d.id)
                      }}
                    />
                  </div>
                </li>
              )
            })}
          </ul>
        )}
      </section>

      <TeacherFormDialog
        open={open}
        onOpenChange={(v) => {
          setOpen(v)
          if (!v) setCurrent(null)
        }}
        draft={current ? toTeacher(current) : null}
        subjects={subjects ?? []}
        educations={educations ?? []}
        submitting={commitMutation.isPending}
        fieldMeta={current?.payload?.meta}
        warnings={current?.payload?.warnings}
        extra={
          current?.raw_text ? (
            <details className="rounded-[var(--r-thumb)] bg-[var(--track)] px-3 py-2 text-[13px] text-[var(--text-2)]">
              <summary className="cursor-pointer select-none">识别原文（用于核对）</summary>
              <pre className="mt-2 max-h-56 overflow-auto whitespace-pre-wrap font-sans">
                {current.raw_text}
              </pre>
            </details>
          ) : null
        }
        onSubmit={async (fields) => {
          if (!current) return
          await commitMutation.mutateAsync({ id: current.id, fields })
        }}
      />
    </div>
  )
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
