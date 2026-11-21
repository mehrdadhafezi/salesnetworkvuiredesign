/**
 * RM-P03 activity timeline model (presentation only, no business rule).
 *
 * Every event keeps the timestamp exactly as its source delivered it and
 * says which CLOCK it was read from. Three clocks exist in the current API
 * and they are never compared with each other:
 *
 *  - LEGACY_SOURCE: a raw legacy wall-clock text (e.g. `created_at` of the
 *    legacy database). Its time zone is not recorded, so it is never turned
 *    into an instant; it is only ordered against other legacy wall-clock
 *    texts, and only when its format is recognised.
 *  - TARGET_UTC: an RFC 3339 instant written by the target system when the
 *    event itself happened (e.g. a contact outcome command).
 *  - IMPORT_OBSERVED: an RFC 3339 instant at which the shadow import
 *    observed something. It is not when the event happened.
 *
 * Anything else (absent, empty or an unrecognised raw format) has no
 * comparable timestamp and is kept apart, in source order.
 */

export type Clock = 'LEGACY_SOURCE' | 'TARGET_UTC' | 'IMPORT_OBSERVED'

/** How finely a timestamp was recorded. A coarser value is an interval. */
export type Precision = 'day' | 'minute' | 'second' | 'fraction'

export type EventTime =
  /** A recognised legacy wall-clock text. `start`/`end` are wall-clock µs used for ordering only. */
  | { kind: 'legacy'; raw: string; start: number; end: number; precision: Precision }
  /** An RFC 3339 instant from the server. */
  | { kind: 'instant'; clock: 'TARGET_UTC' | 'IMPORT_OBSERVED'; iso: string; start: number; end: number; precision: Precision }
  /** A raw value is present but its format is not recognised (never guessed). */
  | { kind: 'unparsed'; raw: string }
  /** The source has no timestamp for this event. */
  | { kind: 'missing' }

export function clockOf(t: EventTime): Clock | null {
  if (t.kind === 'legacy') return 'LEGACY_SOURCE'
  if (t.kind === 'instant') return t.clock
  return null
}

const MICRO: Record<Precision, number> = { day: 86_400_000_000, minute: 60_000_000, second: 1_000_000, fraction: 0 }

function fractionMicros(frac: string | undefined): { micros: number; width: number } {
  if (!frac) return { micros: 0, width: 0 }
  const digits = frac.slice(0, 6)
  return { micros: Number(digits.padEnd(6, '0')), width: 10 ** (6 - digits.length) }
}

/** Validates calendar fields (rejects e.g. 2024-02-30 and MySQL zero dates). */
function wallClockMs(y: number, mo: number, d: number, h: number, mi: number, s: number): number | null {
  if (mo < 1 || mo > 12 || d < 1 || d > 31 || h > 23 || mi > 59 || s > 59 || y < 1) return null
  const ms = Date.UTC(y, mo - 1, d, h, mi, s)
  const back = new Date(ms)
  if (back.getUTCFullYear() !== y || back.getUTCMonth() !== mo - 1 || back.getUTCDate() !== d) return null
  return ms
}

const LEGACY = /^(\d{4})-(\d{2})-(\d{2})(?:[ T](\d{2}):(\d{2})(?::(\d{2})(?:\.(\d{1,9}))?)?)?$/

/**
 * Reads a raw legacy wall-clock text. Only `YYYY-MM-DD`, `YYYY-MM-DD HH:MM`,
 * `YYYY-MM-DD HH:MM:SS` and `…SS.fraction` (space or `T`) are recognised;
 * any other text is `unparsed`, null or empty is `missing`. No zone is
 * assumed and no value is ever converted to an instant.
 */
export function legacyTime(raw: string | null | undefined): EventTime {
  if (raw === null || raw === undefined || raw.trim() === '') return { kind: 'missing' }
  const m = LEGACY.exec(raw.trim())
  if (!m) return { kind: 'unparsed', raw }
  const [, y, mo, d, h, mi, s, f] = m
  const ms = wallClockMs(Number(y), Number(mo), Number(d), Number(h ?? 0), Number(mi ?? 0), Number(s ?? 0))
  if (ms === null) return { kind: 'unparsed', raw }
  const precision: Precision = h === undefined ? 'day' : s === undefined ? 'minute' : f === undefined ? 'second' : 'fraction'
  const { micros, width } = fractionMicros(f)
  const start = ms * 1000 + micros
  return { kind: 'legacy', raw, start, end: start + (precision === 'fraction' ? width : MICRO[precision]), precision }
}

const INSTANT = /^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2})(?:\.(\d{1,9}))?(Z|[+-]\d{2}:\d{2})$/

/** Reads an RFC 3339 instant from the server; anything else is unparsed / missing. */
export function instantTime(iso: string | null | undefined, clock: 'TARGET_UTC' | 'IMPORT_OBSERVED'): EventTime {
  if (iso === null || iso === undefined || iso === '') return { kind: 'missing' }
  const m = INSTANT.exec(iso)
  if (!m) return { kind: 'unparsed', raw: iso }
  const [, base, f, zone] = m
  const ms = Date.parse(`${base}${zone}`)
  if (Number.isNaN(ms)) return { kind: 'unparsed', raw: iso }
  const { micros, width } = fractionMicros(f)
  const start = ms * 1000 + micros
  return { kind: 'instant', clock, iso, start, end: start + (f === undefined ? MICRO.second : width), precision: f === undefined ? 'second' : 'fraction' }
}

/* ------------------------------------------------------------------ */

/**
 * How the server orders a source. Source order is authoritative for that
 * source and is always preserved in the per-source view.
 */
export type SourceOrder =
  /** Ascending legacy insertion ordinal (case history, custody events). */
  | 'SOURCE_ORDINAL'
  /** Newest first by the event's own instant (contact outcomes). */
  | 'NEWEST_FIRST'
  /** Fields of one record, listed in a fixed order (not a history). */
  | 'RECORD_FIELDS'
  /** The order of a server list page (not a chronology). */
  | 'LIST_ORDER'

/** Per-source coverage of what is currently loaded. */
export type Coverage =
  /** Every event of the source is loaded (first page, no further page, not capped). */
  | 'COMPLETE'
  /** One page of a cursor-paged source; other pages exist or were visited. */
  | 'PAGE'
  /** The server returns at most `cap` events without a cursor; reaching it means completeness is unknown. */
  | 'CAPPED'

export interface Paging { depth: number; hasMore: boolean }

export function coverageOf(count: number, paging: Paging | undefined, cap: number | undefined): Coverage {
  if (cap !== undefined && count >= cap) return 'CAPPED'
  if (paging && (paging.depth > 0 || paging.hasMore)) return 'PAGE'
  return 'COMPLETE'
}

/** The minimum an event needs for ordering; the UI adds presentation. */
export interface OrderedInput { key: string; source: string; sourceRank: number; index: number; time: EventTime; sourceOrder: SourceOrder }

export type OrderNote =
  /** Same instant and precision as another event in the group. */
  | 'TIE'
  /** Its interval overlaps another event recorded at a different precision. */
  | 'PRECISION_OVERLAP'
  /** The time order disagrees with this source's own insertion order. */
  | 'SOURCE_ORDER_DIFFERS'

export interface Ordered<T extends OrderedInput> { item: T; notes: OrderNote[] }

export interface ClockGroup<T extends OrderedInput> { clock: Clock; items: Ordered<T>[] }

export interface Combined<T extends OrderedInput> {
  /** One group per clock, oldest first; groups are never interleaved. */
  groups: ClockGroup<T>[]
  /** Events without a comparable timestamp, in source order (source rank, then source index). */
  unordered: T[]
}

const CLOCK_ORDER: Clock[] = ['LEGACY_SOURCE', 'TARGET_UTC', 'IMPORT_OBSERVED']

/**
 * Deterministic ordering within one clock: interval start, then interval
 * end (a finer timestamp before the coarser interval that starts with it),
 * then the fixed source rank, then the source's own index. Ties and
 * precision overlaps are reported, never hidden; nothing is ordered across
 * clocks.
 */
export function combine<T extends OrderedInput>(inputs: T[]): Combined<T> {
  const unordered = inputs.filter(i => clockOf(i.time) === null)
    .sort((a, b) => a.sourceRank - b.sourceRank || a.index - b.index)
  const groups: ClockGroup<T>[] = []
  for (const clock of CLOCK_ORDER) {
    const of = inputs.filter(i => clockOf(i.time) === clock)
    if (of.length === 0) continue
    const span = (i: T) => i.time as Extract<EventTime, { start: number }>
    of.sort((a, b) => span(a).start - span(b).start || span(a).end - span(b).end || a.sourceRank - b.sourceRank || a.index - b.index)
    const items: Ordered<T>[] = of.map((item, pos) => {
      const notes = new Set<OrderNote>()
      const t = span(item)
      for (const other of of) {
        if (other === item) continue
        const o = span(other)
        if (o.start === t.start && o.end === t.end) notes.add('TIE')
        else if (o.start < t.end && t.start < o.end) notes.add('PRECISION_OVERLAP')
      }
      // Insertion-ordered sources: flag an event placed before an event the source inserted earlier.
      if (item.sourceOrder === 'SOURCE_ORDINAL') {
        for (let j = pos + 1; j < of.length; j++) {
          const later = of[j]
          if (later && later.source === item.source && later.index < item.index) { notes.add('SOURCE_ORDER_DIFFERS'); break }
        }
        for (let j = 0; j < pos; j++) {
          const earlier = of[j]
          if (earlier && earlier.source === item.source && earlier.index > item.index) { notes.add('SOURCE_ORDER_DIFFERS'); break }
        }
      }
      return { item, notes: [...notes] }
    })
    groups.push({ clock, items })
  }
  return { groups, unordered }
}
