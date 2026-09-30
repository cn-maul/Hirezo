<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import Pencil from 'lucide-vue-next/dist/esm/icons/pencil.js'
import Trash2 from 'lucide-vue-next/dist/esm/icons/trash-2.js'
import Download from 'lucide-vue-next/dist/esm/icons/download.js'
import {
  deleteTeacher,
  fetchTeacher,
  updateTeacher,
  ageOrDash,
  genderLabel,
  type TeacherInput,
} from '@/api/teachers'
import { downloadTeacherResume } from '@/api/resume'
import { fetchDictionaries } from '@/api/dicts'
import PageHeader from '@/components/PageHeader.vue'
import TeacherFormDialog from '@/components/teachers/TeacherFormDialog.vue'
import DeleteConfirm from '@/components/DeleteConfirm.vue'
import PageSpinner from '@/components/PageSpinner.vue'
import Button from '@/components/ui/Button.vue'
import Card from '@/components/ui/Card.vue'
import CardHeader from '@/components/ui/CardHeader.vue'
import CardTitle from '@/components/ui/CardTitle.vue'
import CardContent from '@/components/ui/CardContent.vue'
import { useQuery, useMutation, invalidateQueries, setQueryData } from '@/composables/useQuery'
import { useToast } from '@/composables/useToast'
import { categoryTint, cn, errMsg } from '@/lib/utils'

const route = useRoute()
const router = useRouter()
const { success, error } = useToast()
const editing = ref(false)

const teacherId = Number(route.params.id)

const { data: teacher, isLoading, error: loadError } = useQuery(
  ['teachers', teacherId],
  () => fetchTeacher(teacherId),
  { enabled: computed(() => Number.isInteger(teacherId) && teacherId > 0) },
)

const { data: subjects } = useQuery<import('@/api/dicts').Dictionary[]>(['dicts', 'subject'], () =>
  fetchDictionaries('subject'),
)
const { data: educations } = useQuery<import('@/api/dicts').Dictionary[]>(
  ['dicts', 'education'],
  () => fetchDictionaries('education'),
)

const remove = useMutation(() => deleteTeacher(teacherId))
const update = useMutation((input: TeacherInput) => updateTeacher(teacherId, input))
const updatePending = update.isPending

const onDelete = async () => {
  try {
    await remove.mutate()
    success('已删除')
    invalidateQueries(['teachers'])
    void router.replace('/')
  } catch (e) {
    error(errMsg(e, '删除失败'))
  }
}

const onUpdate = async (input: TeacherInput) => {
  try {
    const t = await update.mutate(input)
    success('已保存')
    setQueryData(['teachers', teacherId], t)
    invalidateQueries(['teachers'])
    editing.value = false
  } catch (e) {
    error(errMsg(e, '保存失败'))
  }
}

const subjectColor = computed(() => {
  const d = (subjects.value ?? []).find((x) => x.name === teacher.value?.subject)
  return d?.color ?? ''
})
const tint = computed(() => categoryTint(subjectColor.value))

const onResumeDownload = async () => {
  if (!teacher.value) return
  try {
    await downloadTeacherResume(teacherId, teacher.value.name)
  } catch (e) {
    error(errMsg(e, '下载简历失败'))
  }
}
</script>

<template>
  <div v-if="isLoading" class="teacher-detail">
    <PageSpinner />
  </div>

  <div v-else-if="loadError || !teacher" class="teacher-detail teacher-detail__not-found">
    <p class="teacher-detail__not-found-text">未找到该人员记录，可能已被删除</p>
    <Button variant="outline" @click="router.push('/')">返回列表</Button>
  </div>

  <div v-else class="teacher-detail">
    <PageHeader :title="teacher.name" description="人员档案详情" @back="router.back()">
      <template #action>
        <Button v-if="teacher.resume_file" variant="outline" @click="onResumeDownload">
          <Download :size="18" /> 下载简历
        </Button>
        <Button variant="outline" @click="editing = true">
          <Pencil :size="18" /> 编辑
        </Button>
        <DeleteConfirm
          title="确认删除"
          :description="`确定要删除「${teacher.name}」吗？删除后无法恢复。`"
          @confirm="onDelete"
        >
          <Button variant="outline" class="teacher-detail__danger-btn">
            <Trash2 :size="18" /> 删除
          </Button>
        </DeleteConfirm>
      </template>
    </PageHeader>

    <Card>
      <CardHeader>
        <CardTitle>基本信息</CardTitle>
      </CardHeader>
      <CardContent class="teacher-detail__fields">
        <div class="teacher-detail__field">
          <span class="teacher-detail__label">姓名</span>
          <span class="teacher-detail__value">{{ teacher.name }}</span>
        </div>
        <div class="teacher-detail__field">
          <span class="teacher-detail__label">性别</span>
          <span class="teacher-detail__value">{{ genderLabel[teacher.gender] ?? '—' }}</span>
        </div>
        <div class="teacher-detail__field">
          <span class="teacher-detail__label">年龄</span>
          <span class="teacher-detail__value tabular-nums">{{ ageOrDash(teacher.age) }}</span>
        </div>
        <div class="teacher-detail__field">
          <span class="teacher-detail__label">联系电话</span>
          <span v-if="teacher.phone" class="teacher-detail__phone">{{ teacher.phone }}</span>
          <span v-else>—</span>
        </div>
        <div class="teacher-detail__field">
          <span class="teacher-detail__label">学历</span>
          <span class="teacher-detail__value">{{ teacher.education || '—' }}</span>
        </div>
        <div class="teacher-detail__field">
          <span class="teacher-detail__label">毕业院校</span>
          <span class="teacher-detail__value">{{ teacher.university || '—' }}</span>
        </div>
        <div class="teacher-detail__field">
          <span class="teacher-detail__label">学科</span>
          <span class="teacher-detail__value">
            <span
              v-if="teacher.subject"
              :class="cn('cat-pill', !tint && 'teacher-detail__neutral-pill')"
              :style="tint ?? undefined"
            >
              {{ teacher.subject }}
            </span>
            <span v-else>—</span>
          </span>
        </div>
        <div class="teacher-detail__field">
          <span class="teacher-detail__label">专业</span>
          <span class="teacher-detail__value">{{ teacher.major || '—' }}</span>
        </div>
        <div class="teacher-detail__field teacher-detail__field--full">
          <span class="teacher-detail__label">教师资格证</span>
          <span
            :class="teacher.has_cert === 1 ? 'teacher-detail__cert-yes' : 'teacher-detail__cert-no'"
          >
            {{ teacher.has_cert === 1 ? '有' : '无' }}
          </span>
        </div>
        <div class="teacher-detail__field teacher-detail__field--full">
          <span class="teacher-detail__label">备注</span>
          <span class="teacher-detail__value">{{ teacher.remark || '—' }}</span>
        </div>
      </CardContent>
    </Card>

    <Card>
      <CardHeader>
        <CardTitle>时间</CardTitle>
      </CardHeader>
      <CardContent class="teacher-detail__fields">
        <div class="teacher-detail__field">
          <span class="teacher-detail__label">录入时间</span>
          <span class="teacher-detail__value tabular-nums">{{ teacher.created_at }}</span>
        </div>
        <div class="teacher-detail__field">
          <span class="teacher-detail__label">更新时间</span>
          <span class="teacher-detail__value tabular-nums">{{ teacher.updated_at }}</span>
        </div>
      </CardContent>
    </Card>

    <TeacherFormDialog
      v-model:open="editing"
      :editing="teacher"
      :subjects="subjects ?? []"
      :educations="educations ?? []"
      :submitting="updatePending"
      @submit="onUpdate"
    />
  </div>
</template>

<style scoped>
.teacher-detail {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}
.teacher-detail__not-found {
  align-items: center;
  padding: 32px 0;
  text-align: center;
}
.teacher-detail__not-found-text {
  font-size: 15px;
  color: var(--text-2);
}
.teacher-detail__danger-btn {
  color: var(--danger);
}
.teacher-detail__danger-btn:hover {
  background: rgba(215, 0, 21, 0.06);
  color: var(--danger);
}

/* 两列成行，与核对表单同样的配对；column-gap 留 0 好让行分隔线不断 */
.teacher-detail__fields {
  display: grid;
  grid-template-columns: 1fr 1fr;
}
.teacher-detail__field {
  display: flex;
  gap: 0.75rem;
  padding: 8px 24px 8px 0;
  border-top: 1px solid var(--hairline);
}
.teacher-detail__field:nth-child(-n + 2) {
  border-top: 0;
}
.teacher-detail__field--full {
  grid-column: 1 / -1;
  padding-right: 0;
}
@media (max-width: 720px) {
  .teacher-detail__fields {
    grid-template-columns: 1fr;
  }
  .teacher-detail__field:nth-child(-n + 2) {
    border-top: 1px solid var(--hairline);
  }
  .teacher-detail__field:first-child {
    border-top: 0;
  }
}
.teacher-detail__label {
  width: 96px;
  flex-shrink: 0;
  font-size: 14px;
  color: var(--text-3);
}
.teacher-detail__value {
  min-width: 0;
  flex: 1;
  font-size: 14px;
  color: var(--text);
  word-break: break-word;
}
.teacher-detail__neutral-pill {
  background: var(--track);
  color: var(--text-2);
}
.teacher-detail__cert-yes {
  color: var(--accent-link);
}
.teacher-detail__cert-no {
  color: var(--text-3);
}
.teacher-detail__phone {
  font-family: var(--mono);
  font-variant-numeric: tabular-nums;
}
</style>