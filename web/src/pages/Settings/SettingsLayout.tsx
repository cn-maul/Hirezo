import { NavLink, Outlet, Navigate } from 'react-router-dom'
import { Settings2, Tags, Database } from 'lucide-react'
import { cn } from '@/lib/utils'
import { APP_VERSION_LABEL } from '@/lib/version'
import { useAuth } from '@/auth'

const tabs = [
  { to: '/settings/general', icon: Settings2, label: '通用' },
  { to: '/settings/dicts', icon: Tags, label: '字典管理' },
  { to: '/settings/data', icon: Database, label: '数据备份' },
]

export default function SettingsLayout() {
  const { authed, user } = useAuth()

  if (authed === null) {
    return (
      <div className="flex h-64 items-center justify-center">
        <div className="size-7 animate-spin rounded-full border-2 border-[var(--track)] border-t-[var(--accent)]" />
      </div>
    )
  }

  if (!authed || user?.role !== 'admin') {
    return <Navigate to="/" replace />
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-end justify-between gap-2">
        <h1 className="text-[26px] leading-tight font-semibold tracking-[-0.025em]">设置</h1>
        <span className="rounded-full bg-[var(--track)] px-3 py-1 font-mono text-[12px] text-[var(--text-2)]">
          {APP_VERSION_LABEL}
        </span>
      </div>

      {/* Apple 分段控件：横向滚动只在窄屏发生，灰轨 + 滑动白 pill */}
      <div className="max-w-full overflow-x-auto">
        <div className="flex w-max items-center gap-0.5 rounded-[var(--r-pill)] bg-[var(--track)] p-1">
          {tabs.map((t) => (
            <NavLink
              key={t.to}
              to={t.to}
              className={({ isActive }) =>
                cn(
                  'flex items-center gap-2 rounded-[var(--r-pill)] px-4 py-2 text-[14px] font-medium whitespace-nowrap transition-[background-color,color,box-shadow] duration-200 ease-[var(--ease-out-quart)]',
                  isActive
                    ? 'bg-surface text-foreground shadow-[0_1px_3px_rgba(0,0,0,0.08),0_2px_8px_rgba(0,0,0,0.06)]'
                    : 'text-[var(--text-2)] hover:text-foreground',
                )
              }
            >
              <t.icon className="size-[18px] shrink-0" strokeWidth={1.75} />
              {t.label}
            </NavLink>
          ))}
        </div>
      </div>

      <Outlet />
    </div>
  )
}
