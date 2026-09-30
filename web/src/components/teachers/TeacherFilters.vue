<script setup lang="ts">
import { Download, Search } from 'lucide-vue-next'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import Select from '@/components/ui/Select.vue'
import SelectItem from '@/components/ui/SelectItem.vue'

export interface TeacherFilterValues {
  keyword: string
  subject: string
  gender: '' | 'male' | 'female'
  education: string
  hasCert: '' | '0' | '1'
}

const ALL = '__all__'

defineProps<{
  keyword: string
  subjects: string[]
  educations: string[]
  values: Omit<TeacherFilterValues, 'keyword'>
  exporting: boolean
}>()

const emit = defineEmits<{
  'update:keyword': [value: string]
  change: [next: Partial<Omit<TeacherFilterValues, 'keyword'>>]
  export: []
}>()
</script>

<template>
  <div class="filters">
    <div class="filters__search">
      <Search :size="18" :stroke-width="1.75" class="filters__search-icon" />
      <Input
        class="filters__search-input"
        placeholder="搜索姓名"
        :model-value="keyword"
        @update:model-value="emit('update:keyword', $event)"
      />
    </div>

    <Select
      :model-value="values.subject || ALL"
      class="filters__select filters__select--subject"
      @update:model-value="emit('change', { subject: $event === ALL ? '' : $event })"
    >
      <template #value="{ selected }">
        {{ selected === ALL ? '全部学科' : selected }}
      </template>
      <SelectItem :value="ALL">全部学科</SelectItem>
      <SelectItem v-for="s in subjects" :key="s" :value="s">{{ s }}</SelectItem>
    </Select>

    <Select
      :model-value="values.gender || ALL"
      class="filters__select filters__select--gender"
      @update:model-value="
        emit('change', { gender: ($event === ALL ? '' : $event) as '' | 'male' | 'female' })
      "
    >
      <template #value="{ selected }">
        {{ selected === ALL ? '性别' : selected === 'male' ? '男' : '女' }}
      </template>
      <SelectItem :value="ALL">全部性别</SelectItem>
      <SelectItem value="male">男</SelectItem>
      <SelectItem value="female">女</SelectItem>
    </Select>

    <Select
      :model-value="values.education || ALL"
      class="filters__select filters__select--education"
      @update:model-value="emit('change', { education: $event === ALL ? '' : $event })"
    >
      <template #value="{ selected }">
        {{ selected === ALL ? '学历' : selected }}
      </template>
      <SelectItem :value="ALL">全部学历</SelectItem>
      <SelectItem v-for="s in educations" :key="s" :value="s">{{ s }}</SelectItem>
    </Select>

    <Select
      :model-value="values.hasCert || ALL"
      class="filters__select filters__select--cert"
      @update:model-value="
        emit('change', { hasCert: ($event === ALL ? '' : $event) as '' | '0' | '1' })
      "
    >
      <template #value="{ selected }">
        {{
          selected === ALL
            ? '教师资格证'
            : selected === '1'
              ? '有资格证'
              : '无资格证'
        }}
      </template>
      <SelectItem :value="ALL">资格证不限</SelectItem>
      <SelectItem value="1">有资格证</SelectItem>
      <SelectItem value="0">无资格证</SelectItem>
    </Select>

    <Button variant="outline" :loading="exporting" @click="emit('export')">
      <template v-if="!exporting"><Download :size="18" /></template>
      导出 Excel
    </Button>
  </div>
</template>

<style scoped>
.filters {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.5rem;
}

.filters__search {
  position: relative;
  min-width: 220px;
  flex: 1;
}
@media (min-width: 640px) {
  .filters__search {
    flex: none;
  }
}
.filters__search-icon {
  position: absolute;
  top: 50%;
  left: 12px;
  transform: translateY(-50%);
  color: var(--text-3);
  pointer-events: none;
}
.filters__search-input {
  padding-left: 40px;
  width: 100%;
}
@media (min-width: 640px) {
  .filters__search-input {
    width: 256px;
  }
}

.filters__select--subject {
  width: 144px;
}
.filters__select--gender {
  width: 128px;
}
.filters__select--education {
  width: 128px;
}
.filters__select--cert {
  width: 144px;
}
</style>