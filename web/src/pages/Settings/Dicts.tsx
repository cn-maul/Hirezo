import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { Plus, Pencil, Trash2 } from 'lucide-react'
import {
  createDictionary,
  deleteDictionary,
  fetchDictionaries,
  updateDictionary,
  type DictKind,
  type Dictionary,
} from '../../api/dicts'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { DataTable, type Column } from '@/components/Table'
import DeleteConfirm from '@/components/DeleteConfirm'
import { useFormState, validateDict, type DictFormValues } from '@/lib/validation'
import { cn, categoryTint } from '@/lib/utils'

const KIND_LABEL: Record<DictKind, string> = { subject: '学科', education: '学历' }

const tabs: { kind: DictKind; label: string }[] = [
  { kind: 'subject', label: '学科' },
  { kind: 'education', label: '学历' },
]

const emptyDict: DictFormValues = { name: '', color: '#0071e3', sort: 0, enabled: true }

export default function Dicts() {
  const [kind, setKind] = useState<DictKind>('subject')
  const [modal, setModal] = useState<{ open: boolean; item?: Dictionary }>({ open: false })
  const queryClient = useQueryClient()

  const { data: items = [], isLoading } = useQuery({
    queryKey: ['dicts', kind],
    queryFn: () => fetchDictionaries(kind),
  })

  const invalidate = () => {
    queryClient.invalidateQueries({ queryKey: ['dicts', kind] })
    queryClient.invalidateQueries({ queryKey: ['teachers'] })
  }

  const createMutation = useMutation({
    mutationFn: (p: { name: string; color: string; sort: number; enabled: number }) =>
      createDictionary(kind, p),
    onSuccess: () => {
      invalidate()
      toast.success('已创建')
      setModal({ open: false })
    },
    onError: (e: any) => toast.error(e.message || '创建失败'),
  })

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: number; data: Partial<Dictionary> }) =>
      updateDictionary(kind, id, data),
    onSuccess: () => {
      invalidate()
      toast.success('已更新')
      setModal({ open: false })
    },
    onError: (e: any) => toast.error(e.message || '更新失败'),
  })

  const deleteMutation = useMutation({
    mutationFn: (id: number) => deleteDictionary(kind, id),
    onSuccess: () => {
      invalidate()
      toast.success('已删除')
    },
    onError: (e: any) => toast.error(e.message || '删除失败'),
  })

  const { values, errors, set, reset, submit } = useFormState<DictFormValues>(emptyDict, validateDict)

  const openModal = (item?: Dictionary) => {
    if (item) {
      reset({
        name: item.name,
        color: item.color || '#0071e3',
        sort: item.sort,
        enabled: item.enabled === 1,
      })
    } else {
      reset({ ...emptyDict })
    }
    setModal({ open: true, item })
  }

  const onToggle = (r: Dictionary, checked: boolean) => {
    updateMutation.mutate({
      id: r.id,
      data: { name: r.name, color: r.color, sort: r.sort, enabled: checked ? 1 : 0 },
    })
  }

  const onFinish = (v: DictFormValues) => {
    const payload = { name: v.name.trim(), color: v.color, sort: v.sort, enabled: v.enabled ? 1 : 0 }
    if (modal.item) updateMutation.mutate({ id: modal.item.id, data: payload })
    else createMutation.mutate(payload)
  }

  const columns: Column<Dictionary>[] = [
    {
      title: '名称',
      key: 'name',
      render: (r) => {
        const tint = categoryTint(r.color)
        return (
          <span
            className={cn(
              'inline-flex w-fit items-center rounded-full px-2.5 py-0.5 text-[12px] font-medium',
              tint ? 'cat-pill' : 'bg-[var(--track)] text-[var(--text-2)]',
            )}
            style={tint ?? undefined}
          >
            {r.name}
          </span>
        )
      },
    },
    {
      title: '颜色',
      key: 'color',
      width: 120,
      render: (r) => (
        <span className="flex items-center gap-2">
          <span className="size-4 rounded-full ring-1 ring-[var(--hairline)]" style={{ backgroundColor: r.color }} />
          <span className="font-mono text-[12px]">{r.color}</span>
        </span>
      ),
    },
    { title: '排序', key: 'sort', width: 80 },
    {
      title: '启用',
      key: 'enabled',
      width: 100,
      render: (r) => (
        <Switch
          checked={r.enabled === 1}
          onCheckedChange={(c) => onToggle(r, c)}
          disabled={updateMutation.isPending}
          aria-label={`切换 ${r.name}`}
        />
      ),
    },
    {
      title: '操作',
      key: 'action',
      width: 110,
      render: (r) => (
        <div className="flex items-center gap-0.5">
          <Button variant="ghost" size="icon" onClick={() => openModal(r)} aria-label="编辑">
            <Pencil />
          </Button>
          <DeleteConfirm
            title="确认删除"
            description={`确定删除${KIND_LABEL[kind]}「${r.name}」吗？若已有人员使用，需先解除引用。`}
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
  ]

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-center gap-0.5 rounded-[var(--r-pill)] bg-[var(--track)] p-1">
          {tabs.map((t) => (
            <button
              key={t.kind}
              type="button"
              onClick={() => setKind(t.kind)}
              className={cn(
                'rounded-[var(--r-pill)] px-4 py-2 text-[14px] font-medium transition-[background-color,color,box-shadow] duration-200 ease-[var(--ease-out-quart)]',
                kind === t.kind
                  ? 'bg-surface text-foreground shadow-[0_1px_3px_rgba(0,0,0,0.08),0_2px_8px_rgba(0,0,0,0.06)]'
                  : 'text-[var(--text-2)] hover:text-foreground',
              )}
            >
              {t.label}
            </button>
          ))}
        </div>
        <Button onClick={() => openModal()}>
          <Plus /> 新增{KIND_LABEL[kind]}
        </Button>
      </div>

      <p className="text-[13px] text-[var(--text-3)]">
        {KIND_LABEL[kind]}用于人员表单下拉；停用后不再出现在新建选项中，但已有记录不受影响。
        重命名会同步更新所有引用该值的人员记录。
      </p>

      <DataTable<Dictionary>
        columns={columns}
        dataSource={items}
        rowKey={(r) => r.id}
        loading={isLoading}
        empty={`还没有${KIND_LABEL[kind]}，点击右上角新增`}
      />

      <Dialog open={modal.open} onOpenChange={(open) => setModal((m) => ({ ...m, open }))}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              {modal.item ? '编辑' : '新增'}
              {KIND_LABEL[kind]}
            </DialogTitle>
          </DialogHeader>
          <form onSubmit={submit(onFinish)} className="space-y-3">
            <div className="space-y-2">
              <Label htmlFor="dict-name">名称</Label>
              <Input
                id="dict-name"
                placeholder={kind === 'subject' ? '如：物理' : '如：硕士研究生'}
                value={values.name}
                onChange={(e) => set('name', e.target.value)}
                maxLength={32}
              />
              {errors.name && <p className="text-[13px] text-destructive">{errors.name}</p>}
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="dict-color">颜色</Label>
                <div className="flex items-center gap-2">
                  <Input
                    id="dict-color"
                    type="color"
                    className="size-10 w-14 p-1"
                    value={values.color}
                    onChange={(e) => set('color', e.target.value)}
                  />
                  <span className="font-mono text-[12px] text-[var(--text-3)]">{values.color}</span>
                </div>
                {errors.color && <p className="text-[13px] text-destructive">{errors.color}</p>}
              </div>
              <div className="space-y-2">
                <Label htmlFor="dict-sort">排序</Label>
                <Input
                  id="dict-sort"
                  type="number"
                  value={values.sort}
                  onChange={(e) => set('sort', Number(e.target.value))}
                />
                <p className="text-[12px] text-[var(--text-3)]">数字越小越靠前</p>
              </div>
            </div>
            <div className="flex items-center justify-between rounded-[var(--r-thumb)] bg-background p-3">
              <div>
                <Label htmlFor="dict-enabled">启用</Label>
                <p className="text-[12px] text-[var(--text-3)]">停用后不再出现在人员表单下拉中</p>
              </div>
              <Switch
                id="dict-enabled"
                checked={values.enabled}
                onCheckedChange={(c) => set('enabled', c)}
              />
            </div>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setModal({ open: false })}>
                取消
              </Button>
              <Button type="submit" loading={createMutation.isPending || updateMutation.isPending}>
                保存
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  )
}
