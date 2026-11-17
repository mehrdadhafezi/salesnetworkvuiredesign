import { useCallback, useEffect } from 'react'
import { useSearchParams } from 'react-router'

/**
 * Drawer target kept in the URL query (prototype deep links ?lead= / ?inv= / ?case=) so the preview
 * can be screenshotted and linked. Presentation state only — never an authorization input.
 */
export function useDrawerParam(key: string): [string | null, (id: string | null) => void] {
  const [params, setParams] = useSearchParams()
  const value = params.get(key)
  const set = useCallback((id: string | null) => {
    setParams(prev => {
      const next = new URLSearchParams(prev)
      if (id === null) { next.delete(key); next.delete('outcome') } else next.set(key, id)
      return next
    }, { replace: true })
  }, [key, setParams])
  return [value, set]
}

/** True when a key press should be left to the focused control (prototype keydown `onControl`). */
export function isTypingTarget(el: Element | null): boolean {
  if (!el) return false
  const tag = el.tagName
  return tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT' || (el as HTMLElement).isContentEditable
}

export function isControlTarget(el: Element | null): boolean {
  if (!el) return false
  return isTypingTarget(el) || !!el.closest('button, a[href], [role="button"], [role="menuitem"], [role="option"], summary, [role="dialog"][aria-modal="true"]')
}

/** Page-level single-key shortcuts; ignored with modifiers and while a control has focus. */
export function usePageKeys(handler: (key: string, e: KeyboardEvent) => boolean, enabled = true) {
  useEffect(() => {
    if (!enabled) return
    const onKey = (e: KeyboardEvent) => {
      if (e.defaultPrevented || e.ctrlKey || e.metaKey || e.altKey) return
      if (isControlTarget(document.activeElement) && !(document.activeElement as HTMLElement | null)?.closest('tr[data-row]')) return
      const fa = '۰۱۲۳۴۵۶۷۸۹'.indexOf(e.key)
      const k = fa > -1 ? String(fa) : e.key === 'ب' ? 'f' : e.key === 'ت' ? 'j' : e.key === 'ن' ? 'k' : e.key.toLowerCase()
      if (handler(k, e)) e.preventDefault()
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [handler, enabled])
}
