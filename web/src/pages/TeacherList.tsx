import { useEffect, useMemo, useRef, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { toast } from 'sonner'
import { Pencil, Plus, Trash2 } from 'lucide-react'
import {
  batchDeleteTeachers,
  createTeacher,
  deleteTeacher,
  exportXlsx,
  fetchTeacher,
  fetchTeachers,
  updateTeacher,
  ageOrDash,
  genderLabel,
  type ListParams,
  type Teacher,
  type TeacherInput,
} from '../api/teachers'
import { fetchDictionaries, type Dictionary } from '../api/dicts'
import TeacherFilters, { type TeacherFilterValues } from '@/components/teachers/TeacherFilters'
import TeacherFormDialog from '@/components/teachers/TeacherFormDialog'
import { Button } from '@/components/ui/button'
import { DataTable, PaginationBar, type Column } from '@/components/Table'
import PageHeader from '@/components/PageHeader'
import DeleteConfirm from '@/components/DeleteConfirm'
import { cn, categoryTint } from '@/lib/utils'

const PAGE_SIZE = 20

export default function TeacherList() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [searchParams, setSearchParams] = useSearchParams()

  const urlKeyword = searchParams.get('keyword') ?? ''
  const [keyword, setKeyword] = useState(urlKeyword)
  const [debounced, setDebounced] = useState(urlKeyword)
  const [subject, setSubject] = useState(searchParams.get('subject') ?? '')
  const [gender, setGender] = useState<'' | 'male' | 'female'>(
    (searchParams.get('gender') as '' | 'male' | 'female') ?? '',
  )
  const [education, setEducation] = useState(searchParams.get('education') ?? '')
  const [hasCert, setHasCert] = useState<'' | '0' | '1'>(searchParams.get('hasCert') as '' | '0' | '1' ?? '')
  const [selected, setSelected] = useState<(string | number)[]>([])
  const [page, setPage] = useState(1)
  const [exporting, setExporting] = useState(false)

  // 新增 / 编辑弹窗：id 为 undefined 表示新增
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editingId, setEditingId] = useState<number | null>(null)

  const syncedKeywordRef = useRef<string | null>(null)

  useEffect(() => {
    if (urlKeyword === syncedKeywordRef.current) return
    setKeyword(urlKeyword)
    setPage(1)
  }, [urlKeyword])

  useEffect(() => {
    const t = setTimeout(() => setDebounced(keyword), 300)
    return () => clearTimeout(t)
  }, [keyword])

  // 关键字与筛选同步到 URL，刷新/分享后条件不丢
  useEffect(() => {
    syncedKeywordRef.current = debounced
    setSearchParams(
      (prev) => {
        const next = new URLSearchParams(prev)
        const setOrDel = (k: string, v: string) => {
          if (v) next.set(k, v)
          else next.delete(k)
        }
        setOrDel('keyword', debounced.trim())
        setOrDel('subject', subject)
        setOrDel('gender', gender)
        setOrDel('education', education)
        setOrDel('hasCert', hasCert)
        return next
      },
      { replace: true },
    )
  }, [debounced, subject, gender, education, hasCert, setSearchParams])

  useEffect(() => {
    setSelected([])
  }, [debounced, subject, gender, education, hasCert, page])

  const { data: subjects = [] } = useQuery({
    queryKey: ['dicts', 'subject'],
    queryFn: () => fetchDictionaries('subject'),
  })
  const { data: educations = [] } = useQuery({
    queryKey: ['dicts', 'education'],
    queryFn: () => fetchDictionaries('education'),
  })

  const subjectColor = useMemo(() => {
    const m: Record<string, string> = {}
    for (const d of subjects) if (d.color) m[d.name] = d.color
    return m
  }, [subjects])

  const listParams = useMemo<ListParams>(
    () => ({
      keyword: debounced.trim() || undefined,
      subject: subject || undefined,
      gender: gender || undefined,
      education: education || undefined,
      hasCert: (hasCert === '' ? '' : Number(hasCert)) as '' | 0 | 1,
    }),
    [debounced, subject, gender, education, hasCert],
  )

  const { data, isLoading, refetch } = useQuery({
    queryKey: ['teachers', listParams, page],
    queryFn: () => fetchTeachers({ ...listParams, page, size: PAGE_SIZE }),
  })

  const { data: editing } = useQuery({
    queryKey: ['teachers', editingId],
    queryFn: () => fetchTeacher(editingId as number),
    enabled: editingId !== null,
  })

  const refresh = () => {
    refetch()
    queryClient.invalidateQueries({ queryKey: ['teachers'] })
  }

  const createMutation = useMutation({
    mutationFn: createTeacher,
    onSuccess: () => {
      toast.success('已创建')
      setDialogOpen(false)
      refresh()
    },
    onError: (e: any) => toast.error(e?.message ?? '创建失败'),
  })

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: number; data: TeacherInput }) => updateTeacher(id, data),
    onSuccess: () => {
      toast.success('已保存')
      setDialogOpen(false)
      setEditingId(null)
      refresh()
    },
    onError: (e: any) => toast.error(e?.message ?? '保存失败'),
  })

  const deleteMutation = useMutation({
    mutationFn: deleteTeacher,
    onSuccess: () => {
      toast.success('已删除')
      if ((data?.items.length ?? 0) === 1 && page > 1) setPage(page - 1)
      else refresh()
    },
    onError: (e: any) => toast.error(e?.message ?? '删除失败'),
  })

  const onExport = async () => {
    setExporting(true)
    try {
      await exportXlsx(listParams)
      toast.success('已导出 Excel')
    } catch (e: any) {
      toast.error(e?.message ?? '导出失败')
    } finally {
      setExporting(false)
    }
  }

  const openCreate = () => {
    setEditingId(null)
    setDialogOpen(true)
  }

  const openEdit = (id: number) => {
    setEditingId(id)
    setDialogOpen(true)
  }

  const columns: Column<Teacher>[] = useMemo(
    () => [
      {
        title: '姓名',
        key: 'name',
        width: 120,
        render: (r) => (
          <Link
            to={`/teachers/${r.id}`}
            className="font-medium text-[var(--accent-link)] hover:underline"
          >
            {r.name}
          </Link>
        ),
      },
      {
        title: '性别',
        key: 'gender',
        width: 70,
        render: (r) => <span className="text-[var(--text-2)]">{genderLabel[r.gender] ?? '—'}</span>,
      },
      {
        title: '年龄',
        key: 'age',
        width: 70,
        render: (r) => <span className="tabular-nums text-[var(--text-2)]">{ageOrDash(r.age)}</span>,
      },
      {
        title: '学科',
        key: 'subject',
        width: 110,
        render: (r) => {
          if (!r.subject) return <span className="text-[var(--text-4)]">—</span>
          const tint = categoryTint(subjectColor[r.subject])
          return (
            <span
              className={cn(
                'inline-flex w-fit items-center rounded-full px-2.5 py-0.5 text-[12px] font-medium',
                tint ? 'cat-pill' : 'bg-[var(--track)] text-[var(--text-2)]',
              )}
              style={tint ?? undefined}
            >
              {r.subject}
            </span>
          )
        },
      },
      {
        title: '资格证',
        key: 'has_cert',
        width: 80,
        render: (r) => (
          <span className={r.has_cert === 1 ? 'text-[var(--accent-link)]' : 'text-[var(--text-3)]'}>
            {r.has_cert === 1 ? '有' : '无'}
          </span>
        ),
      },
      {
        title: '联系电话',
        key: 'phone',
        width: 130,
        render: (r) =>
          r.phone ? (
            <span className="font-mono text-[13px] tabular-nums">{r.phone}</span>
          ) : (
            <span className="text-[var(--text-4)]">—</span>
          ),
      },
      {
        title: '学历',
        key: 'education',
        width: 90,
        render: (r) =>
          r.education || <span className="text-[var(--text-4)]">—</span>,
      },
      {
        title: '毕业院校',
        key: 'university',
        render: (r) => (
          <span className="block max-w-[200px] truncate text-[var(--text-2)]" title={r.university}>
            {r.university || '—'}
          </span>
        ),
      },
      {
        title: '专业',
        key: 'major',
        render: (r) => (
          <span className="block max-w-[160px] truncate text-[var(--text-2)]" title={r.major}>
            {r.major || '—'}
          </span>
        ),
      },
      {
        title: '录入时间',
        key: 'created_at',
        width: 150,
        render: (r) => <span className="text-[13px] tabular-nums text-[var(--text-2)]">{r.created_at}</span>,
      },
      {
        title: '操作',
        key: 'action',
        width: 120,
        render: (r) => (
          <div className="flex items-center gap-0.5">
            <Button variant="ghost" size="icon" onClick={() => navigate(`/teachers/${r.id}`)} aria-label="查看">
              查看
            </Button>
            <Button variant="ghost" size="icon" onClick={() => openEdit(r.id)} aria-label="编辑">
              <Pencil />
            </Button>
            <DeleteConfirm
              title="确认删除"
              description={`确定要删除「${r.name}」吗？删除后无法恢复。`}
              trigger={
                <Button
                  variant="ghost"
                  size="icon"
                  className="text-destructive hover:bg-[rgba(215,0,21,0.08)] hover:text-destructive"
                  aria-label="删除"
                >
                  <Trash2 />
                </Button>
              }
              onConfirm={async () => {
                await deleteMutation.mutateAsync(r.id)
              }}
            />
          </div>
        ),
      },
    ],
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [subjectColor, navigate, data, page],
  )

  const onFilterChange = (next: Partial<Omit<TeacherFilterValues, 'keyword'>>) => {
    if (next.subject !== undefined) setSubject(next.subject)
    if (next.gender !== undefined) setGender(next.gender)
    if (next.education !== undefined) setEducation(next.education)
    if (next.hasCert !== undefined) setHasCert(next.hasCert)
    setPage(1)
  }

  const onSubmitDialog = async (input: TeacherInput) => {
    if (editingId) await updateMutation.mutateAsync({ id: editingId, data: input })
    else await createMutation.mutateAsync(input)
  }

  return (
    <div className="space-y-5">
      <PageHeader
        title="人员管理"
        description="学校教师与工作人员档案"
        action={
          <Button onClick={openCreate}>
            <Plus /> 新增人员
          </Button>
        }
      />

      <TeacherFilters
        keyword={keyword}
        onKeywordChange={(v) => {
          setKeyword(v)
          setPage(1)
        }}
        subjects={subjects.filter((d: Dictionary) => d.enabled === 1).map((d: Dictionary) => d.name)}
        educations={educations.filter((d: Dictionary) => d.enabled === 1).map((d: Dictionary) => d.name)}
        values={{ subject, gender, education, hasCert }}
        onChange={onFilterChange}
        onExport={onExport}
        exporting={exporting}
      />

      {selected.length > 0 && (
        <div className="flex flex-wrap items-center gap-3 surface-card rounded-[var(--r-thumb)] px-4 py-2.5">
          <span className="text-[14px] tabular-nums text-[var(--text-2)]">已选 {selected.length} 条</span>
          <DeleteConfirm
            title="批量删除"
            description={`确定要删除选中的 ${selected.length} 条人员记录吗？删除后无法恢复。`}
            trigger={
              <Button
                variant="outline"
                size="sm"
                className="text-destructive hover:bg-[rgba(215,0,21,0.06)] hover:text-destructive"
              >
                <Trash2 /> 删除
              </Button>
            }
            onConfirm={async () => {
              try {
                const n = await batchDeleteTeachers(selected.map(Number))
                toast.success(`已删除 ${n} 条记录`)
                setSelected([])
                refresh()
              } catch (e: any) {
                toast.error(e?.message ?? '批量删除失败')
              }
            }}
          />
          <Button variant="ghost" size="sm" onClick={() => setSelected([])}>
            取消选择
          </Button>
        </div>
      )}

      <DataTable<Teacher>
        columns={columns}
        dataSource={data?.items ?? []}
        rowKey={(r) => r.id}
        loading={isLoading}
        empty="没有符合条件的人员"
        selectable
        selectedKeys={selected}
        onSelectedChange={setSelected}
      />

      <PaginationBar page={page} pageSize={PAGE_SIZE} total={data?.total ?? 0} onChange={setPage} />

      <TeacherFormDialog
        open={dialogOpen}
        onOpenChange={(open) => {
          setDialogOpen(open)
          if (!open) setEditingId(null)
        }}
        editing={editingId !== null ? editing ?? null : null}
        subjects={subjects}
        educations={educations}
        onSubmit={onSubmitDialog}
        submitting={createMutation.isPending || updateMutation.isPending}
      />
    </div>
  )
}
