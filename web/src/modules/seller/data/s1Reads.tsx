import { createContext, useContext, useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { useSellerPreview } from '../state/SellerPreviewContext'
import { classifyAnswer } from '../state/uiState'
import { ReadSequencer, createSyntheticS1Source, isAbortError, type S1Read, type S1ReadSource } from './s1ReadSource'

/*
 * RM-E04K — React side of the S1 read boundary: which source the screen reads from, and a sequenced
 * read hook that only ever shows the answer to the CURRENT request. Presentation only; grants nothing.
 */

const S1SourceCtx = createContext<S1ReadSource | null>(null)

/**
 * Supplies an explicit S1 read source (tests; a future authorized transport source). Without it the
 * screen reads from the synthetic source for the current preview scenario.
 */
export function S1ReadSourceProvider({ source, children }: { source: S1ReadSource; children: ReactNode }) {
  return <S1SourceCtx.Provider value={source}>{children}</S1SourceCtx.Provider>
}

export function useS1ReadSource(): S1ReadSource {
  const injected = useContext(S1SourceCtx)
  const { sim } = useSellerPreview()
  // A new scenario is a new source: every read restarts and nothing from the previous scenario survives.
  const synthetic = useMemo(() => (injected ? null : createSyntheticS1Source({ scenario: sim })), [injected, sim])
  return injected ?? (synthetic as S1ReadSource)
}

export type ReadPhase<T> =
  | { status: 'idle' }
  | { status: 'loading' }
  | { status: 'done'; read: S1Read<T> }

/** A rejected read that was not an abort is a transport failure — never an empty result. */
function transportFailure<T>(): S1Read<T> {
  const answer = { status: 0 }
  return { answer, state: classifyAnswer(answer), data: null }
}

/**
 * Runs `run` whenever `key` changes and returns the phase for THAT key only.
 * - A result is applied only if its request is still the newest one (ReadSequencer); a late answer for an
 *   earlier key, a closed drawer (key → null) or an unmounted screen is dropped and its request aborted.
 * - The returned phase is computed against the current key at render time, so the frame after a key change
 *   already shows `loading`, never the previous key's data.
 * - `onSettled` sees each applied result once (used to latch authorization blocks).
 */
export function useSequencedRead<T>(
  key: string | null,
  run: (signal: AbortSignal) => Promise<S1Read<T>>,
  onSettled?: (read: S1Read<T>) => void,
): ReadPhase<T> {
  const [seq] = useState(() => new ReadSequencer())
  const [held, setHeld] = useState<{ key: string; read: S1Read<T> } | null>(null)
  const latest = useRef({ run, onSettled })
  useEffect(() => { latest.current = { run, onSettled } })

  useEffect(() => {
    if (key === null) { seq.cancel(); return }
    const { generation, signal } = seq.begin()
    const apply = (read: S1Read<T>) => {
      if (!seq.isCurrent(generation)) return
      latest.current.onSettled?.(read)
      setHeld({ key, read })
    }
    latest.current.run(signal).then(apply, (e: unknown) => { if (!isAbortError(e)) apply(transportFailure<T>()) })
    return () => { seq.cancel(); setHeld(null) }
  }, [key, seq])

  if (key === null) return { status: 'idle' }
  if (held && held.key === key) return { status: 'done', read: held.read }
  return { status: 'loading' }
}
