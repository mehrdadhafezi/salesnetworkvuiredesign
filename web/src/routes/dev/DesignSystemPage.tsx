import { useCallback, useRef, useState, type MouseEvent } from 'react'
import { useSearchParams } from 'react-router'
import type { Outcome } from '../../shared/api'
import { outcomePresentation } from '../../shared/ui/outcome'
import { StatusPill, type Tone } from '../../shared/ui/primitives'
import {
  AppHeader, Banner, ContextHint, Drawer, DrawerSection, DsAppShell, Freshness, Glossary, HeaderIconButton,
  HelpCenterButton, MenuButton, Metric, MetricGroup, MobileBottomNavigation, PageHead, Panel, PanelHeader, Popover,
  QueueTabs, RoleNavigation, ShortcutList, StatePanel, StatusIndicator, Toolbar, dataStatePresentation,
  type DataState, type DsNavItem, type DsQueue, type MenuEntry, type PanelStateKind,
} from '../../shared/ui/ds'
import './showcase.css'

/*
 * DEV-ONLY design-system showcase (RM-E01). Mounted only when import.meta.env.DEV && config.devTools;
 * production bundles never contain it (web-ci checks the bundle).
 * All content is SYNTHETIC. Labels of the navigation and queues reuse the frozen Seller V1
 * prototype copy for visual comparison only; counts are invented and nothing here reads or
 * writes business data. Every "action" only updates a local status line.
 * Deep links for screenshots: ?theme=light|dim|dark&density=comfortable|compact&drawer=open|wide&drawerState=content|loading|error
 */

const NAV: DsNavItem[] = [
  { id: 'leads', label: 'شماره‌های من', href: '#leads', icon: 'users', count: 12, alert: true, mobileLabel: 'شماره‌ها' },
  { id: 'conversions', label: 'آماده‌های تبدیل', href: '#conversions', icon: 'swap', count: 3, mobileLabel: 'تبدیل' },
  { id: 'repeat', label: 'نیاز به اقدام مجدد', href: '#repeat', icon: 'repeat', count: 0 },
  { id: 'invoices', label: 'فاکتورها', href: '#invoices', icon: 'receipt', count: 'unknown', mobileLabel: 'فاکتورها' },
  { id: 'behavior', label: 'رفتار مشتریان', href: '#behavior', icon: 'activity', group: 'more' },
  { id: 'request', label: 'درخواست شماره', href: '#request', icon: 'plus', group: 'more' },
  { id: 'wallet', label: 'کیف پول', href: '#wallet', icon: 'wallet', group: 'more', mobileLabel: 'کیف پول' },
]

const QUEUES: DsQueue[] = [
  { id: 'none', label: 'بدون وضعیت', tone: 'neutral', icon: 'dashed', count: 4, shortcut: '1' },
  { id: 'no_answer', label: 'جواب نداده', tone: 'amber', icon: 'phoneOff', count: 2, shortcut: '2' },
  { id: 'callback', label: 'تماس مجدد', tone: 'blue', icon: 'clock', count: 1, shortcut: '3' },
  { id: 'not_purchased', label: 'عدم خرید', tone: 'red', icon: 'xCircle', count: 0, shortcut: '4' },
  { id: 'duplicate', label: 'تکراری', tone: 'slate', icon: 'copy', count: 'unknown', shortcut: '5' },
  { id: 'invoiced', label: 'پیش‌فاکتور', tone: 'violet', icon: 'file', count: 1, shortcut: '6' },
  { id: 'all', label: 'همه', tone: 'neutral', count: 'unknown', shortcut: '0' },
]

const PANEL_STATES: { kind: PanelStateKind; title: string; detail?: string }[] = [
  { kind: 'loading', title: 'در حال بارگذاری فهرست نمونه' },
  { kind: 'empty', title: 'موردی در این صف نیست', detail: 'صف خالی است؛ این یک واقعیت است، نه خطا.' },
  { kind: 'noresult', title: 'نتیجه‌ای پیدا نشد', detail: 'فیلتر همه موارد را کنار گذاشت.' },
  { kind: 'error', title: 'دریافت فهرست انجام نشد', detail: 'خطای شناخته‌شده؛ می‌توان دوباره تلاش کرد.' },
  { kind: 'locked', title: 'دسترسی به این بخش فعال نیست', detail: 'نمایش‌ندادن داده به معنی نبودن داده نیست.' },
  { kind: 'stale', title: 'داده قدیمی است', detail: 'اقدام‌ها هنگام ثبت دوباره بررسی می‌شوند.' },
  { kind: 'incomplete', title: 'پوشش ناقص', detail: 'جمع نهایی نیست و صفر هم نیست.' },
  { kind: 'conflict', title: 'منابع با هم تعارض دارند', detail: 'پیش از اقدام تطبیق دهید.' },
  { kind: 'partial', title: 'بخشی انجام شد', detail: 'برخی موارد اعمال شد و برخی نه.' },
  { kind: 'retry', title: 'ناموفق — تلاش دوباره امن است', detail: 'شکست قطعی و قابل تکرار.' },
  { kind: 'unknown', title: 'نتیجه نامعلوم', detail: 'ممکن است ثبت شده باشد؛ پیش از هر تکرار تطبیق دهید.' },
  { kind: 'offline', title: 'اتصال برقرار نیست', detail: 'اقدام‌ها تا برقراری اتصال متوقف‌اند.' },
]

const TONES: { tone: Tone; meaning: string }[] = [
  { tone: 'green', meaning: 'تکمیل / تأیید' }, { tone: 'teal', meaning: 'آماده / واجد شرایط' },
  { tone: 'amber', meaning: 'نیاز به توجه / نامعلوم' }, { tone: 'orange', meaning: 'تعارض / بخشی' },
  { tone: 'red', meaning: 'ناموفق / مسدود' }, { tone: 'violet', meaning: 'پیش‌نویس / مشروط' },
  { tone: 'slate', meaning: 'محدود / غیرفعال' }, { tone: 'blue', meaning: 'اطلاعاتی' }, { tone: 'neutral', meaning: 'خنثی' },
]

/* SN-207 §16 core glossary — wording reused verbatim from prototype/shared/help-engine.js CORE_TERMS. */
const CORE_TERMS = [
  { term: 'مسئول فعلی', definition: 'پرونده الان نزد کیست (Current Custody).' },
  { term: 'مالک اولیه', definition: 'نخستین دریافت‌کننده پرونده؛ با تغییر مسئول عوض نمی‌شود.' },
  { term: 'اقدام بعدی با', definition: 'چه کسی یا کدام صف باید گام بعدی همین گردش‌کار را انجام دهد؛ طبق گردش‌کار همان حوزه تعیین می‌شود و حدس زده نمی‌شود.' },
  { term: 'عامل رویداد', definition: 'کسی که رویداد (مثلاً انتقال یا صدور) را انجام داده؛ مالک فعلی یا مالک اعتبار نیست.' },
  { term: 'مالک اعتبار', definition: 'طبق قوانین مالی فعلی تعیین می‌شود؛ تخصیص، برگشت یا تغییر والد آن را عوض نمی‌کند.' },
  { term: 'نامعلوم', definition: 'پاسخ نرسید و ممکن است ثبت شده باشد؛ پیش از تکرار باید تطبیق شود.' },
]

type Theme = 'light' | 'dim' | 'dark'
type Density = 'comfortable' | 'compact'
type DrawerContent = 'content' | 'loading' | 'error'

function pick<T extends string>(v: string | null, allowed: readonly T[], fallback: T): T {
  return allowed.includes(v as T) ? (v as T) : fallback
}

export function DesignSystemPage() {
  const [params] = useSearchParams()
  const [theme, setTheme] = useState<Theme>(() => pick(params.get('theme'), ['light', 'dim', 'dark'] as const, 'light'))
  const [density, setDensity] = useState<Density>(() => pick(params.get('density'), ['comfortable', 'compact'] as const, 'comfortable'))
  const [view, setView] = useState('leads')
  const [queue, setQueue] = useState('none')
  const [drawer, setDrawer] = useState<null | 'normal' | 'wide'>(() => {
    const d = params.get('drawer')
    return d === 'open' ? 'normal' : d === 'wide' ? 'wide' : null
  })
  const [drawerContent, setDrawerContent] = useState<DrawerContent>(() => pick(params.get('drawerState'), ['content', 'loading', 'error'] as const, 'content'))
  const [status, setStatus] = useState('هنوز اقدامی انجام نشده است.')
  const [popOpen, setPopOpen] = useState(false)
  const popBtn = useRef<HTMLButtonElement>(null)
  const closePop = useCallback(() => setPopOpen(false), [])
  const closeDrawer = useCallback(() => setDrawer(null), [])

  const demo = (what: string) => setStatus(`نمایشی: «${what}» — چیزی ثبت یا ارسال نشد.`)
  const current = NAV.find(n => n.id === view)
  const navigate = (item: DsNavItem, e?: MouseEvent<HTMLAnchorElement>) => { e?.preventDefault(); setView(item.id); demo(`رفتن به ${item.label}`) }

  const userMenu: MenuEntry[] = [
    { kind: 'heading', id: 'h-theme', label: 'پوسته' },
    { kind: 'item', id: 'light', label: 'روشن', current: theme === 'light', onSelect: () => setTheme('light') },
    { kind: 'item', id: 'dim', label: 'کم‌نور', current: theme === 'dim', onSelect: () => setTheme('dim') },
    { kind: 'item', id: 'dark', label: 'تیره', current: theme === 'dark', onSelect: () => setTheme('dark') },
    { kind: 'separator', id: 's1' },
    { kind: 'item', id: 'shortcuts', label: 'میانبرهای صفحه‌کلید', icon: 'key', onSelect: () => demo('میانبرها') },
    { kind: 'item', id: 'password', label: 'رمز من', icon: 'lock', disabled: true, disabledReason: 'در نمونه اجزا غیرفعال است.' },
    { kind: 'item', id: 'logout', label: 'خروج', icon: 'logout', danger: true, onSelect: () => demo('خروج') },
  ]

  const exampleMenu: MenuEntry[] = [
    { kind: 'item', id: 'view', label: 'مشاهده جزئیات', icon: 'file', onSelect: () => setDrawer('normal') },
    { kind: 'item', id: 'copy', label: 'کپی شناسه نمونه', icon: 'copy', onSelect: () => demo('کپی شناسه') },
    { kind: 'separator', id: 's' },
    { kind: 'item', id: 'denied', label: 'اقدام بدون مجوز', icon: 'lock', disabled: true, disabledReason: 'نمونه: دکمه دیده می‌شود ولی مجوز ندارد؛ پنهان نمی‌شود.' },
  ]

  const outcomes = Object.keys(outcomePresentation) as Outcome[]
  const states = Object.keys(dataStatePresentation) as DataState[]

  return (
    <div data-theme={theme} data-density={density} dir="rtl" lang="fa" className="ds-showcase-root">
      <DsAppShell
        header={
          <AppHeader
            brandMark="C"
            brandName="CRM"
            crumb={current?.label}
            tag="نمونه اجزا · داده ساختگی"
            mobileTitle={current?.label}
            search={{ label: 'جستجو و فرمان‌ها (Ctrl+K)', placeholder: 'جستجوی مشتری، فاکتور یا رفتن به…', shortcut: 'Ctrl K', onOpen: () => demo('پالت فرمان (به RM-E02 موکول است)') }}
            actions={
              <>
                <HeaderIconButton label="اعلان‌ها — ۲ مورد جدید (نمایشی)" icon="bell" badge="۲" onClick={() => demo('اعلان‌ها')} />
                <HelpCenterButton
                  subtitle="محتوای راهنما از نقش‌ها می‌آید؛ این‌جا فقط نمونه است."
                  topics={[
                    { id: 'terms', title: 'واژه‌ها و مفاهیم', summary: 'مسئول فعلی، مالک اولیه، نامعلوم و…', icon: 'info', onOpen: () => { document.getElementById('ds-help')?.scrollIntoView(); demo('واژه‌نامه') } },
                    { id: 'keys', title: 'میانبرهای صفحه‌کلید', summary: 'Esc، پیکان‌ها، Tab', icon: 'key', onOpen: () => { document.getElementById('ds-keyboard')?.scrollIntoView(); demo('میانبرها') } },
                  ]}
                />
                <MenuButton label="منوی کاربر و پوسته" className="ds-user" entries={userMenu}>
                  <span className="ds-avatar" aria-hidden="true">ن‌س</span>
                  <span className="ds-user-meta"><b>کاربر نمونه</b><span>نقش ساختگی</span></span>
                </MenuButton>
              </>
            }
          />
        }
        nav={<RoleNavigation items={NAV} currentId={view} label="بخش‌های پنل نمونه" onNavigate={navigate} />}
        bottomNav={<MobileBottomNavigation items={NAV} currentId={view} onNavigate={navigate} />}
        overlay={
          <Drawer
            open={drawer !== null}
            wide={drawer === 'wide'}
            onClose={closeDrawer}
            title="پرونده نمونه ۱۰۴ (ساختگی)"
            footerHint="Esc برای بستن"
            footer={
              <>
                <button type="button" className="btn btn-lg" onClick={closeDrawer}>بستن</button>
                <button type="button" className="btn btn-primary btn-lg" disabled aria-describedby="ds-dr-why">ثبت نتیجه</button>
              </>
            }
          >
            {drawerContent === 'loading'
              ? <StatePanel kind="loading" title="در حال بارگذاری جزئیات" />
              : drawerContent === 'error'
                ? <StatePanel kind="error" title="جزئیات بارگذاری نشد" detail="پنجره باز می‌ماند تا دوباره تلاش کنید." actions={<button type="button" className="btn btn-primary" onClick={() => setDrawerContent('content')}>تلاش مجدد</button>} />
                : (
                  <>
                    <DrawerSection title="وضعیت داده" aside="نمونه">
                      <p className="ds-sub">وضعیت‌ها همیشه برچسب + نماد + رنگ‌اند:</p>
                      <div className="ds-row-wrap"><StatusIndicator state="unknown" /><StatusIndicator state="stale" /><StatusIndicator state="incomplete" /></div>
                    </DrawerSection>
                    <DrawerSection title="چرا ثبت غیرفعال است؟">
                      <p id="ds-dr-why" className="ds-sub">در نمونه اجزا هیچ اقدامی ثبت نمی‌شود؛ دکمه غیرفعال با دلیل نمایش داده می‌شود و پنهان نمی‌شود.</p>
                    </DrawerSection>
                    <DrawerSection title="محتوای بلند (اسکرول داخلی)">
                      {Array.from({ length: 8 }, (_, i) => <p key={i} className="ds-sub">ردیف نمونه {i + 1} — متن ساختگی برای نمایش اسکرول بدنه کشو.</p>)}
                    </DrawerSection>
                  </>
                )}
          </Drawer>
        }
      >
        <PageHead
          title="سامانه طراحی CRM — نمونه اجزا"
          sub="فقط محیط توسعه. همه داده‌ها ساختگی‌اند و هیچ اقدامی ثبت نمی‌شود."
          kpis={
            <MetricGroup label="شاخص‌های نمونه">
              <Metric label="بدون وضعیت" value="۴" accent="neutral" />
              <Metric label="تماس مجدد امروز" value="۱" accent="blue" state="stale" />
              <Metric label="نرخ پیگیری" accent="teal" state="unavailable" />
              <Metric label="پوشش منبع" accent="violet" state="undefined" />
              <Metric label="مانده قابل وصول" accent="amber" state="unknown" />
            </MetricGroup>
          }
          actions={<button type="button" className="btn" onClick={() => demo('اقدام سرصفحه')}>اقدام نمونه</button>}
          meta={<><Freshness text="به‌روزرسانی ۵ دقیقه پیش (ساختگی)" /><Freshness text="داده قدیمی‌تر از بازه تازگی (ساختگی)" stale /></>}
        />

        <p className="ds-status-line" role="status" aria-live="polite">{status}</p>

        <Banner kind="stale">داده‌های این صفحه قدیمی‌اند؛ اقدام‌ها هنگام ثبت دوباره بررسی می‌شوند.</Banner>
        <Banner kind="incomplete">پوشش منبع ناقص است؛ جمع‌ها نهایی نیستند.</Banner>
        <Banner kind="unknown">نتیجه آخرین ارسال نامعلوم است؛ پیش از تکرار تطبیق دهید.</Banner>
        <Banner kind="locked">این بخش برای نقش شما فقط‌خواندنی است.</Banner>

        <Panel label="کشو">
          <PanelHeader title="Drawer / Sheet — کشو" icon="file" />
          <div className="ds-demo-body ds-row-wrap">
            <button type="button" className="btn btn-soft" onClick={() => { setDrawerContent('content'); setDrawer('normal') }}>باز کردن کشو</button>
            <button type="button" className="btn" onClick={() => { setDrawerContent('content'); setDrawer('wide') }}>کشوی پهن</button>
            <span className="ds-sub">۵۶۰ / ۶۶۰ پیکسل در دسکتاپ؛ برگه پایینی در موبایل. Esc می‌بندد و فوکوس برمی‌گردد.</span>
          </div>
        </Panel>

        <Panel label="صف‌ها (QueueTabs)">
          <PanelHeader title="QueueTabs — صف‌ها" icon="layers">
            <ContextHint text="«—» یعنی تعداد نامعلوم است، نه صفر. صفرِ شناخته‌شده نمایش داده می‌شود." />
          </PanelHeader>
          <QueueTabs queues={QUEUES} value={queue} onChange={id => { setQueue(id); demo(`صف ${QUEUES.find(q => q.id === id)?.label ?? id}`) }} />
          <Toolbar>
            <label className="ds-field"><span className="sr">جستجو در این صف</span><input className="input" type="search" placeholder="جستجو در این صف (نمایشی)" /></label>
            <button type="button" className="ds-chip-btn" onClick={() => demo('فیلتر منبع')}>منبع</button>
            <button type="button" className="ds-chip-btn" disabled>فیلتر غیرفعال</button>
          </Toolbar>
          <StatePanel kind="empty" title="فهرست نمونه این‌جا نمایش داده می‌شود" detail="جدول‌ها در RM-E03 پیاده می‌شوند." />
        </Panel>

        <Panel label="وضعیت‌ها">
          <PanelHeader title="Status / state indicators" icon="info" />
          <div className="ds-demo-body">
            <h3 className="ds-demo-h">وضعیت داده (متمایز؛ هیچ‌کدام صفر یا موفق نیست)</h3>
            <div className="ds-row-wrap">{states.map(s => <StatusIndicator key={s} state={s} />)}</div>
            <h3 className="ds-demo-h">نتیجه‌های ثابت (outcome)</h3>
            <div className="ds-row-wrap">{outcomes.map(o => { const p = outcomePresentation[o]; return <StatusPill key={o} tone={p.tone} icon={p.icon} label={p.label} /> })}</div>
          </div>
        </Panel>

        <Panel label="حالت‌های سیستم">
          <PanelHeader title="StatePanel — حالت‌های سیستم" icon="grid" />
          <div className="ds-state-grid">
            {PANEL_STATES.map(s => (
              <div key={s.kind} className="ds-state-cell">
                <StatePanel kind={s.kind} title={s.title} detail={s.detail} headingLevel={3}
                  actions={s.kind === 'retry' || s.kind === 'error' ? <button type="button" className="btn btn-sm" onClick={() => demo('تلاش مجدد')}>تلاش مجدد</button> : s.kind === 'unknown' ? <button type="button" className="btn btn-sm" disabled>تکرار (تا تطبیق غیرفعال)</button> : undefined} />
              </div>
            ))}
          </div>
        </Panel>

        <Panel label="منو و پاپ‌اور">
          <PanelHeader title="Menu / Popover" icon="more" />
          <div className="ds-demo-body ds-row-wrap">
            <MenuButton label="اقدام‌های ردیف نمونه" entries={exampleMenu} icon="more">اقدام‌ها</MenuButton>
            <button ref={popBtn} type="button" className="btn" aria-haspopup="dialog" aria-expanded={popOpen} onClick={() => setPopOpen(o => !o)}>پاپ‌اور توضیح</button>
            <Popover open={popOpen} onClose={closePop} anchorRef={popBtn} label="توضیح نمونه">
              <div className="ds-demo-body">
                <b>پاپ‌اور غیرمودال</b>
                <p className="ds-sub">Esc یا کلیک بیرون آن را می‌بندد و فوکوس به دکمه برمی‌گردد.</p>
              </div>
            </Popover>
            <button type="button" className="btn btn-primary" disabled>دکمه اصلی غیرفعال</button>
            <button type="button" className="btn" disabled>دکمه غیرفعال</button>
          </div>
        </Panel>

        <Panel label="راهنما">
          <PanelHeader title="Help — راهنمای زمینه‌ای" icon="help" />
          <div className="ds-demo-body" id="ds-help">
            <p className="ds-sub">راهنمای زمینه‌ای کنار برچسب: مسئول فعلی <ContextHint text="پرونده الان نزد کیست (Current Custody)." /></p>
            <Glossary entries={CORE_TERMS} />
          </div>
        </Panel>

        <Panel label="صفحه‌کلید">
          <PanelHeader title="دسترس‌پذیری صفحه‌کلید" icon="key" />
          <div className="ds-demo-body" id="ds-keyboard">
            <ShortcutList entries={[
              { keys: ['Tab'], description: 'رفتن به عنصر بعدی؛ پیوند پرش اولین عنصر است' },
              { keys: ['←', '→'], description: 'جابه‌جایی میان صف‌ها (راست‌به‌چپ)' },
              { keys: ['↑', '↓'], description: 'جابه‌جایی میان گزینه‌های منو' },
              { keys: ['Esc'], description: 'بستن منو، پاپ‌اور یا کشو و بازگشت فوکوس' },
            ]} />
          </div>
        </Panel>

        <Panel label="توکن‌ها">
          <PanelHeader title="Tokens — رنگ، تراکم، پوسته" icon="layers" />
          <div className="ds-demo-body">
            <div className="ds-row-wrap" role="group" aria-label="پوسته">
              {(['light', 'dim', 'dark'] as const).map(t => <button key={t} type="button" className="btn btn-sm" aria-pressed={theme === t} onClick={() => setTheme(t)}>{{ light: 'روشن', dim: 'کم‌نور', dark: 'تیره' }[t]}</button>)}
            </div>
            <div className="ds-row-wrap" role="group" aria-label="تراکم">
              {(['comfortable', 'compact'] as const).map(d => <button key={d} type="button" className="btn btn-sm" aria-pressed={density === d} onClick={() => setDensity(d)}>{{ comfortable: 'راحت', compact: 'فشرده' }[d]}</button>)}
            </div>
            <div className="ds-row-wrap" role="group" aria-label="محتوای کشو">
              {(['content', 'loading', 'error'] as const).map(c => <button key={c} type="button" className="btn btn-sm" aria-pressed={drawerContent === c} onClick={() => setDrawerContent(c)}>{{ content: 'کشو: محتوا', loading: 'کشو: بارگذاری', error: 'کشو: خطا' }[c]}</button>)}
            </div>
            <ul className="ds-swatches">
              {TONES.map(t => (
                <li key={t.tone} className={`t-${t.tone}`}>
                  <span className="ds-swatch" aria-hidden="true" />
                  <span><b className="num">{t.tone}</b> — {t.meaning}</span>
                </li>
              ))}
            </ul>
          </div>
        </Panel>
      </DsAppShell>
    </div>
  )
}
