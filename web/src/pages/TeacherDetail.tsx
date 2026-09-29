import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useNavigate, useParams } from 'react-router-dom'
import { useState, type ReactNode } from 'react'
import { toast } from 'sonner'
import { Pencil, Trash2 } from 'lucide-react'
import {
  deleteTeacher,
  fetchTeacher,
  updateTeacher,
  ageOrDash,
  genderLabel,
  type TeacherInput,
} from '@/api/teachers'
import { fetchDictionaries } from '@/api/dicts'
import PageHeader from '@/components/PageHeader'
import TeacherFormDialog from '@/components/teachers/TeacherFormDialog'
import DeleteConfirm from '@/components/DeleteConfirm'
import PageSpinner from '@/components/PageSpinner'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { categoryTint, cn, errMsg } from '@/lib/utils'

function Field({ label, value }: { label: string; value: ReactNode }) {
  return (
    <div className="flex gap-3 py-2">
      <span className="w-24 shrink-0 text-[14px] text-[var(--text-3)]">{label}</span>
      <span className="min-w-0 flex-1 break-words text-[14px] text-foreground">{value}</span>
    </div>
  )
}

export default function TeacherDetail() {
  const { id } = useParams()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [editing, setEditing] = useState(false)

  const teacherId = Number(id)

  const { data: teacher, isLoading, error } = useQuery({
    queryKey: ['teachers', teacherId],
    queryFn: () => fetchTeacher(teacherId),
    enabled: Number.isInteger(teacherId) && teacherId > 0,
  })

  const { data: subjects = [] } = useQuery({
    queryKey: ['dicts', 'subject'],
    queryFn: () => fetchDictionaries('subject'),
  })
  const { data: educations = [] } = useQuery({
    queryKey: ['dicts', 'education'],
    queryFn: () => fetchDictionaries('education'),
  })

  const remove = useMutation({
    mutationFn: () => deleteTeacher(teacherId),
    onSuccess: () => {
      toast.success('已删除')
      queryClient.invalidateQueries({ queryKey: ['teachers'] })
      navigate('/', { replace: true })
    },
    onError: (e) => toast.error(errMsg(e, '删除失败')),
  })

  const update = useMutation({
    mutationFn: (input: TeacherInput) => updateTeacher(teacherId, input),
    onSuccess: (t) => {
      toast.success('已保存')
      queryClient.setQueryData(['teachers', teacherId], t)
      queryClient.invalidateQueries({ queryKey: ['teachers'] })
      setEditing(false)
    },
    onError: (e) => toast.error(errMsg(e, '保存失败')),
  })

  if (isLoading) return <PageSpinner />
  if (error || !teacher) {
    return (
      <div className="space-y-4 py-8 text-center">
        <p className="text-[15px] text-[var(--text-2)]">未找到该人员记录，可能已被删除</p>
        <Button variant="outline" onClick={() => navigate('/')}>
          返回列表
        </Button>
      </div>
    )
  }

  const subjectColor = subjects.find((d) => d.name === teacher.subject)?.color ?? ''
  const tint = categoryTint(subjectColor)

  return (
    <div className="space-y-5">
      <PageHeader
        onBack={() => navigate(-1)}
        title={teacher.name}
        description="人员档案详情"
        action={
          <>
            <Button variant="outline" onClick={() => setEditing(true)}>
              <Pencil /> 编辑
            </Button>
            <DeleteConfirm
              title="确认删除"
              description={`确定要删除「${teacher.name}」吗？删除后无法恢复。`}
              trigger={
                <Button
                  variant="outline"
                  className="text-destructive hover:bg-[rgba(215,0,21,0.06)] hover:text-destructive"
                >
                  <Trash2 /> 删除
                </Button>
              }
              onConfirm={async () => {
                await remove.mutateAsync()
              }}
            />
          </>
        }
      />

      <Card>
        <CardHeader>
          <CardTitle>基本信息</CardTitle>
        </CardHeader>
        <CardContent className="divide-y divide-[var(--hairline)]">
          <Field label="姓名" value={teacher.name} />
          <Field label="性别" value={genderLabel[teacher.gender] ?? '—'} />
          <Field label="年龄" value={<span className="tabular-nums">{ageOrDash(teacher.age)}</span>} />
          <Field
            label="学科"
            value={
              teacher.subject ? (
                <span
                  className={cn(
                    'inline-flex w-fit items-center rounded-full px-2.5 py-0.5 text-[12px] font-medium',
                    tint ? 'cat-pill' : 'bg-[var(--track)] text-[var(--text-2)]',
                  )}
                  style={tint ?? undefined}
                >
                  {teacher.subject}
                </span>
              ) : (
                '—'
              )
            }
          />
          <Field
            label="教师资格证"
            value={
              <span className={teacher.has_cert === 1 ? 'text-[var(--accent-link)]' : 'text-[var(--text-3)]'}>
                {teacher.has_cert === 1 ? '有' : '无'}
              </span>
            }
          />
          <Field
            label="联系电话"
            value={
              teacher.phone ? (
                <span className="font-mono tabular-nums">{teacher.phone}</span>
              ) : (
                '—'
              )
            }
          />
          <Field label="学历" value={teacher.education || '—'} />
          <Field label="毕业院校" value={teacher.university || '—'} />
          <Field label="专业" value={teacher.major || '—'} />
          <Field label="备注" value={teacher.remark || '—'} />
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>时间</CardTitle>
        </CardHeader>
        <CardContent className="divide-y divide-[var(--hairline)]">
          <Field label="录入时间" value={<span className="tabular-nums">{teacher.created_at}</span>} />
          <Field label="更新时间" value={<span className="tabular-nums">{teacher.updated_at}</span>} />
        </CardContent>
      </Card>

      <TeacherFormDialog
        open={editing}
        onOpenChange={setEditing}
        editing={teacher}
        subjects={subjects}
        educations={educations}
        onSubmit={async (input) => {
          await update.mutateAsync(input)
        }}
        submitting={update.isPending}
      />
    </div>
  )
}
