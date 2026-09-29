import { Link } from 'react-router-dom'
import { FileQuestion } from 'lucide-react'
import { Button } from '@/components/ui/button'

export default function NotFound() {
  return (
    <div className="flex min-h-screen flex-col items-center justify-center gap-5 bg-background p-8 text-center">
      <FileQuestion className="size-11 text-[var(--faint)]" strokeWidth={1.5} />
      <div>
        <h1 className="text-[21px] font-semibold tracking-[-0.02em]">页面不存在</h1>
        <p className="mt-1.5 text-[14px] text-[var(--text-2)]">您访问的页面不存在或已被移除</p>
      </div>
      <Button asChild>
        <Link to="/">返回首页</Link>
      </Button>
    </div>
  )
}
