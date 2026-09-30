<script setup lang="ts">
import { computed, watch } from 'vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import Label from '@/components/ui/Label.vue'
import Badge from '@/components/ui/Badge.vue'
import Dialog from '@/components/ui/Dialog.vue'
import DialogHeader from '@/components/ui/DialogHeader.vue'
import DialogTitle from '@/components/ui/DialogTitle.vue'
import DialogDescription from '@/components/ui/DialogDescription.vue'
import DialogFooter from '@/components/ui/DialogFooter.vue'
import { useAuth } from '@/composables/useAuth'
import { useToast } from '@/composables/useToast'
import { useForm } from '@/composables/useForm'
import { changePassword } from '@/api/auth'
import { validatePasswordChange, type PasswordChangeValues } from '@/lib/validation'
import { errMsg } from '@/lib/utils'

const props = withDefaults(defineProps<{ open?: boolean }>(), { open: false })
const emit = defineEmits<{ 'update:open': [value: boolean] }>()

const { user } = useAuth()
const { success, error } = useToast()

// v-model 兼容：open 是 prop（只读），用 computed 转发 update:open
const dialogOpen = computed({
  get: () => props.open,
  set: (v: boolean) => emit('update:open', v),
})

const { values, errors, reset, submit } = useForm<PasswordChangeValues>(
  { old_password: '', new_password: '', confirm: '' },
  validatePasswordChange,
)

// 打开时清空表单
watch(
  () => props.open,
  (v) => {
    if (v) reset()
  },
)

const onFinish = async ({ old_password, new_password }: PasswordChangeValues) => {
  try {
    await changePassword(old_password, new_password)
    reset()
    emit('update:open', false)
    success('密码已修改，其他设备需重新登录')
  } catch (e) {
    error(errMsg(e, '修改失败'))
  }
}

// 预绑定 submit 处理器（避免 @submit="submit(onFinish)" 被 Vue 编译器误判为 withModifiers）
const onSubmit = submit(onFinish)
</script>

<template>
  <Dialog v-model:open="dialogOpen">
    <DialogHeader>
      <DialogTitle>账号</DialogTitle>
      <DialogDescription class="account-desc">
        <span>{{ user?.display_name }}（@{{ user?.username }}）</span>
        <Badge variant="secondary">{{ user?.role === 'admin' ? '管理员' : '普通用户' }}</Badge>
      </DialogDescription>
    </DialogHeader>

    <form class="account-form" @submit="onSubmit">
      <div class="account-field">
        <Label html-for="acc-old">旧密码</Label>
        <Input
          id="acc-old"
          type="password"
          autocomplete="current-password"
          v-model="values.old_password"
        />
        <p v-if="errors.old_password" class="account-error">{{ errors.old_password }}</p>
      </div>
      <div class="account-field">
        <Label html-for="acc-new">新密码</Label>
        <Input
          id="acc-new"
          type="password"
          autocomplete="new-password"
          placeholder="至少 6 位"
          v-model="values.new_password"
        />
        <p v-if="errors.new_password" class="account-error">{{ errors.new_password }}</p>
      </div>
      <div class="account-field">
        <Label html-for="acc-confirm">确认新密码</Label>
        <Input
          id="acc-confirm"
          type="password"
          autocomplete="new-password"
          v-model="values.confirm"
        />
        <p v-if="errors.confirm" class="account-error">{{ errors.confirm }}</p>
      </div>
      <DialogFooter>
        <Button type="button" variant="outline" @click="emit('update:open', false)">取消</Button>
        <Button type="submit">修改密码</Button>
      </DialogFooter>
    </form>
  </Dialog>
</template>

<style scoped>
.account-desc {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}
.account-form {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}
.account-field {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}
.account-error {
  font-size: 13px;
  color: var(--danger);
}
</style>