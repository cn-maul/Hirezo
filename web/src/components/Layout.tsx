import { useEffect, useRef, useState } from 'react'
import { NavLink, Outlet, useNavigate, useLocation } from 'react-router-dom'
import {
  Users,
  Settings,
  Sun,
  Moon,
  LogOut,
  PanelLeft,
  GraduationCap,
  FileSearch,
} from 'lucide-react'
import { useQuery } from '@tanstack/react-query'
import { Button } from '@/components/ui/button'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { cn } from '@/lib/utils'
import { useTheme } from '@/lib/theme'
import { useAuth } from '@/auth'
import { getSettings } from '@/api/settings'
import AccountDialog from '@/components/AccountDialog'

const DEFAULT_SITE_NAME = 'Hirezo 教师管理'

function NavItem({ to, icon: Icon, label, collapsed, end = false }: {
  to: string
  icon: typeof Users
  label: string
  collapsed: boolean
  end?: boolean
}) {
  return (
    <NavLink
      to={to}
      end={end}
      className={({ isActive }) =>
        cn(
          'flex items-center gap-3 rounded-[var(--r-thumb)] px-3 py-2 text-[14px] font-medium transition-[background-color,color] duration-200 ease-[var(--ease-out-quart)]',
          collapsed && 'justify-center px-2',
          isActive
            ? 'bg-[rgba(0,113,227,0.1)] text-[var(--accent-link)]'
            : 'text-[var(--text-2)] hover:bg-hover hover:text-foreground',
        )
      }
    >
      <Icon className="size-[18px] shrink-0" strokeWidth={1.75} />
      {!collapsed && <span>{label}</span>}
    </NavLink>
  )
}

export default function Layout() {
  const [collapsed, setCollapsed] = useState(false)
  const [accountOpen, setAccountOpen] = useState(false)
  const [scrolled, setScrolled] = useState(false)
  const scrollRef = useRef<HTMLElement>(null)
  const navigate = useNavigate()
  const location = useLocation()
  const { dark, toggle } = useTheme()
  const { logout, user } = useAuth()
  const isAdmin = user?.role === 'admin'

  const { data: settings } = useQuery({
    queryKey: ['settings'],
    queryFn: getSettings,
  })
  const siteName = settings?.site_name || DEFAULT_SITE_NAME

  // Below the tablet breakpoint a 224px sidebar eats the viewport, so force it
  // to icon-only. The user's own toggle still wins once they're above it.
  const [narrow, setNarrow] = useState(
    () => typeof window !== 'undefined' && window.matchMedia('(max-width: 900px)').matches,
  )
  useEffect(() => {
    const mq = window.matchMedia('(max-width: 900px)')
    const onChange = (e: MediaQueryListEvent) => setNarrow(e.matches)
    mq.addEventListener('change', onChange)
    return () => mq.removeEventListener('change', onChange)
  }, [])
  const railCollapsed = collapsed || narrow

  // Scroll-edge: the topbar only grows its hairline once content slides underneath.
  useEffect(() => {
    const el = scrollRef.current
    if (!el) return
    const onScroll = () => setScrolled(el.scrollTop > 4)
    onScroll()
    el.addEventListener('scroll', onScroll, { passive: true })
    return () => el.removeEventListener('scroll', onScroll)
  }, [])

  // Route changes reset the scroll position; keep the edge state honest.
  useEffect(() => {
    scrollRef.current?.scrollTo({ top: 0 })
    setScrolled(false)
  }, [location.pathname])

  const onLogout = async () => {
    await logout()
    navigate('/login', { replace: true })
  }

  const displayName = user?.display_name || user?.username || '管'
  const initial = displayName.charAt(0)

  return (
    <div className="flex h-screen overflow-hidden bg-background">
      <aside
        className={cn(
          'flex shrink-0 flex-col bg-sidebar transition-[width] duration-300 ease-[var(--ease-out-quart)]',
          railCollapsed ? 'w-16' : 'w-56',
        )}
      >
        <div className="flex h-16 shrink-0 items-center gap-2.5 px-3">
          <div
            className={cn(
              'grid size-8 shrink-0 place-items-center rounded-[var(--r-chip)] bg-primary text-primary-foreground',
              railCollapsed && 'mx-auto',
            )}
          >
            <GraduationCap className="size-[18px]" strokeWidth={1.75} />
          </div>
          {!railCollapsed && (
            <span className="truncate text-[15px] font-semibold tracking-[-0.015em] text-foreground">
              {siteName}
            </span>
          )}
        </div>

        <nav className="flex-1 space-y-1 overflow-y-auto px-2 pb-4">
          <NavItem to="/teachers" icon={Users} label="人员管理" collapsed={railCollapsed} end />
          <NavItem to="/resume" icon={FileSearch} label="简历识别" collapsed={railCollapsed} />
          {isAdmin && <NavItem to="/settings" icon={Settings} label="设置" collapsed={railCollapsed} />}
        </nav>

        {/* 侧栏底部仅保留折叠；主题与账号操作统一收在顶栏右侧 */}
        <div className="p-2">
          <Button
            variant="ghost"
            size="icon"
            onClick={() => setCollapsed((c) => !c)}
            aria-label="收起侧边栏"
            className={cn(railCollapsed && 'mx-auto')}
          >
            <PanelLeft strokeWidth={1.75} />
          </Button>
        </div>
      </aside>

      <div className="flex min-w-0 flex-1 flex-col">
        <header
          className={cn(
            'glass-nav z-20 flex h-16 shrink-0 items-center justify-end gap-1 px-3 transition-shadow duration-300 ease-[var(--ease-out-quart)] sm:px-4',
            scrolled && 'border-b border-[var(--hairline)] shadow-[0_4px_16px_rgba(0,0,0,0.03)]',
          )}
        >
          <Tooltip>
            <TooltipTrigger asChild>
              <Button variant="ghost" size="icon" onClick={toggle} aria-label="切换主题">
                {dark ? <Sun strokeWidth={1.75} /> : <Moon strokeWidth={1.75} />}
              </Button>
            </TooltipTrigger>
            <TooltipContent>{dark ? '切换到亮色模式' : '切换到暗色模式'}</TooltipContent>
          </Tooltip>
          <Tooltip>
            <TooltipTrigger asChild>
              <Button variant="ghost" size="icon" onClick={onLogout} aria-label="退出登录">
                <LogOut strokeWidth={1.75} />
              </Button>
            </TooltipTrigger>
            <TooltipContent>退出登录</TooltipContent>
          </Tooltip>
          <Tooltip>
            <TooltipTrigger asChild>
              <button
                type="button"
                onClick={() => setAccountOpen(true)}
                aria-label="账号设置"
                className="ml-1 grid size-9 cursor-pointer place-items-center rounded-full bg-[var(--track)] text-[13px] font-semibold text-[var(--text-2)] transition-[background-color,color,transform] duration-200 ease-[var(--ease-out-quart)] hover:bg-[var(--faint)] hover:text-foreground active:scale-95"
              >
                {initial}
              </button>
            </TooltipTrigger>
            <TooltipContent>{displayName}</TooltipContent>
          </Tooltip>
        </header>

        <main ref={scrollRef} className="flex-1 overflow-y-auto">
          <div className="mx-auto w-full max-w-[1600px] px-5 py-6 md:px-8 md:py-8">
            <Outlet />
          </div>
        </main>
      </div>

      <AccountDialog open={accountOpen} onOpenChange={setAccountOpen} />
    </div>
  )
}
