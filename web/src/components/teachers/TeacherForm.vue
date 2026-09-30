<script setup lang="ts">
import { computed, watch } from 'vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import Label from '@/components/ui/Label.vue'
import Switch from '@/components/ui/Switch.vue'
import Textarea from '@/components/ui/Textarea.vue'
import Select from '@/components/ui/Select.vue'
import SelectItem from '@/components/ui/SelectItem.vue'
import TeacherFieldNote from './TeacherFieldNote.vue'
import { useForm } from '@/composables/useForm'
import { validateTeacher, type TeacherFormValues } from '@/lib/validation'
import { cn } from '@/lib/utils'
import { genderLabel } from '@/api/teachers'
import type { Dictionary } from '@/api/dicts'
import type { Teacher, TeacherInput } from '@/api/teachers'
import type { ResumeFieldMeta, ResumeWarning } from '@/api/resume'

const ALL = '__all__'

type StringFieldKey = Exclude<keyof TeacherFormValues, 'has_cert'>

interface FieldDef {
  key: StringFieldKey
  label: string
  control: 'text' | 'number' | 'select' | 'textarea'
  placeholder?: string
  required?: boolean
  /** 字典类下拉：留空选项的名字 */
  allOption?: string
  dict?: 'subject' | 'education'
  full?: boolean
  min?: number
  max?: number
}

// 字段清单即渲染清单：增删字段只改这里，模板不再逐个重复。
// 顺序即双列栅格里的成行关系：姓名/性别、年龄/电话、学历/院校、学科/专业各一行，备注与教资整幅。
const FIELDS: FieldDef[] = [
  { key: 'name', label: '姓名', control: 'text', required: true, placeholder: '请输入姓名' },
  { key: 'gender', label: '性别', control: 'select' },
  {
    key: 'age',
    label: '年龄',
    control: 'number',
    placeholder: '18-100',
    min: 18,
    max: 100,
  },
  { key: 'phone', label: '联系电话', control: 'text', placeholder: '11 位手机号' },
  { key: 'education', label: '学历', control: 'select', dict: 'education', allOption: '未填写' },
  { key: 'university', label: '毕业院校', control: 'text', placeholder: '如：北京师范大学' },
  { key: 'subject', label: '学科', control: 'select', dict: 'subject', allOption: '未分配' },
  { key: 'major', label: '专业', control: 'text', placeholder: '如：汉语言文学' },
  { key: 'remark', label: '备注', control: 'textarea', placeholder: '可留空，最多 500 字', full: true },
]

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
    editing?: Teacher | null
    draft?: Teacher | null
    subjects?: Dictionary[]
    educations?: Dictionary[]
    submitting?: boolean
    fieldMeta?: Record<string, ResumeFieldMeta>
    warnings?: ResumeWarning[]
    submitLabel?: string
    cancelLabel?: string
  }>(),
  {
    editing: null,
    draft: null,
    subjects: () => [],
    educations: () => [],
    submitting: false,
    fieldMeta: undefined,
    warnings: undefined,
    submitLabel: '保存',
    cancelLabel: '取消',
  },
)

const emit = defineEmits<{
  submit: [values: TeacherInput]
  cancel: []
}>()

const source = computed(() => props.editing ?? props.draft ?? null)

// 字典值可能被停用但仍是某条记录的既有值：下拉只列启用项，
// 校验则放行全量（与后端 validDictValue 口径一致）。
function options(dict: Dictionary[]) {
  return {
    enabled: dict.filter((d) => d.enabled === 1).map((d) => d.name),
    all: dict.map((d) => d.name),
  }
}

const s = computed(() => options(props.subjects))
const ed = computed(() => options(props.educations))

function localErrors(v: TeacherFormValues) {
  const e = validateTeacher(v)
  if (v.subject !== ALL && !s.value.all.includes(v.subject)) e.subject = '学科不存在或已失效'
  if (v.education !== ALL && !ed.value.all.includes(v.education)) e.education = '学历不存在或已失效'
  return e
}

const { values, errors, reset, submit } = useForm<TeacherFormValues>(blank, localErrors)

// 切换录入对象时灌入新值（组件每次挂载即首跑一次，弹窗与内嵌表单都靠这里初始化）
watch(source, (t) => reset(toFormValues(t ?? undefined)), { immediate: true })

/** 下拉选项：性别固定两项，字典类列启用项，既有但已停用的值补在末尾 */
function selectOptions(f: FieldDef) {
  if (!f.dict) {
    return [
      { value: 'male', label: genderLabel.male },
      { value: 'female', label: genderLabel.female },
    ]
  }
  const enabled = (f.dict === 'subject' ? s.value : ed.value).enabled
  const cur = values[f.key]
  const items = enabled.map((n) => ({ value: n, label: n }))
  if (cur !== ALL && !enabled.includes(cur)) items.push({ value: cur, label: `${cur}（已停用）` })
  return [{ value: ALL, label: f.allOption ?? '未选择' }, ...items]
}

/** 触发器文字：编码值换成可读名，未选择时给引导语 */
function displayLabel(f: FieldDef, selected: string) {
  if (!f.dict) return genderLabel[selected] ?? '请选择性别'
  return selected === ALL ? `请选择${f.label}` : selected
}

// 简历识别回填：识别出的问题字段加色框提示人工复核
function fieldWrap(key: keyof TeacherFormValues) {
  const meta = props.fieldMeta?.[key]
  if (!meta || meta.level === 'ok') return ''
  const tint =
    meta.level === 'error'
      ? 'border-destructive/50 bg-destructive/5'
      : 'border-amber-500/50 bg-amber-500/5'
  return `rounded-[var(--r-thumb)] border p-2 ${tint}`
}

function onConfirm(v: TeacherFormValues) {
  const age = v.age.trim()
  emit('submit', {
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
  })
}

// 下拉面板向下展开会被滚动容器裁掉：点开时把该字段滚到容器上沿，留足 300px 弹层空间
function scrollBoxOf(el: HTMLElement) {
  for (let p = el.parentElement; p && p !== document.body; p = p.parentElement) {
    const oy = getComputedStyle(p).overflowY
    if ((oy === 'auto' || oy === 'scroll') && p.scrollHeight > p.clientHeight) return p
  }
  return null
}

function revealSelect(e: MouseEvent) {
  const sel = (e.target as HTMLElement | null)?.closest<HTMLElement>('.select')
  const box = sel ? scrollBoxOf(sel) : null
  if (!sel || !box) return
  const top = sel.getBoundingClientRect().top - box.getBoundingClientRect().top + box.scrollTop
  box.scrollTo({ top: Math.max(0, top - 12), behavior: 'smooth' })
}

// 预绑定 submit 处理器（避免 @submit="submit(onConfirm)" 被 Vue 编译器误判为 withModifiers）
const onSubmit = submit(onConfirm)
</script>

<template>
  <form class="teacher-form" @submit="onSubmit">
    <!-- 识别告警汇总 -->
    <div v-if="warnings?.length" class="teacher-warnings">
      <p class="teacher-warnings__title">识别到 {{ warnings.length }} 处需要核对的内容</p>
      <ul class="teacher-warnings__list">
        <li v-for="w in warnings" :key="`${w.field}-${w.message}`">{{ w.message }}</li>
      </ul>
    </div>

    <div class="teacher-grid" @click="revealSelect">
      <div
        v-for="f in FIELDS"
        :key="f.key"
        :class="cn('teacher-field', f.full && 'teacher-field--full', fieldWrap(f.key))"
      >
        <Label :html-for="`t-${f.key}`">{{ f.required ? `${f.label} *` : f.label }}</Label>

        <Select v-if="f.control === 'select'" :id="`t-${f.key}`" v-model="values[f.key]">
          <template #value="{ selected }">{{ displayLabel(f, selected) }}</template>
          <SelectItem v-for="o in selectOptions(f)" :key="o.value" :value="o.value">
            {{ o.label }}
          </SelectItem>
        </Select>

        <Textarea
          v-else-if="f.control === 'textarea'"
          :id="`t-${f.key}`"
          :rows="3"
          :placeholder="f.placeholder"
          v-model="values[f.key]"
        />

        <Input
          v-else
          :id="`t-${f.key}`"
          :type="f.control === 'number' ? 'number' : 'text'"
          :min="f.min"
          :max="f.max"
          :placeholder="f.placeholder"
          v-model="values[f.key]"
        />

        <TeacherFieldNote :error="errors[f.key]" />
      </div>

      <div :class="cn('teacher-cert', 'teacher-field--full', fieldWrap('has_cert'))">
        <div>
          <Label html-for="t-cert">教师资格证</Label>
          <p class="teacher-cert__hint">开启表示已取得教师资格证</p>
          <TeacherFieldNote :error="errors.has_cert" />
        </div>
        <Switch id="t-cert" v-model="values.has_cert" />
      </div>
    </div>

    <div class="teacher-form__actions">
      <Button type="button" variant="outline" @click="emit('cancel')">{{ cancelLabel }}</Button>
      <Button type="submit" :loading="submitting">
        {{ submitLabel }}
      </Button>
    </div>
  </form>
</template>

<style scoped>
.teacher-form {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}
.teacher-form__actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
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
  gap: 0.75rem;
}
@media (min-width: 640px) {
  .teacher-grid {
    grid-template-columns: 1fr 1fr;
  }
}

.teacher-field {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 0.375rem;
}
/* 窄列里下拉与输入框同宽，右边缘对齐 */
.teacher-field :deep(.select) {
  display: block;
}
.teacher-field :deep(.select-trigger) {
  width: 100%;
}
.teacher-field--full {
  grid-column: 1 / -1;
}

.teacher-cert {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  border-radius: var(--r-thumb);
  background: var(--bg);
  padding: 12px;
}
.teacher-cert__hint {
  font-size: 12px;
  color: var(--text-3);
}
</style>
