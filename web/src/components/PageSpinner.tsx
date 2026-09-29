// 页面级加载态统一组件：替代各页手写 spinner（尺寸/样式保持一致）。
// 表格类数据仍使用 DataTable 内置的骨架屏。
export default function PageSpinner({ label = '加载中…' }: { label?: string }) {
  return (
    <div className="flex h-64 flex-col items-center justify-center gap-3" role="status" aria-label={label}>
      <div className="size-7 animate-spin rounded-full border-2 border-[var(--track)] border-t-[var(--accent)]" />
      <span className="text-[13px] text-[var(--text-2)]">{label}</span>
    </div>
  )
}
