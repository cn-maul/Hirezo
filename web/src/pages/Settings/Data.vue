<script setup lang="ts">
import Card from '@/components/ui/Card.vue'
import CardHeader from '@/components/ui/CardHeader.vue'
import CardTitle from '@/components/ui/CardTitle.vue'
import CardDescription from '@/components/ui/CardDescription.vue'
import CardContent from '@/components/ui/CardContent.vue'
</script>

<template>
  <div class="data">
    <h1 class="data__title">数据备份</h1>
    <p class="data__intro">
      数据存储在 SQLite 文件 <code class="data__code">hirezo.db</code> 中（WAL 模式下还有
      <code class="data__code">-wal</code> / <code class="data__code">-shm</code> 伴生文件），建议先停机再整目录备份。
    </p>

    <Card>
      <CardHeader>
        <CardTitle>备份建议</CardTitle>
        <CardDescription>两种常用方式</CardDescription>
      </CardHeader>
      <CardContent class="data__content">
        <p>
          <span class="data__strong">停机复制：</span>
          停止 Hirezo 进程后复制 <code class="data__code">hirezo.db</code>（及伴生文件）到安全位置。
        </p>
        <p>
          <span class="data__strong">在线快照：</span>
          不停机执行
          <code class="data__code">sqlite3 hirezo.db "VACUUM INTO '/backup/hirezo-snapshot.db'"</code>
          ，得到一致性强、体积更小的单文件快照。
        </p>
        <div class="data__divider" />
        <p>
          人员清单的 Excel 导出在「人员管理」页：按学科 / 性别 / 学历 / 资格证筛选后，
          点击工具栏的「导出 Excel」即可按当前条件导出。
        </p>
      </CardContent>
    </Card>
  </div>
</template>

<style scoped>
.data {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}
.data__title {
  font-size: 19px;
  line-height: 1.2;
  font-weight: 600;
  letter-spacing: -0.02em;
}
.data__intro {
  font-size: 14px;
  color: var(--text-2);
}
.data__code {
  border-radius: var(--r-chip);
  background: var(--bg);
  padding: 2px 6px;
  font-family: var(--mono);
  font-size: 13px;
  color: var(--text-body);
}
.data__content {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  font-size: 14px;
  color: var(--text-2);
}
.data__strong {
  font-weight: 500;
  color: var(--text);
}
.data__divider {
  border-top: 1px solid var(--hairline);
}
</style>