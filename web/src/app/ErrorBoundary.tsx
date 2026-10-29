import { Component, type ErrorInfo, type ReactNode } from 'react'
import { StateBlock } from '../shared/ui/primitives'

/** Top-level boundary: a render error shows a truthful error state, never a blank page. */
export class ErrorBoundary extends Component<{ children: ReactNode }, { failed: boolean }> {
  override state = { failed: false }

  static getDerivedStateFromError(): { failed: boolean } {
    return { failed: true }
  }

  override componentDidCatch(error: Error, info: ErrorInfo): void {
    console.error('render error', error, info.componentStack)
  }

  override render(): ReactNode {
    if (this.state.failed) {
      return (
        <main id="ws" className="ws">
          <StateBlock kind="error" title="خطا در نمایش صفحه"
            detail="این بخش به‌درستی نمایش داده نشد. صفحه را دوباره بارگذاری کنید." />
        </main>
      )
    }
    return this.props.children
  }
}
