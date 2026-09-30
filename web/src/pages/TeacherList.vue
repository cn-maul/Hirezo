<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import Pencil from 'lucide-vue-next/dist/esm/icons/pencil.js'
import Plus from 'lucide-vue-next/dist/esm/icons/plus.js'
import Trash2 from 'lucide-vue-next/dist/esm/icons/trash-2.js'
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
} from '@/api/teachers'
import { fetchDictionaries, type Dictionary } from '@/api/dicts'
import TeacherFilters, { type TeacherFilterValues } from '@/components/teachers/TeacherFilters.vue'
import TeacherFormDialog from '@/components/teachers/TeacherFormDialog.vue'
import Button from '@/components/ui/Button.vue'
import DataTable, { type Column } from '@/components/DataTable.vue'
import PaginationBar from '@/components/PaginationBar.vue'
import PageHeader from '@/components/PageHeader.vue'
import DeleteConfirm from '@/components/DeleteConfirm.vue'
import { useQuery, useMutation, invalidateQueries } from '@/composables/useQuery'
import { useToast } from '@/composables/useToast'
import { cn, categoryTint, errMsg } from '@/lib/utils'

const PAGE_SIZE = 20

const route = useRoute()
const router = useRouter()
const { success, error } = useToast()

const urlKeyword = typeof route.query.keyword === 'string' ? route.query.keyword : ''
const keyword = ref(urlKeyword)
const debounced = ref(urlKeyword)
const subject = ref(typeof route.query.subject === 'string' ? route.query.subject : '')
const gender = ref<'' | 'male' | 'female'>(
  (typeof route.query.gender === 'string' ? route.query.gender : '') as '' | 'male' | 'female',
)
const education = ref(typeof route.query.education === 'string' ? route.query.education : '')
const hasCert = ref<'' | '0' | '1'>(
  (typeof route.query.hasCert === 'string' ? route.query.hasCert : '') as '' | '0' | '1',
)
const selected = ref<(string | number)[]>([])
const page = ref(1)
const exporting = ref(false)

// 新增 / 编辑弹窗：id 为 undefined 表示新增
const dialogOpen = ref(false)
const editingId = ref<number | null>(null)

// 关键字防抖 300ms
let debounceTimer: ReturnType<typeof setTimeout> | null = null
watch(keyword, (v) => {
  if (debounceTimer) clearTimeout(debounceTimer)
  debounceTimer = setTimeout(() => {
    debounced.value = v
  }, 300)
})

// 关键字与筛选同步到 URL，刷新/分享后条件不丢
watch([debounced, subject, gender, education, hasCert], () => {
  const q: Record<string, string> = {}
  if (debounced.value.trim()) q.keyword = debounced.value.trim()
  if (subject.value) q.subject = subject.value
  if (gender.value) q.gender = gender.value
  if (education.value) q.education = education.value
  if (hasCert.value) q.hasCert = hasCert.value
  void router.replace({ query: q })
})

// 筛选变化清空多选
watch([debounced, subject, gender, education, hasCert, page], () => {
  selected.value = []
})

const { data: subjects } = useQuery<Dictionary[]>(['dicts', 'subject'], () =>
  fetchDictionaries('subject'),
)
const { data: educations } = useQuery<Dictionary[]>(['dicts', 'education'], () =>
  fetchDictionaries('education'),
)

const subjectColor = computed(() => {
  const m: Record<string, string> = {}
  for (const d of subjects.value ?? []) if (d.color) m[d.name] = d.color
  return m
})

const listParams = computed<ListParams>(() => ({
  keyword: debounced.value.trim() || undefined,
  subject: subject.value || undefined,
  gender: gender.value || undefined,
  education: education.value || undefined,
  hasCert: (hasCert.value === '' ? '' : Number(hasCert.value)) as '' | 0 | 1,
}))

const { data, isLoading, refetch } = useQuery(
  ['teachers', listParams.value, page.value],
  () => fetchTeachers({ ...listParams.value, page: page.value, size: PAGE_SIZE }),
)

const { data: editing } = useQuery<Teacher>(
  ['teachers', editingId.value],
  () => fetchTeacher(editingId.value as number),
  { enabled: computed(() => editingId.value !== null) },
)

const refresh = () => {
  void refetch()
  invalidateQueries(['teachers'])
}

const createMutation = useMutation(createTeacher)
const updateMutation = useMutation(({ id, data }: { id: number; data: TeacherInput }) =>
  updateTeacher(id, data),
)
const formSubmitting = computed(() => createMutation.isPending.value || updateMutation.isPending.value)
const deleteMutation = useMutation(deleteTeacher)

const onExport = async () => {
  exporting.value = true
  try {
    await exportXlsx(listParams.value)
    success('已导出 Excel')
  } catch (e) {
    error(errMsg(e, '导出失败'))
  } finally {
    exporting.value = false
  }
}

const openCreate = () => {
  editingId.value = null
  dialogOpen.value = true
}

const openEdit = (id: number) => {
  editingId.value = id
  dialogOpen.value = true
}

const columns: Column[] = [
  { title: '姓名', key: 'name', width: 120 },
  { title: '性别', key: 'gender', width: 70 },
  { title: '年龄', key: 'age', width: 70 },
  { title: '学科', key: 'subject', width: 110 },
  { title: '资格证', key: 'has_cert', width: 80 },
  { title: '联系电话', key: 'phone', width: 130 },
  { title: '学历', key: 'education', width: 90 },
  { title: '毕业院校', key: 'university' },
  { title: '专业', key: 'major' },
  { title: '录入时间', key: 'created_at', width: 150 },
  { title: '操作', key: 'action', width: 120 },
]

const onFilterChange = (next: Partial<Omit<TeacherFilterValues, 'keyword'>>) => {
  if (next.subject !== undefined) subject.value = next.subject
  if (next.gender !== undefined) gender.value = next.gender
  if (next.education !== undefined) education.value = next.education
  if (next.hasCert !== undefined) hasCert.value = next.hasCert
  page.value = 1
}

const onSubmitDialog = async (input: TeacherInput) => {
  if (editingId.value) {
    try {
      await updateMutation.mutate({ id: editingId.value, data: input })
      success('已保存')
      dialogOpen.value = false
      editingId.value = null
      refresh()
    } catch (e) {
      error(errMsg(e, '保存失败'))
    }
  } else {
    try {
      await createMutation.mutate(input)
      success('已创建')
      dialogOpen.value = false
      refresh()
    } catch (e) {
      error(errMsg(e, '创建失败'))
    }
  }
}

const onDelete = async (id: number) => {
  try {
    await deleteMutation.mutate(id)
    success('已删除')
    if ((data.value?.items.length ?? 0) === 1 && page.value > 1) page.value -= 1
    else refresh()
  } catch (e) {
    error(errMsg(e, '删除失败'))
  }
}

const onBatchDelete = async () => {
  try {
    const n = await batchDeleteTeachers(selected.value.map(Number))
    success(`已删除 ${n} 条记录`)
    selected.value = []
    refresh()
  } catch (e) {
    error(errMsg(e, '批量删除失败'))
  }
}
</script>

<template>
  <div class="teacher-list">
    <PageHeader title="人员管理" description="学校教师与工作人员档案">
      <template #action>
        <Button @click="openCreate"><Plus :size="18" /> 新增人员</Button>
      </template>
    </PageHeader>

    <TeacherFilters
      :keyword="keyword"
      :subjects="(subjects ?? []).filter((d) => d.enabled === 1).map((d) => d.name)"
      :educations="(educations ?? []).filter((d) => d.enabled === 1).map((d) => d.name)"
      :values="{ subject, gender, education, hasCert }"
      :exporting="exporting"
      @update:keyword="(v) => { keyword = v; page = 1 }"
      @change="onFilterChange"
      @export="onExport"
    />

    <div v-if="selected.length > 0" class="teacher-list__selection surface-card">
      <span class="teacher-list__selection-count">已选 {{ selected.length }} 条</span>
      <DeleteConfirm
        title="批量删除"
        :description="`确定要删除选中的 ${selected.length} 条人员记录吗？删除后无法恢复。`"
        @confirm="onBatchDelete"
      >
        <Button variant="outline" size="sm" class="teacher-list__danger-btn">
          <Trash2 :size="18" /> 删除
        </Button>
      </DeleteConfirm>
      <Button variant="ghost" size="sm" @click="selected = []">取消选择</Button>
    </div>

    <DataTable
      :columns="columns"
      :data-source="data?.items ?? []"
      :row-key="(r) => r.id"
      :loading="isLoading"
      empty="没有符合条件的人员"
      selectable
      v-model:selected-keys="selected"
    >
      <template #cell="{ column, record }">
        <template v-if="column.key === 'name'">
          <RouterLink :to="`/teachers/${record.id}`" class="teacher-list__name-link">
            {{ record.name }}
          </RouterLink>
        </template>

        <template v-else-if="column.key === 'gender'">
          <span class="teacher-list__meta">{{ genderLabel[record.gender] ?? '—' }}</span>
        </template>

        <template v-else-if="column.key === 'age'">
          <span class="teacher-list__meta">{{ ageOrDash(record.age) }}</span>
        </template>

        <template v-else-if="column.key === 'subject'">
          <template v-if="record.subject">
            <span
              :class="cn('cat-pill', !categoryTint(subjectColor[record.subject]) && 'teacher-list__neutral-pill')"
              :style="categoryTint(subjectColor[record.subject]) ?? undefined"
            >
              {{ record.subject }}
            </span>
          </template>
          <span v-else class="teacher-list__muted">—</span>
        </template>

        <template v-else-if="column.key === 'has_cert'">
          <span :class="record.has_cert === 1 ? 'teacher-list__cert-yes' : 'teacher-list__cert-no'">
            {{ record.has_cert === 1 ? '有' : '无' }}
          </span>
        </template>

        <template v-else-if="column.key === 'phone'">
          <span v-if="record.phone" class="teacher-list__phone">{{ record.phone }}</span>
          <span v-else class="teacher-list__muted">—</span>
        </template>

        <template v-else-if="column.key === 'education'">
          <template v-if="record.education">{{ record.education }}</template>
          <span v-else class="teacher-list__muted">—</span>
        </template>

        <template v-else-if="column.key === 'university'">
          <span v-if="record.university" class="teacher-list__truncate" :title="record.university">
            {{ record.university }}
          </span>
          <span v-else class="teacher-list__muted">—</span>
        </template>

        <template v-else-if="column.key === 'major'">
          <span v-if="record.major" class="teacher-list__truncate" :title="record.major">
            {{ record.major }}
          </span>
          <span v-else class="teacher-list__muted">—</span>
        </template>

        <template v-else-if="column.key === 'created_at'">
          <span class="teacher-list__meta">{{ record.created_at }}</span>
        </template>

        <template v-else-if="column.key === 'action'">
          <div class="teacher-list__actions">
            <Button variant="ghost" size="sm" @click="router.push(`/teachers/${record.id}`)">
              查看
            </Button>
            <Button variant="ghost" size="icon" aria-label="编辑" @click="openEdit(record.id)">
              <Pencil :size="18" />
            </Button>
            <DeleteConfirm
              title="确认删除"
              :description="`确定要删除「${record.name}」吗？删除后无法恢复。`"
              @confirm="onDelete(record.id)"
            >
              <Button
                variant="ghost"
                size="icon"
                aria-label="删除"
                class="teacher-list__danger-btn"
              >
                <Trash2 :size="18" />
              </Button>
            </DeleteConfirm>
          </div>
        </template>
      </template>
    </DataTable>

    <PaginationBar
      :page="page"
      :page-size="PAGE_SIZE"
      :total="data?.total ?? 0"
      @change="(p) => (page = p)"
    />

    <TeacherFormDialog
      v-model:open="dialogOpen"
      :editing="editingId !== null ? editing ?? null : null"
      :subjects="subjects ?? []"
      :educations="educations ?? []"
      :submitting="formSubmitting"
      @submit="onSubmitDialog"
      @update:open="(v) => { if (!v) editingId = null }"
    />
  </div>
</template>

<style scoped>
.teacher-list {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.teacher-list__selection {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.75rem;
  border-radius: var(--r-thumb);
  padding: 10px 16px;
}
.teacher-list__selection-count {
  font-size: 14px;
  font-variant-numeric: tabular-nums;
  color: var(--text-2);
}
.teacher-list__danger-btn {
  color: var(--danger);
}
.teacher-list__danger-btn:hover {
  background: rgba(215, 0, 21, 0.08);
  color: var(--danger);
}

.teacher-list__name-link {
  font-weight: 500;
  color: var(--accent-link);
}
.teacher-list__name-link:hover {
  text-decoration: underline;
}

.teacher-list__neutral-pill {
  background: var(--track);
  color: var(--text-2);
}

.teacher-list__muted {
  color: var(--text-4);
}
.teacher-list__cert-yes {
  color: var(--accent-link);
}
.teacher-list__cert-no {
  color: var(--text-3);
}
.teacher-list__phone {
  font-family: var(--mono);
  font-size: 13px;
  font-variant-numeric: tabular-nums;
}
.teacher-list__meta {
  font-size: 13px;
  font-variant-numeric: tabular-nums;
  color: var(--text-2);
}
.teacher-list__truncate {
  display: block;
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--text-2);
}
.teacher-list__actions {
  display: flex;
  align-items: center;
  gap: 2px;
}
</style>