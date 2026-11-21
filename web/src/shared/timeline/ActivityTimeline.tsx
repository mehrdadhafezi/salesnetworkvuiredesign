import './timeline.css'
import { useState, type ReactNode } from 'react'
import { Link } from 'react-router'
import type { PageInfo } from '../api'
import type { ReadState } from '../read/load'
import { ReadStates } from '../read/ReadStates'
import { Banner, DsIcon, StatusIndicator, TonePill, type BannerKind, type DsIconName } from '../ui/ds'
import { Button, StateBlock, type Tone } from '../ui/primitives'
import { formatInstant, formatWallClock } from '../ui/time'
import { combine, coverageOf, type Clock, type Coverage, type EventTime, type OrderNote, type OrderedInput, type SourceOrder } from './model'

/**
 * RM-P03 shared, read-only activity timeline. It renders events that each
 * page has already read through its own module endpoints; it never reads,
 * joins or infers anything itself. Each source keeps its own authorization
 * state, server order, server pagination and coverage. A source that is
 * denied, blocked, gated or failed contributes no event anywhere (its
 * read state holds no data), and is shown as such.
 *
 * Two views: per source (default; server order, server pager) and a
 * combined view that orders only timestamps of the same clock (see model.ts),
 * keeps every clock in its own group and lists events without a comparable
 * timestamp apart. The combined view only ever contains the loaded pages
 * and says so.
 */

export interface TimelineEvent {
  /** Source-native id (unique within the source). */
  id: string
  /** Machine event type, e.g. `case.state_changed`. */
  type: string
  /** Human-readable Persian label. */
  label: string
  icon: DsIconName
  time: EventTime
  /** Other timestamps of the same record (e.g. import observation), shown in details. */
  otherTimes?: { label: string; time: EventTime }[] | undefined
  /** Raw historical transition, when the source records one. */
  transition?: { from: string | null | undefined; to: string | null | undefined } | undefined
  /** undefined: the source has no actor; null: the source supports one but none is recorded. */
  actor?: ReactNode
  /** Related record; `href` only for a verified, authorized relationship. */
  related?: { label: string; href?: string | undefined } | undefined
  /** Source provenance of this event (spec · key, batch, …). */
  provenance: string
  /** Further source facts shown on the event (raw values, subjects, states). */
  facts?: { label: string; value: ReactNode }[] | undefined
  flags?: { tone: Tone; icon: DsIconName; label: string }[] | undefined
  /** Expandable details; a function is rendered (and so may read) only once opened. */
  details?: ReactNode | (() => ReactNode)
}

export type SourceRead =
  | ReadState<TimelineEvent[]>
  /** Not requested: the server's allowed_actions hint says the read is not allowed. */
  | { status: 'locked'; title: string }
  /** Not requested yet: depends on another read that has not succeeded. */
  | { status: 'awaiting'; title: string }

/** Maps the data of a settled read to events; every non-data state passes through unchanged (nothing leaks). */
export function mapRead<T>(r: ReadState<T>, f: (d: T) => TimelineEvent[]): SourceRead {
  return r.status === 'ok' ? { ...r, data: f(r.data) } : r
}

export interface TimelineSource {
  id: string
  label: string
  icon: DsIconName
  /** Where the events come from (module, table / record). */
  provenance: string
  order: SourceOrder
  read: SourceRead
  blockedDetail: string
  /** Server cursor paging, when the source is paged. */
  paging?: { depth: number; page: PageInfo; back: () => void; next: (cursor: string) => void } | undefined
  /** Server-side maximum for an unpaged source. */
  cap?: number | undefined
  notes?: { kind: BannerKind; text: string }[] | undefined
  footer?: ReactNode
}

const orderLabel: Record<SourceOrder, string> = {
  SOURCE_ORDINAL: 'ترتیب درج در منبع (قدیمی‌ترین ابتدا؛ ترتیب سرور)',
  NEWEST_FIRST: 'جدیدترین ابتدا (ترتیب سرور)',
  RECORD_FIELDS: 'فیلدهای زمانی رکورد؛ سابقه تغییرات نیست',
  LIST_ORDER: 'ترتیب فهرست سرور؛ ترتیب زمانی نیست',
}

const clockTitle: Record<Clock, string> = {
  LEGACY_SOURCE: 'زمان‌های منبع قدیمی',
  TARGET_UTC: 'زمان‌های سامانه جدید (لحظه رخداد)',
  IMPORT_OBSERVED: 'زمان‌های مشاهده در واردسازی',
}

const clockNote: Record<Clock, string> = {
  LEGACY_SOURCE: 'زمان دیواری خام سامانه قدیمی؛ منطقه زمانی آن ثبت نشده و به لحظه تبدیل نمی‌شود.',
  TARGET_UTC: 'لحظه ثبت رویداد در سامانه جدید (UTC، نمایش در منطقه زمانی نشست).',
  IMPORT_OBSERVED: 'زمانی که واردسازی سایه تغییر را دید؛ زمان رخداد نیست.',
}

const precisionLabel = { day: 'روز', minute: 'دقیقه', second: 'ثانیه', fraction: 'کسر ثانیه' } as const

const noteLabel: Record<OrderNote, { state: 'unknown' | 'partial' | 'conflict'; label: string }> = {
  TIE: { state: 'unknown', label: 'هم‌زمان با رویداد دیگر؛ ترتیب بر اساس منبع' },
  PRECISION_OVERLAP: { state: 'partial', label: 'دقت زمانی متفاوت؛ ترتیب نسبی نامطمئن' },
  SOURCE_ORDER_DIFFERS: { state: 'conflict', label: 'با ترتیب درج منبع مغایر است' },
}

function coverageText(c: Coverage, depth: number, cap: number | undefined): string {
  if (c === 'COMPLETE') return 'همه رویدادهای این منبع بارگذاری شده است'
  if (c === 'CAPPED') return `حداکثر ${cap ?? ''} رویداد بازگردانده شد؛ کامل بودن نامعلوم است`
  return `فقط صفحه ${depth + 1} بارگذاری شده است`
}

/** Coverage of a loaded source, or null when it has no data. */
function sourceCoverage(s: TimelineSource): Coverage | null {
  if (s.read.status !== 'ok') return null
  return coverageOf(s.read.data.length, s.paging ? { depth: s.paging.depth, hasMore: s.paging.page.has_more } : undefined, s.cap)
}

function statusPill(s: TimelineSource): ReactNode {
  switch (s.read.status) {
    case 'loading': return <TonePill tone="slate" icon="clock" label="در حال دریافت" />
    case 'denied': return <TonePill tone="slate" icon="lock" label="دسترسی ندارید" />
    case 'locked': return <TonePill tone="slate" icon="lock" label="دسترسی ندارید (درخواست نشد)" />
    case 'blocked': return <TonePill tone="red" icon="ban" label="مسدود" />
    case 'failed': return <TonePill tone="red" icon="xCircle" label="خطا" />
    case 'awaiting': return <TonePill tone="slate" icon="dashed" label="در انتظار رکورد اصلی" />
    case 'ok': {
      const c = sourceCoverage(s)
      return c === 'COMPLETE' ? <TonePill tone="green" icon="check" label="همه بارگذاری شد" />
        : <StatusIndicator state={c === 'CAPPED' ? 'unknown' : 'partial'} label={c === 'CAPPED' ? 'کامل بودن نامعلوم' : 'بخشی (صفحه‌بندی)'} />
    }
  }
}

export function ActivityTimeline({ label, coverage, sources, zone }: { label: string; coverage: string; sources: TimelineSource[]; zone: string }) {
  const [view, setView] = useState<'source' | 'combined'>('source')
  return (
    <div className="tl" aria-label={label} role="group">
      <p className="tl-coverage" role="note"><strong>پوشش: </strong>{coverage}</p>
      <ul className="tl-sources" aria-label="منابع خط زمانی">
        {sources.map(s => (
          <li key={s.id}><DsIcon name={s.icon} /> <span>{s.label}</span> {statusPill(s)}</li>
        ))}
      </ul>
      {sources.length > 1 && (
        <div className="tl-views" role="group" aria-label="نمای خط زمانی">
          <Button small aria-pressed={view === 'source'} onClick={() => setView('source')}>به تفکیک منبع</Button>
          <Button small aria-pressed={view === 'combined'} onClick={() => setView('combined')}>ترکیب زمانی</Button>
        </div>
      )}
      {view === 'source' || sources.length < 2
        ? sources.map(s => <SourceLane key={s.id} s={s} zone={zone} />)
        : <CombinedView sources={sources} zone={zone} />}
    </div>
  )
}

function SourceLane({ s, zone }: { s: TimelineSource; zone: string }) {
  const cov = sourceCoverage(s)
  return (
    <section className="tl-lane" aria-label={s.label}>
      <h3 className="tl-lane-title"><DsIcon name={s.icon} /> {s.label}</h3>
      <dl className="tl-meta">
        <div><dt>منبع</dt><dd>{s.provenance}</dd></div>
        <div><dt>ترتیب</dt><dd>{orderLabel[s.order]}</dd></div>
        {cov && <div><dt>پوشش</dt><dd>{coverageText(cov, s.paging?.depth ?? 0, s.cap)}</dd></div>}
      </dl>
      {s.read.status === 'locked' ? <StateBlock kind="locked" title={s.read.title} />
        : s.read.status === 'awaiting' ? <StateBlock kind="unknown" title={s.read.title} />
          : <ReadStates state={s.read} what={s.label} blockedDetail={s.blockedDetail} />}
      {s.read.status === 'ok' && (
        <>
          {s.notes?.map(n => <Banner key={n.text} kind={n.kind}>{n.text}</Banner>)}
          {s.read.data.length === 0
            ? <StateBlock kind="empty" title="رویدادی در این منبع ثبت نشده است" />
            : (
              <ol className="tl-list" aria-label={`رویدادهای ${s.label}`}>
                {s.read.data.map(e => <EventItem key={e.id} e={e} zone={zone} />)}
              </ol>
            )}
          {s.paging && (s.paging.depth > 0 || s.paging.page.has_more) && <LanePager p={s.paging} label={s.label} />}
        </>
      )}
      {s.footer}
    </section>
  )
}

function LanePager({ p, label }: { p: NonNullable<TimelineSource['paging']>; label: string }) {
  const next = p.page.next_cursor
  return (
    <nav className="tl-pager" aria-label={`صفحه‌بندی ${label}`}>
      <span className="muted">{`صفحه ${p.depth + 1}`}</span>
      <Button small disabled={p.depth === 0} onClick={p.back}>صفحه قبل</Button>
      <Button small disabled={!p.page.has_more || !next} onClick={() => { if (next) p.next(next) }}>صفحه بعد</Button>
    </nav>
  )
}

interface Entry extends OrderedInput { e: TimelineEvent; sourceLabel: string }

function CombinedView({ sources, zone }: { sources: TimelineSource[]; zone: string }) {
  const entries: Entry[] = []
  sources.forEach((s, rank) => {
    if (s.read.status !== 'ok') return
    s.read.data.forEach((e, index) => entries.push({ key: `${s.id}:${e.id}`, source: s.id, sourceRank: rank, index, time: e.time, sourceOrder: s.order, e, sourceLabel: s.label }))
  })
  const { groups, unordered } = combine(entries)
  const missing = sources.filter(s => s.read.status !== 'ok')
  const partial = sources.filter(s => { const c = sourceCoverage(s); return c !== null && c !== 'COMPLETE' })
  return (
    <section className="tl-combined" aria-label="ترکیب زمانی">
      {missing.length > 0 || partial.length > 0
        ? (
          <Banner kind="incomplete">
            {'این ترکیب کامل نیست. '}
            {missing.length > 0 && `منابع بدون داده (بدون دسترسی، مسدود، خطا یا در انتظار) در ترکیب نیستند: ${missing.map(s => s.label).join('، ')}. `}
            {partial.length > 0 && `فقط صفحه بارگذاری‌شده از این منابع آمده است: ${partial.map(s => s.label).join('، ')}.`}
          </Banner>
        )
        : <Banner kind="info">همه رویدادهای منابع بالا بارگذاری شده است؛ ترتیب فقط درون هر گروه زمانی معتبر است.</Banner>}
      <p className="muted tl-rule" role="note">
        ترتیب درون هر گروه: زمان (قدیمی‌ترین ابتدا)، سپس دقت زمان، سپس ترتیب ثابت منابع، سپس ترتیب خود منبع. گروه‌ها با هم مقایسه یا ادغام نمی‌شوند.
      </p>
      {groups.length === 0 && unordered.length === 0 && <StateBlock kind="empty" title="رویدادی برای ترکیب وجود ندارد" />}
      {groups.map(g => (
        <section key={g.clock} className="tl-group" aria-label={clockTitle[g.clock]}>
          <h3 className="tl-lane-title"><DsIcon name="clock" /> {clockTitle[g.clock]}</h3>
          <p className="muted tl-rule">{clockNote[g.clock]}</p>
          <ol className="tl-list" aria-label={`رویدادهای ${clockTitle[g.clock]}`}>
            {g.items.map(({ item, notes }) => <EventItem key={item.key} e={item.e} zone={zone} source={item.sourceLabel} notes={notes} />)}
          </ol>
        </section>
      ))}
      {unordered.length > 0 && (
        <section className="tl-group" aria-label="رویدادهای بدون زمان قابل مقایسه">
          <h3 className="tl-lane-title"><DsIcon name="dashed" /> رویدادهای بدون زمان قابل مقایسه</h3>
          <p className="muted tl-rule">زمان این رویدادها ثبت نشده یا قالب آن شناخته نشد؛ به ترتیب منبع فهرست شده‌اند و در ترتیب زمانی جایی ندارند.</p>
          <ol className="tl-list" aria-label="رویدادهای بدون زمان قابل مقایسه">
            {unordered.map(x => <EventItem key={x.key} e={x.e} zone={zone} source={x.sourceLabel} />)}
          </ol>
        </section>
      )}
    </section>
  )
}

function EventItem({ e, zone, source, notes }: { e: TimelineEvent; zone: string; source?: string; notes?: OrderNote[] }) {
  const [open, setOpen] = useState(false)
  const hasDetails = e.details !== undefined || (e.otherTimes !== undefined && e.otherTimes.length > 0)
  return (
    <li className="tl-event" data-type={e.type}>
      <span className="tl-dot" aria-hidden="true"><DsIcon name={e.icon} /></span>
      <div className="tl-body">
        <div className="tl-head">
          <strong className="tl-label">{e.label}</strong>
          {source && <span className="tl-src">{source}</span>}
          {e.flags?.map(f => <TonePill key={f.label} tone={f.tone} icon={f.icon} label={f.label} />)}
          {notes?.map(n => <StatusIndicator key={n} state={noteLabel[n].state} label={noteLabel[n].label} />)}
        </div>
        <TimeText t={e.time} zone={zone} />
        {e.transition && (
          <p className="tl-line"><span className="tl-k">تغییر:</span>{' '}
            <span className="tl-ltr">{`${e.transition.from ?? '—'} → ${e.transition.to ?? '—'}`}</span></p>
        )}
        {e.actor !== undefined && (
          <p className="tl-line"><span className="tl-k">اقدام‌کننده:</span> {e.actor ?? <span className="muted">ثبت نشده</span>}</p>
        )}
        {e.related && (
          <p className="tl-line"><span className="tl-k">رکورد مرتبط:</span>{' '}
            {e.related.href
              ? (e.related.href.startsWith('#') ? <a href={e.related.href}>{e.related.label}</a> : <Link to={e.related.href}>{e.related.label}</Link>)
              : <span>{e.related.label}</span>}</p>
        )}
        {e.facts?.map(f => <p key={f.label} className="tl-line"><span className="tl-k">{f.label}:</span> {f.value}</p>)}
        <p className="tl-line muted"><span className="tl-k">منشأ:</span> <span className="tl-ltr">{e.provenance}</span> <span className="tl-ltr">{`(${e.type})`}</span></p>
        {hasDetails && (
          <details className="tl-details" onToggle={ev => setOpen(ev.currentTarget.open)}>
            <summary>جزئیات</summary>
            {open && (
              <>
                {e.otherTimes?.map(o => <div key={o.label} className="tl-line"><span className="tl-k">{o.label}:</span> <TimeText t={o.time} zone={zone} inline /></div>)}
                {typeof e.details === 'function' ? e.details() : e.details}
              </>
            )}
          </details>
        )}
      </div>
    </li>
  )
}

/** A timestamp exactly as its source gave it: never invented, never shifted across clocks. */
export function TimeText({ t, zone, inline = false }: { t: EventTime; zone: string; inline?: boolean }) {
  const Tag = inline ? 'span' : 'p'
  switch (t.kind) {
    case 'missing':
      return <Tag className="tl-time"><StatusIndicator state="unknown" label="زمان ثبت نشده" /></Tag>
    case 'unparsed':
      return <Tag className="tl-time"><StatusIndicator state="unknown" label="قالب زمان شناخته نشد" /> <span className="tl-ltr tl-raw">{t.raw}</span></Tag>
    case 'legacy': {
      const ms = Math.floor(t.start / 1000)
      const iso = new Date(ms).toISOString()
      return (
        <Tag className="tl-time">
          <time dateTime={t.precision === 'day' ? iso.slice(0, 10) : iso.slice(0, 19)}>{formatWallClock(ms, t.precision === 'day')}</time>
          {' '}<span className="tl-ltr tl-raw">{t.raw}</span>
          <span className="muted">{` · زمان منبع قدیمی، منطقه زمانی ثبت نشده · دقت: ${precisionLabel[t.precision]}`}</span>
        </Tag>
      )
    }
    case 'instant':
      return (
        <Tag className="tl-time">
          <time dateTime={t.iso}>{formatInstant(t.iso, zone)}</time>
          <span className="muted">{t.clock === 'TARGET_UTC' ? ' · زمان رخداد در سامانه جدید' : ' · زمان مشاهده در واردسازی (نه زمان رخداد)'}</span>
        </Tag>
      )
  }
}
