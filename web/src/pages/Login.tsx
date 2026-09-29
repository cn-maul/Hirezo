import { useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { toast } from 'sonner'
import { GraduationCap, User, Lock } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { useFormState, validateLogin, type LoginFormValues } from '@/lib/validation'
import { errMsg } from '@/lib/utils'
import { useAuth } from '@/auth'
import { getSettings } from '@/api/settings'

export default function Login() {
  const { login } = useAuth()
  const navigate = useNavigate()
  const [params] = useSearchParams()
  const [loading, setLoading] = useState(false)

  const { data: settings } = useQuery({
    queryKey: ['settings'],
    queryFn: getSettings,
    staleTime: 60_000,
  })
  const siteName = settings?.site_name || 'Hirezo 教师管理'

  const { values, errors, set, submit } = useFormState<LoginFormValues>(
    { username: '', password: '' },
    validateLogin,
  )

  const onFinish = async ({ username, password }: LoginFormValues) => {
    setLoading(true)
    try {
      await login(username, password)
      // 防止开放重定向：解码后必须以/开头，不能是/login，不能以//开头（协议相对URL）。
      // 解码前的原始值可能是 %2F%2F 之类编码形式，故统一在解码后校验。
      let safeNext = '/'
      const next = params.get('next')
      if (next) {
        try {
          const decoded = decodeURIComponent(next)
          if (decoded.startsWith('/') && !decoded.startsWith('//') && !decoded.startsWith('/login')) {
            safeNext = decoded
          }
        } catch {
          // 非法百分号编码：忽略，回首页
        }
      }
      navigate(safeNext, { replace: true })
    } catch (e) {
      toast.error(errMsg(e, '登录失败'))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-background p-5">
      <Card className="w-full max-w-[380px] gap-6 rounded-[var(--r-hero)] px-1 py-8 shadow-[var(--sh-panel)]">
        <CardHeader className="items-center text-center">
          <div className="mb-2 grid size-13 place-items-center rounded-[var(--r-thumb)] bg-primary text-primary-foreground">
            <GraduationCap className="size-6" strokeWidth={1.75} />
          </div>
          <CardTitle className="text-[21px] tracking-[-0.02em]">{siteName}</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={submit(onFinish)} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="username">用户名</Label>
              <div className="relative">
                <User
                  className="absolute top-1/2 left-3 size-[18px] -translate-y-1/2 text-[var(--text-3)]"
                  strokeWidth={1.75}
                />
                <Input
                  id="username"
                  className="pl-10"
                  placeholder="用户名"
                  autoFocus
                  autoComplete="username"
                  value={values.username}
                  onChange={(e) => set('username', e.target.value)}
                />
              </div>
              {errors.username && (
                <p className="text-[13px] text-destructive">{errors.username}</p>
              )}
            </div>
            <div className="space-y-2">
              <Label htmlFor="password">密码</Label>
              <div className="relative">
                <Lock
                  className="absolute top-1/2 left-3 size-[18px] -translate-y-1/2 text-[var(--text-3)]"
                  strokeWidth={1.75}
                />
                <Input
                  id="password"
                  type="password"
                  className="pl-10"
                  placeholder="密码"
                  autoComplete="current-password"
                  value={values.password}
                  onChange={(e) => set('password', e.target.value)}
                />
              </div>
              {errors.password && (
                <p className="text-[13px] text-destructive">{errors.password}</p>
              )}
            </div>
            <Button type="submit" className="mt-2 w-full" size="lg" loading={loading}>
              登录
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}
