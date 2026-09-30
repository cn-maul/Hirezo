<script setup lang="ts">
import { computed } from 'vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import {
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
} from '@/components/ui'
import { useClass } from '@/lib/utils'

export interface Column {
  title: string
  key: string
  width?: number | string
}

const props = withDefaults(
  defineProps<{
    columns: Column[]
    dataSource: any[]
    rowKey: (r: any) => string | number
    loading?: boolean
    empty?: string
    selectable?: boolean
    selectedKeys?: (string | number)[]
  }>(),
  {
    loading: false,
    empty: '暂无数据',
    selectable: false,
    selectedKeys: () => [],
  },
)

const emit = defineEmits<{ 'update:selectedKeys': [value: (string | number)[]] }>()

const classes = useClass('surface-panel data-table')

const allKeys = computed(() => props.dataSource.map(props.rowKey))
const selectedSet = computed(() => new Set(props.selectedKeys))
const allChecked = computed(
  () => allKeys.value.length > 0 && allKeys.value.every((k) => selectedSet.value.has(k)),
)
const someChecked = computed(() => allKeys.value.some((k) => selectedSet.value.has(k)))

function toggleRow(key: string | number) {
  const next = selectedSet.value.has(key)
    ? props.selectedKeys.filter((k) => k !== key)
    : [...props.selectedKeys, key]
  emit('update:selectedKeys', next)
}

function toggleAll() {
  emit('update:selectedKeys', allChecked.value ? [] : allKeys.value)
}

const cols = computed(() => {
  if (!props.selectable) return props.columns
  return [{ title: '', key: '__select__', width: 36 }, ...props.columns]
})
</script>

<template>
  <div :class="classes">
    <div class="data-table__scroll">
      <div class="data-table__inner">
        <Table>
          <TableHeader>
            <TableRow class="data-table__header-row">
              <template v-for="c in cols" :key="c.key">
                <TableHead :style="c.width ? { width: String(c.width) } : undefined">
                  <template v-if="c.key === '__select__'">
                    <input
                      type="checkbox"
                      aria-label="全选本页"
                      :checked="allChecked"
                      :indeterminate="someChecked"
                      @change="toggleAll"
                    />
                  </template>
                  <template v-else>{{ c.title }}</template>
                </TableHead>
              </template>
            </TableRow>
          </TableHeader>
          <TableBody>
            <template v-if="loading">
              <TableRow v-for="i in 5" :key="`skeleton-${i}`" class="data-table__skeleton-row">
                <TableCell v-for="c in cols" :key="c.key">
                  <Skeleton class="data-table__skeleton" />
                </TableCell>
              </TableRow>
            </template>
            <template v-else-if="dataSource.length === 0">
              <TableRow class="data-table__empty-row">
                <TableCell :col-span="cols.length" class="data-table__empty">
                  {{ empty }}
                </TableCell>
              </TableRow>
            </template>
            <template v-else>
              <TableRow v-for="r in dataSource" :key="rowKey(r)">
                <template v-for="c in cols" :key="c.key">
                  <TableCell>
                    <template v-if="c.key === '__select__'">
                      <input
                        type="checkbox"
                        :aria-label="`选择 ${rowKey(r)}`"
                        :checked="selectedSet.has(rowKey(r))"
                        @change="toggleRow(rowKey(r))"
                      />
                    </template>
                    <template v-else>
                      <slot name="cell" :column="c" :record="r">
                        {{ (r as Record<string, unknown>)[c.key] }}
                      </slot>
                    </template>
                  </TableCell>
                </template>
              </TableRow>
            </template>
          </TableBody>
        </Table>
      </div>
    </div>
  </div>
</template>

<style scoped>
.data-table {
  overflow: hidden;
  border-radius: var(--r-panel);
}
.data-table__scroll {
  overflow-x: auto;
}
.data-table__inner {
  min-width: 960px;
}
.data-table__header-row:hover {
  background: transparent;
}
.data-table__skeleton-row:hover {
  background: transparent;
}
.data-table__skeleton {
  height: 16px;
  width: 100%;
}
.data-table__empty-row:hover {
  background: transparent;
}
.data-table__empty {
  height: 128px;
  text-align: center;
  font-size: 14px;
  color: var(--text-3);
}
</style>