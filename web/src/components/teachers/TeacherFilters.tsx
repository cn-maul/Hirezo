import { Download, Search } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

export interface TeacherFilterValues {
  keyword: string
  subject: string
  gender: '' | 'male' | 'female'
  education: string
  hasCert: '' | '0' | '1'
}

const ALL = '__all__'

export default function TeacherFilters({
  keyword,
  onKeywordChange,
  subjects,
  educations,
  values,
  onChange,
  onExport,
  exporting,
}: {
  keyword: string
  onKeywordChange: (v: string) => void
  subjects: string[]
  educations: string[]
  values: Omit<TeacherFilterValues, 'keyword'>
  onChange: (next: Partial<Omit<TeacherFilterValues, 'keyword'>>) => void
  onExport: () => void
  exporting: boolean
}) {
  return (
    <div className="flex flex-wrap items-center gap-2">
      <div className="relative min-w-[220px] flex-1 sm:flex-none">
        <Search
          className="absolute top-1/2 left-3 size-[18px] -translate-y-1/2 text-[var(--text-3)]"
          strokeWidth={1.75}
        />
        <Input
          className="w-full pl-10 sm:w-64"
          placeholder="搜索姓名"
          value={keyword}
          onChange={(e) => onKeywordChange(e.target.value)}
        />
      </div>

      <Select value={values.subject || ALL} onValueChange={(v) => onChange({ subject: v === ALL ? '' : v })}>
        <SelectTrigger className="w-36"><SelectValue placeholder="全部学科" /></SelectTrigger>
        <SelectContent>
          <SelectItem value={ALL}>全部学科</SelectItem>
          {subjects.map((s) => <SelectItem key={s} value={s}>{s}</SelectItem>)}
        </SelectContent>
      </Select>

      <Select value={values.gender || ALL} onValueChange={(v) => onChange({ gender: (v === ALL ? '' : v) as '' | 'male' | 'female' })}>
        <SelectTrigger className="w-32"><SelectValue placeholder="性别" /></SelectTrigger>
        <SelectContent>
          <SelectItem value={ALL}>全部性别</SelectItem>
          <SelectItem value="male">男</SelectItem>
          <SelectItem value="female">女</SelectItem>
        </SelectContent>
      </Select>

      <Select value={values.education || ALL} onValueChange={(v) => onChange({ education: v === ALL ? '' : v })}>
        <SelectTrigger className="w-32"><SelectValue placeholder="学历" /></SelectTrigger>
        <SelectContent>
          <SelectItem value={ALL}>全部学历</SelectItem>
          {educations.map((s) => <SelectItem key={s} value={s}>{s}</SelectItem>)}
        </SelectContent>
      </Select>

      <Select value={values.hasCert || ALL} onValueChange={(v) => onChange({ hasCert: (v === ALL ? '' : v) as '' | '0' | '1' })}>
        <SelectTrigger className="w-36"><SelectValue placeholder="教师资格证" /></SelectTrigger>
        <SelectContent>
          <SelectItem value={ALL}>资格证不限</SelectItem>
          <SelectItem value="1">有资格证</SelectItem>
          <SelectItem value="0">无资格证</SelectItem>
        </SelectContent>
      </Select>

      <Button variant="outline" onClick={onExport} loading={exporting}>
        {!exporting && <Download />}导出 Excel
      </Button>
    </div>
  )
}
