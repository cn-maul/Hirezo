<script setup lang="ts">
import { computed, ref } from 'vue'
import { Plus, Pencil, Trash2 } from 'lucide-vue-next'
import {
  createDictionary,
  deleteDictionary,
  fetchDictionaries,
  updateDictionary,
  type DictKind,
  type Dictionary,
} from '@/api/dicts'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import Label from '@/components/ui/Label.vue'
import Switch from '@/components/ui/Switch.vue'
import Dialog from '@/components/ui/Dialog.vue'
import DialogHeader from '@/components/ui/DialogHeader.vue'
import DialogTitle from '@/components/ui/DialogTitle.vue'
import DialogFooter from '@/components/ui/DialogFooter.vue'
import DataTable, { type Column } from '@/components/DataTable.vue'
import DeleteConfirm from '@/components/DeleteConfirm.vue'
import { useQuery, useMutation, invalidateQueries } from '@/composables/useQuery'
import { useForm } from '@/composables/useForm'
import { useToast } from '@/composables/useToast'
import { cn, categoryTint, errMsg } from '@/lib/utils'
import { validateDict, type DictFormValues } from '@/lib/validation'

const KIND_LABEL: Record<DictKind, string> = { subject: '学科', education: '学历' }

const tabs: { kind: DictKind; label: string }[] = [
  { kind: 'subject', label: '学科' },
  { kind: 'education', label: '学历' },
]

const emptyDict: DictFormValues = { name: '', color: '#0071e3', sort: 0, enabled: true }

const { success, error } = useToast()
const kind = ref<DictKind>('subject')
const modal = ref<{ open: boolean; item?: Dictionary }>({ open: false })

const { data: items, isLoading } = useQuery<Dictionary[]>(['dicts', kind.value], () =>
  fetchDictionaries(kind.value),
)

const invalidate = () => {
  invalidateQueries(['dicts', kind.value])
  invalidateQueries(['teachers'])
}

const createMutation = useMutation((p: { name: string; color: string; sort: number; enabled: number }) =>
  createDictionary(kind.value, p),
)
const updateMutation = useMutation(({ id, data }: { id: number; data: Partial<Dictionary> }) =>
  updateDictionary(kind.value, id, data),
)
const deleteMutation = useMutation((id: number) => deleteDictionary(kind.value, id))
const updatePending = updateMutation.isPending
const formSubmitting = computed(() => createMutation.isPending.value || updateMutation.isPending.value)

const { values, errors, set, reset, submit } = useForm<DictFormValues>(emptyDict, validateDict)

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
  modal.value = { open: true, item }
}

const onToggle = async (r: Dictionary, checked: boolean) => {
  try {
    await updateMutation.mutate({
      id: r.id,
      data: { name: r.name, color: r.color, sort: r.sort, enabled: checked ? 1 : 0 },
    })
    invalidate()
    success('已更新')
  } catch (e) {
    error(errMsg(e, '更新失败'))
  }
}

const onFinish = async (v: DictFormValues) => {
  const payload = { name: v.name.trim(), color: v.color, sort: v.sort, enabled: v.enabled ? 1 : 0 }
  try {
    if (modal.value.item) {
      await updateMutation.mutate({ id: modal.value.item.id, data: payload })
      success('已更新')
    } else {
      await createMutation.mutate(payload)
      success('已创建')
    }
    invalidate()
    modal.value = { open: false }
  } catch (e) {
    error(errMsg(e, modal.value.item ? '更新失败' : '创建失败'))
  }
}

// 预绑定 submit 处理器（避免 @submit="submit(onFinish)" 被 Vue 编译器误判为 withModifiers）
const onSubmit = submit(onFinish)

const onDelete = async (id: number) => {
  try {
    await deleteMutation.mutate(id)
    invalidate()
    success('已删除')
  } catch (e) {
    error(errMsg(e, '删除失败'))
  }
}

const columns: Column[] = [
  { title: '名称', key: 'name' },
  { title: '颜色', key: 'color', width: 120 },
  { title: '排序', key: 'sort', width: 80 },
  { title: '启用', key: 'enabled', width: 100 },
  { title: '操作', key: 'action', width: 110 },
]
</script>

<template>
  <div class="dicts">
    <div class="dicts__toolbar">
      <div class="dicts__tabs">
        <button
          v-for="t in tabs"
          :key="t.kind"
          type="button"
          :class="cn('dicts__tab', kind === t.kind ? 'dicts__tab--active' : '')"
          @click="kind = t.kind"
        >
          {{ t.label }}
        </button>
      </div>
      <Button @click="openModal()">
        <Plus :size="18" /> 新增{{ KIND_LABEL[kind] }}
      </Button>
    </div>

    <p class="dicts__hint">
      {{ KIND_LABEL[kind] }}用于人员表单下拉；停用后不再出现在新建选项中，但已有记录不受影响。
      重命名会同步更新所有引用该值的人员记录。
    </p>

    <DataTable
      :columns="columns"
      :data-source="items ?? []"
      :row-key="(r) => r.id"
      :loading="isLoading"
      :empty="`还没有${KIND_LABEL[kind]}，点击右上角新增`"
    >
      <template #cell="{ column, record }">
        <template v-if="column.key === 'name'">
          <span
            :class="cn('cat-pill', !categoryTint(record.color) && 'dicts__neutral-pill')"
            :style="categoryTint(record.color) ?? undefined"
          >
            {{ record.name }}
          </span>
        </template>

        <template v-else-if="column.key === 'color'">
          <span class="dicts__color">
            <span class="dicts__swatch" :style="{ backgroundColor: record.color }" />
            <span class="dicts__color-code">{{ record.color }}</span>
          </span>
        </template>

        <template v-else-if="column.key === 'enabled'">
          <Switch
            :model-value="record.enabled === 1"
            :aria-label="`切换 ${record.name}`"
            :disabled="updatePending"
            @update:model-value="(c) => onToggle(record, c)"
          />
        </template>

        <template v-else-if="column.key === 'action'">
          <div class="dicts__actions">
            <Button variant="ghost" size="icon" aria-label="编辑" @click="openModal(record)">
              <Pencil :size="18" />
            </Button>
            <DeleteConfirm
              title="确认删除"
              :description="`确定删除${KIND_LABEL[kind]}「${record.name}」吗？若已有人员使用，需先解除引用。`"
              @confirm="onDelete(record.id)"
            >
              <Button
                variant="ghost"
                size="icon"
                aria-label="删除"
                class="dicts__danger-btn"
              >
                <Trash2 :size="18" />
              </Button>
            </DeleteConfirm>
          </div>
        </template>
      </template>
    </DataTable>

    <Dialog v-model:open="modal.open">
      <DialogHeader>
        <DialogTitle>
          {{ modal.item ? '编辑' : '新增' }}{{ KIND_LABEL[kind] }}
        </DialogTitle>
      </DialogHeader>
      <form class="dicts__form" @submit="onSubmit">
        <div class="dicts__field">
          <Label html-for="dict-name">名称</Label>
          <Input
            id="dict-name"
            :placeholder="kind === 'subject' ? '如：物理' : '如：硕士研究生'"
            v-model="values.name"
            :max-length="32"
          />
          <p v-if="errors.name" class="dicts__error">{{ errors.name }}</p>
        </div>
        <div class="dicts__grid">
          <div class="dicts__field">
            <Label html-for="dict-color">颜色</Label>
            <div class="dicts__color-input">
              <Input
                id="dict-color"
                type="color"
                class="dicts__color-picker"
                v-model="values.color"
              />
              <span class="dicts__color-code">{{ values.color }}</span>
            </div>
            <p v-if="errors.color" class="dicts__error">{{ errors.color }}</p>
          </div>
          <div class="dicts__field">
            <Label html-for="dict-sort">排序</Label>
            <Input
              id="dict-sort"
              type="number"
              :model-value="String(values.sort)"
              @update:model-value="(v) => set('sort', Number(v))"
            />
            <p class="dicts__hint">数字越小越靠前</p>
          </div>
        </div>
        <div class="dicts__enabled">
          <div>
            <Label html-for="dict-enabled">启用</Label>
            <p class="dicts__hint">停用后不再出现在人员表单下拉中</p>
          </div>
          <Switch id="dict-enabled" v-model="values.enabled" />
        </div>
        <DialogFooter>
          <Button type="button" variant="outline" @click="modal = { open: false }">取消</Button>
          <Button type="submit" :loading="formSubmitting">
            保存
          </Button>
        </DialogFooter>
      </form>
    </Dialog>
  </div>
</template>

<style scoped>
.dicts {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.dicts__toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
}

.dicts__tabs {
  display: flex;
  align-items: center;
  gap: 2px;
  border-radius: var(--r-pill);
  background: var(--track);
  padding: 4px;
}
.dicts__tab {
  border-radius: var(--r-pill);
  padding: 8px 16px;
  font-size: 14px;
  font-weight: 500;
  color: var(--text-2);
  transition:
    background-color 200ms var(--ease-out-quart),
    color 200ms var(--ease-out-quart),
    box-shadow 200ms var(--ease-out-quart);
}
.dicts__tab:hover {
  color: var(--text);
}
.dicts__tab--active {
  background: var(--surface);
  color: var(--text);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08), 0 2px 8px rgba(0, 0, 0, 0.06);
}

.dicts__hint {
  font-size: 13px;
  color: var(--text-3);
}

.dicts__neutral-pill {
  background: var(--track);
  color: var(--text-2);
}

.dicts__color {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}
.dicts__swatch {
  width: 16px;
  height: 16px;
  border-radius: 999px;
  box-shadow: 0 0 0 1px var(--hairline);
}
.dicts__color-code {
  font-family: var(--mono);
  font-size: 12px;
}

.dicts__actions {
  display: flex;
  align-items: center;
  gap: 2px;
}
.dicts__danger-btn {
  color: var(--danger);
}
.dicts__danger-btn:hover {
  background: rgba(215, 0, 21, 0.08);
  color: var(--danger);
}

.dicts__form {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}
.dicts__field {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}
.dicts__grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1rem;
}
.dicts__color-input {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}
.dicts__color-picker {
  width: 56px;
  height: 40px;
  padding: 4px;
}
.dicts__enabled {
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-radius: var(--r-thumb);
  background: var(--bg);
  padding: 12px;
}
.dicts__error {
  font-size: 13px;
  color: var(--danger);
}
</style>