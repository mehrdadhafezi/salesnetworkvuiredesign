/*
 * Presentation formatting (SN-207 §4): Persian digits for display numerals.
 * A count is either a known non-negative number or explicitly unknown.
 * Unknown is NEVER rendered as zero; anything that is not a finite,
 * non-negative number is treated as unknown rather than coerced.
 */

/** A displayable count. `'unknown'` keeps "no evidence" distinct from 0. */
export type DsCount = number | 'unknown'

const faDigits = '۰۱۲۳۴۵۶۷۸۹'

/** Replace ASCII digits with Persian digits (fa() in the prototype). */
export function toFaDigits(value: string | number): string {
  return String(value).replace(/\d/g, d => faDigits[Number(d)] ?? d)
}

/** Grouped Persian number (num() in the prototype). */
export function formatFaNumber(n: number): string {
  return new Intl.NumberFormat('fa-IR').format(n)
}

export function isKnownCount(c: DsCount | null | undefined): c is number {
  return typeof c === 'number' && Number.isFinite(c) && c >= 0
}

/** Visible text for a count: Persian digits when known, «—» when unknown. */
export function countText(c: DsCount): string {
  return isKnownCount(c) ? formatFaNumber(c) : '—'
}

/** Accessible text for a count. */
export function countLabel(c: DsCount): string {
  return isKnownCount(c) ? formatFaNumber(c) : 'تعداد نامعلوم'
}
