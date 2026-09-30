<script setup lang="ts">
import { computed, watch } from 'vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import Label from '@/components/ui/Label.vue'
import Switch from '@/components/ui/Switch.vue'
import Textarea from '@/components/ui/Textarea.vue'
import Dialog from '@/components/ui/Dialog.vue'
import DialogHeader from '@/components/ui/DialogHeader.vue'
import DialogTitle from '@/components/ui/DialogTitle.vue'
import DialogFooter from '@/components/ui/DialogFooter.vue'
import Select from '@/components/ui/Select.vue'
import SelectItem from '@/components/ui/SelectItem.vue'
import { useForm } from '@/composables/useForm'
import { validateTeacher, type TeacherFormValues } from '@/lib/validation'
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

const props = withDefaults(
  defineProps<{
    open?: boolean
    editing?: Teacher | null
    draft?: Teacher | null
    subjects?: Dictionary[]
    educations?: Dictionary[]
    submitting?: boolean
    fieldMeta?: Record<string, ResumeFieldMeta>
    warnings?: ResumeWarning[]
  }>(),
  {
    open: false,
    editing: null,
    draft: null,
    subjects: () => [],
    educations: () => [],
    submitting: false,
    fieldMeta: undefined,
    warnings: undefined,
  },
)

const emit = defineEmits<{
  'update:open': [value: boolean]
  submit: [values: TeacherInput]
}>()

const source = computed(() => props.editing ?? props.draft ?? null)
const isResume = computed(() => !props.editing && !!props.draft)

// 字典值可能被停用但仍是某条记录的既有值：下拉只列启用项，
// 校验则放行全量（与后端 validDictValue 口径一致）。
function options(dict: Dictionary[]) {
  return {
    enabled: dict.filter((d) => d.enabled === 1).map((d) => d.name),
    all: dict.map((d) => d.name),
  }
}

function localErrors(v: TeacherFormValues) {
  const e = validateTeacher(v)
  const s = options(props.subjects)
  const ed = options(props.educations)
  if (v.subject !== ALL && !s.all.includes(v.subject)) e.subject = '学科不存在或已失效'
  if (v.education !== ALL && !ed.all.includes(v.education)) e.education = '学历不存在或已失效'
  return e
}

const { values, errors, reset, submit } = useForm<TeacherFormValues>(blank, localErrors)

// v-model 兼容：open 是 prop（只读），用 computed 转发 update:open
const dialogOpen = computed({
  get: () => props.open,
  set: (v: boolean) => emit('update:open', v),
})

// 打开时灌入待编辑数据；关闭后清空，避免下次打开残留
watch(
  () => props.open,
  (v) => {
    if (v) reset(toFormValues(source.value ?? undefined))
  },
)

const s = computed(() => options(props.subjects))
const ed = computed(() => options(props.educations))

const title = computed(() =>
  props.editing ? '编辑人员' : isResume.value ? '识别结果 · 请核对' : '新增人员',
)
const submitLabel = computed(() =>
  props.editing ? '保存' : isResume.value ? '确认入库' : '保存',
)

// 简历识别回填：识别出的问题字段加色框提示人工复核
function fieldWrap(meta?: ResumeFieldMeta, pad = true) {
  if (!meta || meta.level === 'ok') return ''
  const tint =
    meta.level === 'error'
      ? 'border-destructive/50 bg-destructive/5'
      : 'border-amber-500/50 bg-amber-500/5'
  return pad ? `rounded-[var(--r-thumb)] border p-2 ${tint}` : `rounded-[var(--r-thumb)] border ${tint}`
}

function onConfirm(v: TeacherFormValues) {
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
  emit('submit', input)
}

// 预绑定 submit 处理器（避免 @submit="submit(onConfirm)" 被 Vue 编译器误判为 withModifiers）
const onSubmit = submit(onConfirm)
</script>

<template>
  <Dialog v-model:open="dialogOpen" class="teacher-dialog">
    <DialogHeader>
      <DialogTitle>{{ title }}</DialogTitle>
    </DialogHeader>

    <form class="teacher-form" @submit="onSubmit">
      <!-- 识别告警汇总 -->
      <div v-if="warnings?.length" class="teacher-warnings">
        <p class="teacher-warnings__title">识别到 {{ warnings.length }} 处需要核对的内容</p>
        <ul class="teacher-warnings__list">
          <li v-for="w in warnings" :key="`${w.field}-${w.message}`">{{ w.message }}</li>
        </ul>
      </div>

      <div class="teacher-grid">
        <div :class="cn('teacher-field', fieldWrap(fieldMeta?.name))">
          <Label html-for="t-name">姓名 *</Label>
          <Input id="t-name" placeholder="请输入姓名" v-model="values.name" />
          <p v-if="fieldMeta?.name?.evidence?.trim()" class="teacher-evidence">
            识别依据：{{ fieldMeta.name.evidence.trim() }}
            <span class="teacher-evidence__conf">
              （{{ Math.round((fieldMeta.name.confidence || 0) * 100) }}%）
            </span>
          </p>
          <p v-if="errors.name" class="teacher-error">{{ errors.name }}</p>
        </div>

        <div :class="cn('teacher-field', fieldWrap(fieldMeta?.gender))">
          <Label html-for="t-gender">性别</Label>
          <Select v-model="values.gender" id="t-gender">
            <SelectItem value="male">男</SelectItem>
            <SelectItem value="female">女</SelectItem>
          </Select>
          <p v-if="fieldMeta?.gender?.evidence?.trim()" class="teacher-evidence">
            识别依据：{{ fieldMeta.gender.evidence.trim() }}
            <span class="teacher-evidence__conf">
              （{{ Math.round((fieldMeta.gender.confidence || 0) * 100) }}%）
            </span>
          </p>
          <p v-if="errors.gender" class="teacher-error">{{ errors.gender }}</p>
        </div>

        <div :class="cn('teacher-field', fieldWrap(fieldMeta?.age))">
          <Label html-for="t-age">年龄</Label>
          <Input
            id="t-age"
            type="number"
            :min="18"
            :max="100"
            placeholder="18-100，留空表示未填"
            v-model="values.age"
          />
          <p v-if="fieldMeta?.age?.evidence?.trim()" class="teacher-evidence">
            识别依据：{{ fieldMeta.age.evidence.trim() }}
            <span class="teacher-evidence__conf">
              （{{ Math.round((fieldMeta.age.confidence || 0) * 100) }}%）
            </span>
          </p>
          <p v-if="errors.age" class="teacher-error">{{ errors.age }}</p>
        </div>

        <div :class="cn('teacher-field', fieldWrap(fieldMeta?.phone))">
          <Label html-for="t-phone">联系电话</Label>
          <Input id="t-phone" placeholder="11 位手机号，可留空" v-model="values.phone" />
          <p v-if="fieldMeta?.phone?.evidence?.trim()" class="teacher-evidence">
            识别依据：{{ fieldMeta.phone.evidence.trim() }}
            <span class="teacher-evidence__conf">
              （{{ Math.round((fieldMeta.phone.confidence || 0) * 100) }}%）
            </span>
          </p>
          <p v-if="errors.phone" class="teacher-error">{{ errors.phone }}</p>
        </div>

        <div :class="cn('teacher-field', fieldWrap(fieldMeta?.subject))">
          <Label html-for="t-subject">学科</Label>
          <Select v-model="values.subject" id="t-subject">
            <template #value="{ selected }">
              {{ selected === ALL ? '请选择学科' : selected }}
            </template>
            <SelectItem :value="ALL">未分配</SelectItem>
            <SelectItem v-for="n in s.enabled" :key="n" :value="n">{{ n }}</SelectItem>
            <SelectItem
              v-if="values.subject !== ALL && !s.enabled.includes(values.subject)"
              :value="values.subject"
            >
              {{ values.subject }}（已停用）
            </SelectItem>
          </Select>
          <p v-if="fieldMeta?.subject?.evidence?.trim()" class="teacher-evidence">
            识别依据：{{ fieldMeta.subject.evidence.trim() }}
            <span class="teacher-evidence__conf">
              （{{ Math.round((fieldMeta.subject.confidence || 0) * 100) }}%）
            </span>
          </p>
          <p v-if="errors.subject" class="teacher-error">{{ errors.subject }}</p>
        </div>

        <div :class="cn('teacher-field', fieldWrap(fieldMeta?.education))">
          <Label html-for="t-education">学历</Label>
          <Select v-model="values.education" id="t-education">
            <template #value="{ selected }">
              {{ selected === ALL ? '请选择学历' : selected }}
            </template>
            <SelectItem :value="ALL">未填写</SelectItem>
            <SelectItem v-for="n in ed.enabled" :key="n" :value="n">{{ n }}</SelectItem>
            <SelectItem
              v-if="values.education !== ALL && !ed.enabled.includes(values.education)"
              :value="values.education"
            >
              {{ values.education }}（已停用）
            </SelectItem>
          </Select>
          <p v-if="fieldMeta?.education?.evidence?.trim()" class="teacher-evidence">
            识别依据：{{ fieldMeta.education.evidence.trim() }}
            <span class="teacher-evidence__conf">
              （{{ Math.round((fieldMeta.education.confidence || 0) * 100) }}%）
            </span>
          </p>
          <p v-if="errors.education" class="teacher-error">{{ errors.education }}</p>
        </div>

        <div :class="cn('teacher-field', fieldWrap(fieldMeta?.university))">
          <Label html-for="t-university">毕业院校</Label>
          <Input id="t-university" placeholder="如：北京师范大学" v-model="values.university" />
          <p v-if="fieldMeta?.university?.evidence?.trim()" class="teacher-evidence">
            识别依据：{{ fieldMeta.university.evidence.trim() }}
            <span class="teacher-evidence__conf">
              （{{ Math.round((fieldMeta.university.confidence || 0) * 100) }}%）
            </span>
          </p>
          <p v-if="errors.university" class="teacher-error">{{ errors.university }}</p>
        </div>

        <div :class="cn('teacher-field', fieldWrap(fieldMeta?.major))">
          <Label html-for="t-major">专业</Label>
          <Input id="t-major" placeholder="如：汉语言文学" v-model="values.major" />
          <p v-if="fieldMeta?.major?.evidence?.trim()" class="teacher-evidence">
            识别依据：{{ fieldMeta.major.evidence.trim() }}
            <span class="teacher-evidence__conf">
              （{{ Math.round((fieldMeta.major.confidence || 0) * 100) }}%）
            </span>
          </p>
          <p v-if="errors.major" class="teacher-error">{{ errors.major }}</p>
        </div>
      </div>

      <div :class="cn('teacher-cert', fieldWrap(fieldMeta?.has_cert, false))">
        <div>
          <Label html-for="t-cert">教师资格证</Label>
          <p class="teacher-cert__hint">开启表示已取得教师资格证</p>
          <p v-if="fieldMeta?.has_cert?.evidence?.trim()" class="teacher-evidence">
            识别依据：{{ fieldMeta.has_cert.evidence.trim() }}
            <span class="teacher-evidence__conf">
              （{{ Math.round((fieldMeta.has_cert.confidence || 0) * 100) }}%）
            </span>
          </p>
        </div>
        <Switch id="t-cert" v-model="values.has_cert" />
      </div>

      <div :class="cn('teacher-field', fieldWrap(fieldMeta?.remark))">
        <Label html-for="t-remark">备注</Label>
        <Textarea id="t-remark" :rows="3" placeholder="可留空，最多 500 字" v-model="values.remark" />
        <p v-if="fieldMeta?.remark?.evidence?.trim()" class="teacher-evidence">
          识别依据：{{ fieldMeta.remark.evidence.trim() }}
          <span class="teacher-evidence__conf">
            （{{ Math.round((fieldMeta.remark.confidence || 0) * 100) }}%）
          </span>
        </p>
        <p v-if="errors.remark" class="teacher-error">{{ errors.remark }}</p>
      </div>

      <slot name="extra" />

      <DialogFooter>
        <Button type="button" variant="outline" @click="emit('update:open', false)">取消</Button>
        <Button type="submit" :loading="submitting">{{ submitLabel }}</Button>
      </DialogFooter>
    </form>
  </Dialog>
</template>

<style scoped>
.teacher-dialog {
  max-width: 640px;
}
.teacher-form {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.teacher-warnings {
  border-radius: var(--r-thumb);
  border: 1px solid rgba(255, 159, 10, 0.5);
  background: rgba(255, 159, 10, 0.1);
  padding: 12px;
}
.teacher-warnings__title {
  font-size: 13px;
  font-weight: 500;
  color: var(--text);
}
.teacher-warnings__list {
  margin-top: 6px;
  padding-left: 20px;
  list-style: disc;
  display: flex;
  flex-direction: column;
  gap: 4px;
  font-size: 13px;
  color: var(--text-2);
}

.teacher-grid {
  display: grid;
  grid-template-columns: 1fr;
  gap: 1rem;
}
@media (min-width: 640px) {
  .teacher-grid {
    grid-template-columns: 1fr 1fr;
  }
}

.teacher-field {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.teacher-cert {
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-radius: var(--r-thumb);
  background: var(--bg);
  padding: 12px;
}
.teacher-cert__hint {
  font-size: 12px;
  color: var(--text-3);
}

.teacher-evidence {
  font-size: 12px;
  line-height: 1.4;
  color: var(--text-3);
}
.teacher-evidence__conf {
  margin-left: 6px;
  font-variant-numeric: tabular-nums;
}

.teacher-error {
  font-size: 13px;
  color: var(--danger);
}
</style>