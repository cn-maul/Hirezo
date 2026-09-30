<script setup lang="ts">
import { ref, watch } from 'vue'
import Card from '@/components/ui/Card.vue'
import CardHeader from '@/components/ui/CardHeader.vue'
import CardTitle from '@/components/ui/CardTitle.vue'
import CardDescription from '@/components/ui/CardDescription.vue'
import CardContent from '@/components/ui/CardContent.vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import Label from '@/components/ui/Label.vue'
import { getSettings, updateSettings } from '@/api/settings'
import { useQuery, invalidateQueries } from '@/composables/useQuery'
import { useToast } from '@/composables/useToast'

const DEFAULT_SITE_NAME = 'Hirezo 教师管理'

const { success, error } = useToast()
const name = ref('')
const saved = ref('')

const { data: settings } = useQuery(['settings'], getSettings)

watch(
  () => settings.value,
  (s) => {
    if (s) {
      const v = s.site_name || DEFAULT_SITE_NAME
      name.value = v
      saved.value = v
    }
  },
  { immediate: true },
)

const save = async () => {
  const v = name.value.trim() || DEFAULT_SITE_NAME
  try {
    await updateSettings({ site_name: v })
    name.value = v
    saved.value = v
    invalidateQueries(['settings'])
    success('已保存')
  } catch {
    error('保存失败')
  }
}

const restoreDefault = async () => {
  try {
    await updateSettings({ site_name: DEFAULT_SITE_NAME })
    name.value = DEFAULT_SITE_NAME
    saved.value = DEFAULT_SITE_NAME
    invalidateQueries(['settings'])
  } catch {
    error('恢复失败')
  }
}
</script>

<template>
  <div class="general">
    <h2 class="general__title">通用设置</h2>
    <Card>
      <CardHeader>
        <CardTitle>站点命名</CardTitle>
        <CardDescription>自定义左上角与登录页显示的站点名称</CardDescription>
      </CardHeader>
      <CardContent class="general__content">
        <div class="general__field">
          <Label html-for="site-name">站点名称</Label>
          <Input
            id="site-name"
            v-model="name"
            :placeholder="DEFAULT_SITE_NAME"
            :max-length="20"
          />
        </div>
        <div class="general__actions">
          <Button :disabled="name.trim() === saved" @click="save">保存</Button>
          <Button v-if="saved !== DEFAULT_SITE_NAME" variant="ghost" @click="restoreDefault">
            恢复默认
          </Button>
        </div>
      </CardContent>
    </Card>
  </div>
</template>

<style scoped>
.general {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}
.general__title {
  font-size: 19px;
  line-height: 1.2;
  font-weight: 600;
  letter-spacing: -0.02em;
}
.general__content {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}
.general__field {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}
.general__actions {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}
</style>