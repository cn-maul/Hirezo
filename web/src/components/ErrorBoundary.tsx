import { Component, type ReactNode } from 'react'
import { AlertTriangle } from 'lucide-react'
import { Button } from '@/components/ui/button'

interface Props {
  children: ReactNode
}

interface State {
  hasError: boolean
  error: Error | null
}

export class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props)
    this.state = { hasError: false, error: null }
  }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error }
  }

  componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
    console.error('ErrorBoundary caught:', error, errorInfo)
  }

  render() {
    if (this.state.hasError) {
      return (
        <div className="flex min-h-screen flex-col items-center justify-center gap-4 bg-background p-8 text-center">
          <AlertTriangle className="size-11 text-[var(--heat)]" strokeWidth={1.5} />
          <div>
            <h1 className="text-[21px] font-semibold tracking-[-0.02em]">页面出错了</h1>
            <p className="mt-1.5 max-w-md text-[14px] leading-relaxed text-[var(--text-2)]">
              {this.state.error?.message || '发生了未知错误'}
            </p>
          </div>
          <Button onClick={() => window.location.reload()}>刷新页面</Button>
        </div>
      )
    }
    return this.props.children
  }
}
