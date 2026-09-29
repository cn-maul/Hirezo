import { useEffect } from 'react'
import type { ReactNode } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  useFormState,
  validateTeacher,
  type TeacherFormValues,
} from '@/lib/validation'
import { cn } from '@/lib/utils'
import type { Dictionary } from '@/api/dicts'
import type { Teacher, TeacherInput } from '@/api/teachers'
import type { ResumeFieldMeta, ResumeWarning } from '@/api/resume'

const ALL = '__all__'

const blank: TeacherFormValues = {
  name: '',
  gender: 'male',
  age: '',
  subject: ALL,
  has_cert: false,
  phone: '',
  education: ALL,
  university: '',
  major: '',
  remark: '',
}

function toFormValues(t?: Teacher): TeacherFormValues {
  if (!t) return { ...blank }
  return {
    name: t.name,
    gender: t.gender,
    age: t.age > 0 ? String(t.age) : '',
    subject: t.subject || ALL,
    has_cert: t.has_cert === 1,
    phone: t.phone,
    education: t.education || ALL,
    university: t.university,
    major: t.major,
    remark: t.remark,
  }
}

// 简历识别回填：识别出的问题字段加色框提示人工复核
function fieldWrap(meta?: ResumeFieldMeta, pad = true) {
  if (!meta || meta.level === 'ok') return undefined
  const tint =
    meta.level === 'error' ? 'border-destructive/50 bg-destructive/5' : 'border-amber-500/50 bg-amber-500/5'
  return pad
    ? `rounded-[var(--r-thumb)] border p-2 ${tint}`
    : `rounded-[var(--r-thumb)] border ${tint}`
}

// 证据引文条：让人一眼看出模型是从哪句话抽出来的
function Evidence({ meta }: { meta?: ResumeFieldMeta }) {
  if (!meta) return null
  const evidence = meta.evidence?.trim()
  if (!evidence) return null
  return (
    <p className="text-[12px] leading-snug text-[var(--text-3)]">
      识别依据：{evidence}
      <span className="ml-1.5 tabular-nums">（{Math.round((meta.confidence || 0) * 100)}%）</span>
    </p>
  )
}

function WarningList({ warnings }: { warnings?: ResumeWarning[] }) {
  if (!warnings?.length) return null
  return (
    <div className="rounded-[var(--r-thumb)] border border-amber-500/50 bg-amber-500/10 p-3">
      <p className="text-[13px] font-medium text-foreground">
        识别到 {warnings.length} 处需要核对的内容
      </p>
      <ul className="mt-1.5 list-disc space-y-1 pl-5 text-[13px] text-[var(--text-2)]">
        {warnings.map((w) => (
          <li key={`${w.field}-${w.message}`}>{w.message}</li>
        ))}
      </ul>
    </div>
  )
}

// 字典值可能被停用但仍是某条记录的既有值：下拉只列启用项，
// 校验则放行全量（与后端 validDictValue 口径一致）。
function options(dict: Dictionary[]) {
  return {
    enabled: dict.filter((d) => d.enabled === 1).map((d) => d.name),
    all: dict.map((d) => d.name),
  }
}

function localErrors(v: TeacherFormValues, dict: { subjects: Dictionary[]; educations: Dictionary[] }) {
  const e = validateTeacher(v)
  const s = options(dict.subjects)
  const ed = options(dict.educations)
  if (v.subject !== ALL && !s.all.includes(v.subject)) e.subject = '学科不存在或已失效'
  if (v.education !== ALL && !ed.all.includes(v.education)) e.education = '学历不存在或已失效'
  return e
}

export default function TeacherFormDialog({
  open,
  onOpenChange,
  editing,
  draft,
  subjects,
  educations,
  onSubmit,
  submitting,
  fieldMeta,
  warnings,
  extra,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  /** 编辑既有记录 */
  editing?: Teacher | null
  /** 简历识别回填（与 editing 互斥，优先级低于 editing） */
  draft?: Teacher | null
  subjects: Dictionary[]
  educations: Dictionary[]
  onSubmit: (values: TeacherInput) => Promise<void>
  submitting: boolean
  /** 识别结果的字段元信息（证据 / 置信度 / 高亮等级） */
  fieldMeta?: Record<string, ResumeFieldMeta>
  /** 闸门产生的告警汇总 */
  warnings?: ResumeWarning[]
  /** 表单主体下方的附加内容（如识别原文） */
  extra?: ReactNode
}) {
  const { values, errors, set, reset, submit } = useFormState<TeacherFormValues>(blank, (v) =>
    localErrors(v, { subjects, educations }),
  )

  const source = editing ?? draft ?? null
  const isResume = !editing && !!draft

  // 打开时灌入待编辑数据；关闭后清空，避免下次打开残留
  useEffect(() => {
    if (open) reset(toFormValues(source ?? undefined))
  }, [open, source, reset])

  const s = options(subjects)
  const ed = options(educations)

  const onConfirm = async (v: TeacherFormValues) => {
    const age = v.age.trim()
    const input: TeacherInput = {
      name: v.name.trim(),
      gender: v.gender,
      age: age === '' ? 0 : Number(age),
      subject: v.subject === ALL ? '' : v.subject,
      has_cert: v.has_cert ? 1 : 0,
      phone: v.phone.trim(),
      education: v.education === ALL ? '' : v.education,
      university: v.university.trim(),
      major: v.major.trim(),
      remark: v.remark.trim(),
    }
    await onSubmit(input)
  }

  const title = editing ? '编辑人员' : isResume ? '识别结果 · 请核对' : '新增人员'
  const submitLabel = editing ? '保存' : isResume ? '确认入库' : '保存'
  const m = fieldMeta

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[88vh] overflow-y-auto sm:max-w-[640px]">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
        </DialogHeader>
        <form onSubmit={submit(onConfirm)} className="space-y-3">
          <WarningList warnings={warnings} />

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div className={cn('space-y-2', fieldWrap(m?.name))}>
              <Label htmlFor="t-name">姓名 *</Label>
              <Input
                id="t-name"
                placeholder="请输入姓名"
                value={values.name}
                onChange={(e) => set('name', e.target.value)}
              />
              <Evidence meta={m?.name} />
              {errors.name && <p className="text-[13px] text-destructive">{errors.name}</p>}
            </div>

            <div className={cn('space-y-2', fieldWrap(m?.gender))}>
              <Label htmlFor="t-gender">性别</Label>
              <Select value={values.gender} onValueChange={(v) => set('gender', v as 'male' | 'female')}>
                <SelectTrigger id="t-gender"><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="male">男</SelectItem>
                  <SelectItem value="female">女</SelectItem>
                </SelectContent>
              </Select>
              <Evidence meta={m?.gender} />
              {errors.gender && <p className="text-[13px] text-destructive">{errors.gender}</p>}
            </div>

            <div className={cn('space-y-2', fieldWrap(m?.age))}>
              <Label htmlFor="t-age">年龄</Label>
              <Input
                id="t-age"
                type="number"
                min={18}
                max={100}
                placeholder="18-100，留空表示未填"
                value={values.age}
                onChange={(e) => set('age', e.target.value)}
              />
              <Evidence meta={m?.age} />
              {errors.age && <p className="text-[13px] text-destructive">{errors.age}</p>}
            </div>

            <div className={cn('space-y-2', fieldWrap(m?.phone))}>
              <Label htmlFor="t-phone">联系电话</Label>
              <Input
                id="t-phone"
                placeholder="11 位手机号，可留空"
                value={values.phone}
                onChange={(e) => set('phone', e.target.value)}
              />
              <Evidence meta={m?.phone} />
              {errors.phone && <p className="text-[13px] text-destructive">{errors.phone}</p>}
            </div>

            <div className={cn('space-y-2', fieldWrap(m?.subject))}>
              <Label htmlFor="t-subject">学科</Label>
              <Select value={values.subject} onValueChange={(v) => set('subject', v)}>
                <SelectTrigger id="t-subject"><SelectValue placeholder="请选择学科" /></SelectTrigger>
                <SelectContent>
                  <SelectItem value={ALL}>未分配</SelectItem>
                  {s.enabled.map((n) => <SelectItem key={n} value={n}>{n}</SelectItem>)}
                  {/* 既有停用值：保留在下拉中以便编辑历史记录 */}
                  {values.subject !== ALL && !s.enabled.includes(values.subject) && (
                    <SelectItem value={values.subject}>{values.subject}（已停用）</SelectItem>
                  )}
                </SelectContent>
              </Select>
              <Evidence meta={m?.subject} />
              {errors.subject && <p className="text-[13px] text-destructive">{errors.subject}</p>}
            </div>

            <div className={cn('space-y-2', fieldWrap(m?.education))}>
              <Label htmlFor="t-education">学历</Label>
              <Select value={values.education} onValueChange={(v) => set('education', v)}>
                <SelectTrigger id="t-education"><SelectValue placeholder="请选择学历" /></SelectTrigger>
                <SelectContent>
                  <SelectItem value={ALL}>未填写</SelectItem>
                  {ed.enabled.map((n) => <SelectItem key={n} value={n}>{n}</SelectItem>)}
                  {values.education !== ALL && !ed.enabled.includes(values.education) && (
                    <SelectItem value={values.education}>{values.education}（已停用）</SelectItem>
                  )}
                </SelectContent>
              </Select>
              <Evidence meta={m?.education} />
              {errors.education && <p className="text-[13px] text-destructive">{errors.education}</p>}
            </div>

            <div className={cn('space-y-2', fieldWrap(m?.university))}>
              <Label htmlFor="t-university">毕业院校</Label>
              <Input
                id="t-university"
                placeholder="如：北京师范大学"
                value={values.university}
                onChange={(e) => set('university', e.target.value)}
              />
              <Evidence meta={m?.university} />
              {errors.university && <p className="text-[13px] text-destructive">{errors.university}</p>}
            </div>

            <div className={cn('space-y-2', fieldWrap(m?.major))}>
              <Label htmlFor="t-major">专业</Label>
              <Input
                id="t-major"
                placeholder="如：汉语言文学"
                value={values.major}
                onChange={(e) => set('major', e.target.value)}
              />
              <Evidence meta={m?.major} />
              {errors.major && <p className="text-[13px] text-destructive">{errors.major}</p>}
            </div>
          </div>

          <div
            className={cn(
              'flex items-center justify-between rounded-[var(--r-thumb)] bg-background p-3',
              fieldWrap(m?.has_cert, false),
            )}
          >
            <div>
              <Label htmlFor="t-cert">教师资格证</Label>
              <p className="text-[12px] text-[var(--text-3)]">开启表示已取得教师资格证</p>
              <Evidence meta={m?.has_cert} />
            </div>
            <Switch
              id="t-cert"
              checked={values.has_cert}
              onCheckedChange={(c) => set('has_cert', c)}
            />
          </div>

          <div className={cn('space-y-2', fieldWrap(m?.remark))}>
            <Label htmlFor="t-remark">备注</Label>
            <Textarea
              id="t-remark"
              rows={3}
              placeholder="可留空，最多 500 字"
              value={values.remark}
              onChange={(e) => set('remark', e.target.value)}
            />
            <Evidence meta={m?.remark} />
            {errors.remark && <p className="text-[13px] text-destructive">{errors.remark}</p>}
          </div>

          {extra}

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              取消
            </Button>
            <Button type="submit" loading={submitting}>{submitLabel}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
