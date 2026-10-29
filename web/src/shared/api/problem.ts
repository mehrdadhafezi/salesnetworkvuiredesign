import type { Problem } from './types'

/** A failed API call: a contract Problem, or a transport failure with no response. */
export type ApiFailure =
  | { kind: 'problem'; problem: Problem }
  | { kind: 'transport'; message: string }

const PROBLEM_TYPE = 'application/problem+json'

/** Classifies a failed call. Unknown error bodies are never guessed into a Problem. */
export function toFailure(error: unknown, response: Response | undefined): ApiFailure {
  if (response && response.headers.get('Content-Type')?.startsWith(PROBLEM_TYPE) && isProblem(error)) {
    return { kind: 'problem', problem: error }
  }
  if (!response) {
    return { kind: 'transport', message: error instanceof Error ? error.message : 'network unavailable' }
  }
  return { kind: 'transport', message: `unexpected response (HTTP ${response.status})` }
}

export function isProblem(v: unknown): v is Problem {
  if (typeof v !== 'object' || v === null) return false
  const p = v as Record<string, unknown>
  return typeof p.type === 'string' && typeof p.title === 'string' && typeof p.status === 'number' &&
    typeof p.reason_code === 'string' && typeof p.correlation_id === 'string'
}
