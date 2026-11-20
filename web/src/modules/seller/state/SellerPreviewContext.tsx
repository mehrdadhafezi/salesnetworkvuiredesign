import { createContext, useCallback, useContext, useMemo, useRef, useState, type ReactNode } from 'react'
import { STATE_POLICY, answerOfSim, stateOfSim, type SellerUiState, type ServerAnswer, type Sim, type StatePolicy } from './uiState'

/*
 * Local, in-memory presentation state of the RM-E03 Seller preview: theme / density, the demo
 * system-state simulator (prototype `?sim=`), and toasts. Nothing is persisted, sent or shared;
 * a reload starts clean. This context grants nothing and is not a session or identity.
 */

export type Theme = 'light' | 'dim' | 'dark'
export type Density = 'comfortable' | 'compact'
export type { Sim } from './uiState'
export type ToastKind = 'info' | 'success' | 'warning' | 'error'

export interface Toast { id: number; msg: string; kind: ToastKind }

interface Ctx {
  base: string
  theme: Theme
  density: Density
  sim: Sim
  setTheme: (t: Theme) => void
  setDensity: (d: Density) => void
  setSim: (s: Sim) => void
  /** A7: the presentation state the current (synthetic) answer stands for, its policy and the answer itself. */
  ui: { state: SellerUiState; policy: StatePolicy; answer: ServerAnswer }
  toasts: Toast[]
  toast: (msg: string, kind?: ToastKind) => void
  dismiss: (id: number) => void
}

const SellerPreviewCtx = createContext<Ctx | null>(null)

export function SellerPreviewProvider({ base, initial, children }: {
  base: string
  initial: { theme: Theme; density: Density; sim: Sim }
  children: ReactNode
}) {
  const [theme, setTheme] = useState(initial.theme)
  const [density, setDensity] = useState(initial.density)
  const [sim, setSimState] = useState(initial.sim)
  const [toasts, setToasts] = useState<Toast[]>([])
  // RM-E04J: a blocking answer also clears pending toasts, so no earlier message outlives the denial.
  const setSim = useCallback((s: Sim) => {
    setSimState(s)
    if (STATE_POLICY[stateOfSim(s)].blocksContent) setToasts([])
  }, [])
  const nextId = useRef(1)
  const dismiss = useCallback((id: number) => setToasts(ts => ts.filter(t => t.id !== id)), [])
  // Prototype toast(): short, auto-dismiss (4.5 s), manually closable, at most 3 stacked.
  const toast = useCallback((msg: string, kind: ToastKind = 'info') => {
    const id = nextId.current++
    setToasts(ts => [...ts, { id, msg, kind }].slice(-3))
    window.setTimeout(() => dismiss(id), 4500)
  }, [dismiss])
  const ui = useMemo(() => { const state = stateOfSim(sim); return { state, policy: STATE_POLICY[state], answer: answerOfSim(sim) } }, [sim])
  const value = useMemo<Ctx>(() => ({ base, theme, density, sim, setTheme, setDensity, setSim, ui, toasts, toast, dismiss }), [base, theme, density, sim, setSim, ui, toasts, toast, dismiss])
  return <SellerPreviewCtx.Provider value={value}>{children}</SellerPreviewCtx.Provider>
}

export function useSellerPreview(): Ctx {
  const c = useContext(SellerPreviewCtx)
  if (!c) throw new Error('useSellerPreview outside SellerPreviewProvider')
  return c
}
