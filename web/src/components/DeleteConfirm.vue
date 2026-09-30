<script setup lang="ts">
import { ref } from 'vue'
import Button from '@/components/ui/Button.vue'
import Dialog from '@/components/ui/Dialog.vue'
import DialogHeader from '@/components/ui/DialogHeader.vue'
import DialogTitle from '@/components/ui/DialogTitle.vue'
import DialogDescription from '@/components/ui/DialogDescription.vue'
import DialogFooter from '@/components/ui/DialogFooter.vue'
import { useToast } from '@/composables/useToast'
import { errMsg } from '@/lib/utils'

withDefaults(
  defineProps<{
    title: string
    description: string
    confirmLabel?: string
  }>(),
  { confirmLabel: '删除' },
)

const emit = defineEmits<{ confirm: [] }>()

const { error } = useToast()
const open = ref(false)
const loading = ref(false)

async function handleConfirm() {
  loading.value = true
  try {
    await emit('confirm')
    open.value = false
  } catch (e) {
    error(errMsg(e, '操作失败'))
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <span @click.stop="open = true">
    <slot />
  </span>

  <Dialog v-model:open="open">
    <DialogHeader>
      <DialogTitle>{{ title }}</DialogTitle>
      <DialogDescription v-if="description">{{ description }}</DialogDescription>
    </DialogHeader>
    <DialogFooter>
      <Button variant="outline" :disabled="loading" @click="open = false">取消</Button>
      <Button variant="destructive" :loading="loading" @click="handleConfirm">
        {{ confirmLabel }}
      </Button>
    </DialogFooter>
  </Dialog>
</template>