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
