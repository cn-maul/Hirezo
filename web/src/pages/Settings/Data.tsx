import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'

const code = (s: string) => (
  <code className="rounded-[var(--r-chip)] bg-background px-1.5 py-0.5 font-mono text-[13px] text-[var(--text-body)]">
    {s}
  </code>
)

// 数据与备份：导出入口在人员列表页（跟随当前筛选条件），本页只做存储说明。
export default function Data() {
  return (
    <div className="space-y-3">
      <h1 className="text-[19px] leading-tight font-semibold tracking-[-0.02em]">数据备份</h1>
      <p className="text-[14px] text-[var(--text-2)]">
        数据存储在 SQLite 文件 {code('hirezo.db')} 中（WAL 模式下还有 {code('-wal')} / {code('-shm')}{' '}
        伴生文件），建议先停机再整目录备份。
      </p>

      <Card>
        <CardHeader>
          <CardTitle>备份建议</CardTitle>
          <CardDescription>两种常用方式</CardDescription>
        </CardHeader>
        <CardContent className="space-y-3 text-[14px] text-[var(--text-2)]">
          <p>
            <span className="font-medium text-foreground">停机复制：</span>
            停止 Hirezo 进程后复制 {code('hirezo.db')}（及伴生文件）到安全位置。
          </p>
          <p>
            <span className="font-medium text-foreground">在线快照：</span>
            不停机执行{' '}
            {code("sqlite3 hirezo.db \"VACUUM INTO '/backup/hirezo-snapshot.db'\"")}
            ，得到一致性强、体积更小的单文件快照。
          </p>
          <div className="border-t border-[var(--hairline)]" />
          <p>
            人员清单的 Excel 导出在「人员管理」页：按学科 / 性别 / 学历 / 资格证筛选后，
            点击工具栏的「导出 Excel」即可按当前条件导出。
          </p>
        </CardContent>
      </Card>
    </div>
  )
}
