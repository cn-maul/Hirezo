<script setup lang="ts">
import { computed } from 'vue'
import Dialog from '@/components/ui/Dialog.vue'
import DialogHeader from '@/components/ui/DialogHeader.vue'
import DialogTitle from '@/components/ui/DialogTitle.vue'
import TeacherForm from './TeacherForm.vue'
import type { Dictionary } from '@/api/dicts'
import type { Teacher, TeacherInput } from '@/api/teachers'

const props = withDefaults(
  defineProps<{
    open?: boolean
    editing?: Teacher | null
    subjects?: Dictionary[]
    educations?: Dictionary[]
    submitting?: boolean
  }>(),
  {
    open: false,
    editing: null,
    subjects: () => [],
    educations: () => [],
    submitting: false,
  },
)

const emit = defineEmits<{
  'update:open': [value: boolean]
  submit: [values: TeacherInput]
}>()

// v-model 兼容：open 是 prop（只读），用 computed 转发 update:open
const dialogOpen = computed({
  get: () => props.open,
  set: (v: boolean) => emit('update:open', v),
})
</script>

<template>
  <Dialog v-model:open="dialogOpen" class="teacher-dialog">
    <DialogHeader>
      <DialogTitle>{{ props.editing ? '编辑人员' : '新增人员' }}</DialogTitle>
    </DialogHeader>

    <TeacherForm
      :editing="props.editing"
      :subjects="props.subjects"
      :educations="props.educations"
      :submitting="props.submitting"
      @submit="emit('submit', $event)"
      @cancel="dialogOpen = false"
    />
  </Dialog>
</template>

<style scoped>
.teacher-dialog {
  max-width: 640px;
}
</style>
