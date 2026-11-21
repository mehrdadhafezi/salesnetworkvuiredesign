/**
 * Presentation-only formatting of an RFC 3339 UTC instant in the Persian
 * (Jalali) calendar and a display zone. The SPA never stores or sends Jalali
 * text or a zone; API timestamps stay canonical UTC.
 */
export function formatInstant(utc: string, displayZone: string): string {
  const d = new Date(utc)
  if (Number.isNaN(d.getTime())) return '—'
  return new Intl.DateTimeFormat('fa-IR-u-ca-persian', {
    timeZone: displayZone,
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(d)
}

/**
 * Presentation-only Jalali rendering of a legacy wall-clock value whose time
 * zone was never recorded. The wall-clock fields are shown unchanged in the
 * Persian calendar (no zone shift is applied or implied); `day` precision
 * shows the date only. `wallClockMs` is UTC-based arithmetic over the raw
 * fields, never an instant.
 */
export function formatWallClock(wallClockMs: number, dateOnly: boolean): string {
  const d = new Date(wallClockMs)
  if (Number.isNaN(d.getTime())) return '—'
  return new Intl.DateTimeFormat('fa-IR-u-ca-persian', {
    timeZone: 'UTC',
    dateStyle: 'medium',
    ...(dateOnly ? {} : { timeStyle: 'medium' }),
  }).format(d)
}
