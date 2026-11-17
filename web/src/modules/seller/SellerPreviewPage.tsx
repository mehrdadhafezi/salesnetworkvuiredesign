import { useCallback, useEffect, useMemo, useState, type MouseEvent } from 'react'
import { Navigate, useNavigate, useParams, useSearchParams } from 'react-router'
import {
  AppHeader, Banner, DsAppShell, DsIcon, HeaderIconButton, HelpCenterButton, MenuButton, MobileBottomNavigation, RoleNavigation,
  toFaDigits, type DsNavItem, type MenuEntry,
} from '../../shared/ui/ds'
import { CommandPalette, Toasts, type PaletteItem } from './components/overlays'
import { CONVERSIONS, INVOICES, LEADS, PREVIEW_USER, SELLER_PREVIEW_MARKER } from './fixtures/sellerFixtures'
import { LEAD_STATUS, SECTION_LABEL, sellerNavItems } from './labels'
import { BehaviorPage } from './pages/BehaviorPage'
import { ConversionsPage } from './pages/ConversionsPage'
import { InvoicesPage } from './pages/InvoicesPage'
import { LeadsPage } from './pages/LeadsPage'
import { RepeatPage } from './pages/RepeatPage'
import { RequestPage } from './pages/RequestPage'
import { WalletPage } from './pages/WalletPage'
import { isTypingTarget } from './state/hooks'
import { leadCounts } from './state/selectors'
import { SellerPreviewProvider, useSellerPreview, type Density, type Sim, type Theme } from './state/SellerPreviewContext'
import { SELLER_SECTIONS, type SellerSection } from './types'
import './seller.css'

/*
 * RM-E03 — Seller V1 panel, DEV/TEST-ONLY PREVIEW (/dev/seller/:section).
 *
 * Mounted only when import.meta.env.DEV && config.devTools (app/routes.tsx), outside the RM-E02
 * module Shell and its fail-closed navigation; the production bundle never contains it (web-ci
 * greps for this file name and SELLER_PREVIEW_MARKER). It renders the frozen Seller Design V1
 * (prototype/index.html) with RM-E01 components over SYNTHETIC fixtures.
 *
 * It grants nothing: no session, identity, role, capability, flag or grant is read or changed, no
 * CRM endpoint is called, and every simulated business action ends in a demo toast.
 * Query parameters (theme, density, sim, lead, inv, case, …) are presentation only.
 */

export const SELLER_PREVIEW_BASE = '/dev/seller'

const pick = <T extends string>(v: string | null, allowed: readonly T[], fallback: T): T => (allowed.includes(v as T) ? (v as T) : fallback)

export function SellerPreviewPage() {
  const [params] = useSearchParams()
  const initial = {
    theme: pick<Theme>(params.get('theme'), ['light', 'dim', 'dark'], 'light'),
    density: pick<Density>(params.get('density'), ['comfortable', 'compact'], 'comfortable'),
    sim: pick<Sim>(params.get('sim'), ['none', 'loading', 'empty', 'error', 'offline'], 'none'),
  }
  return (
    <SellerPreviewProvider base={SELLER_PREVIEW_BASE} initial={initial}>
      <SellerPreviewShell />
    </SellerPreviewProvider>
  )
}

function isSection(v: string | undefined): v is SellerSection {
  return v !== undefined && (SELLER_SECTIONS as readonly string[]).includes(v)
}

function SellerPreviewShell() {
  const { section } = useParams()
  const navigate = useNavigate()
  const { base, theme, density, sim, setTheme, setDensity, setSim, toast } = useSellerPreview()
  const [palette, setPalette] = useState(false)

  // Theme / density on <html> like the prototype, so body, scrollbars and overlays follow; restored on leave.
  useEffect(() => {
    const root = document.documentElement
    const prev = { theme: root.getAttribute('data-theme'), density: root.getAttribute('data-density') }
    root.setAttribute('data-theme', theme)
    root.setAttribute('data-density', density)
    return () => {
      if (prev.theme === null) root.removeAttribute('data-theme'); else root.setAttribute('data-theme', prev.theme)
      if (prev.density === null) root.removeAttribute('data-density'); else root.setAttribute('data-density', prev.density)
    }
  }, [theme, density])

  const current: SellerSection | null = isSection(section) ? section : null
  useEffect(() => { if (current) document.title = `${SECTION_LABEL[current]} — پنل فروشنده (پیش‌نمایش ساختگی)` }, [current])

  const counts = useMemo(() => leadCounts(LEADS), [])
  const nav = useMemo(() => sellerNavItems(base, {
    leadsNone: counts.none, conversions: CONVERSIONS.length, invoicesAction: INVOICES.filter(i => i.next).length,
  }), [base, counts])

  const go = useCallback((s: SellerSection, query = '') => { void navigate(`${base}/${s}${query}`); window.scrollTo(0, 0) }, [navigate, base])
  const onNavigate = (item: DsNavItem, e?: MouseEvent<HTMLAnchorElement>) => { e?.preventDefault(); go(item.id as SellerSection) }

  // Global keys: Ctrl/⌘+K and «/» open the palette (prototype).
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'k') { e.preventDefault(); setPalette(true); return }
      if (e.key === '/' && !e.ctrlKey && !e.metaKey && !e.altKey && !isTypingTarget(document.activeElement)) { e.preventDefault(); setPalette(true) }
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [])

  const paletteItems = useMemo<PaletteItem[]>(() => [
    ...nav.map<PaletteItem>(n => ({ id: `go-${n.id}`, group: 'رفتن به', icon: n.icon, label: n.label, keywords: `رفتن به ${SECTION_LABEL[n.id as SellerSection]}`, run: () => go(n.id as SellerSection) })),
    ...LEADS.map<PaletteItem>(l => ({ id: `lead-${l.id}`, group: 'مشتریان', icon: 'users', label: l.name, meta: `${toFaDigits(l.phone)} · ${LEAD_STATUS[l.status].label}`, keywords: l.phone, run: () => go('leads', `?queue=all&lead=${l.id}`) })),
    ...INVOICES.map<PaletteItem>(i => ({ id: `inv-${i.code}`, group: 'فاکتورها', icon: 'receipt', label: `فاکتور ${i.code}`, meta: i.customer, keywords: i.code, run: () => go('invoices', `?inv=${i.code}`) })),
    ...(['light', 'dim', 'dark'] as const).map<PaletteItem>(t => ({ id: `theme-${t}`, group: 'نمایش', icon: 'grid', label: { light: 'پوسته روشن', dim: 'پوسته کم‌نور', dark: 'پوسته تیره' }[t], meta: theme === t ? 'فعال' : undefined, keywords: 'تغییر پوسته', run: () => setTheme(t) })),
  ], [nav, go, theme, setTheme])

  if (!current) return <Navigate to={`${base}/leads`} replace />

  const userMenu: MenuEntry[] = [
    { kind: 'heading', id: 'h-theme', label: 'پوسته' },
    { kind: 'item', id: 'light', label: 'روشن', current: theme === 'light', onSelect: () => setTheme('light') },
    { kind: 'item', id: 'dim', label: 'کم‌نور', current: theme === 'dim', onSelect: () => setTheme('dim') },
    { kind: 'item', id: 'dark', label: 'تیره', current: theme === 'dark', onSelect: () => setTheme('dark') },
    { kind: 'heading', id: 'h-density', label: 'تراکم فهرست‌ها' },
    { kind: 'item', id: 'comfortable', label: 'راحت', current: density === 'comfortable', onSelect: () => setDensity('comfortable') },
    { kind: 'item', id: 'compact', label: 'فشرده', current: density === 'compact', onSelect: () => setDensity('compact') },
    { kind: 'heading', id: 'h-sim', label: 'حالت‌های نمایشی (دمو)' },
    ...([['none', 'عادی'], ['loading', 'در حال بارگذاری'], ['empty', 'خالی'], ['error', 'خطای فهرست'], ['offline', 'آفلاین']] as const).map<MenuEntry>(([id, label]) => ({
      kind: 'item', id: `sim-${id}`, label, current: sim === id, onSelect: () => setSim(id),
    })),
    { kind: 'separator', id: 's1' },
    { kind: 'item', id: 'password', label: 'رمز من', icon: 'lock', disabled: true, disabledReason: 'پیش‌نمایش حساب کاربری ندارد.' },
    { kind: 'item', id: 'logout', label: 'خروج', icon: 'logout', disabled: true, disabledReason: 'پیش‌نمایش نشستی ندارد؛ چیزی برای خروج نیست.' },
  ]

  const open = LEADS.filter(l => l.status === 'none' || l.status === 'no_answer' || l.status === 'callback').length

  return (
    <div className="sl-root" dir="rtl" lang="fa" data-preview={SELLER_PREVIEW_MARKER}>
      <DsAppShell
        header={
          <AppHeader
            brandMark="س"
            brandName="سامانه فروش"
            crumb={SECTION_LABEL[current]}
            tag="پیش‌نمایش توسعه · داده ساختگی"
            mobileTitle={SECTION_LABEL[current]}
            search={{ label: 'جستجو و فرمان‌ها (Ctrl+K)', placeholder: 'جستجوی مشتری، فاکتور یا رفتن به…', shortcut: 'Ctrl K', onOpen: () => setPalette(true) }}
            actions={
              <>
                <div className="sl-workload" aria-label={`${toFaDigits(open)} مورد باز (ساختگی)`}>
                  <DsIcon name="menu" /><span><b>{toFaDigits(open)}</b> مورد باز</span>
                </div>
                <HeaderIconButton label="اعلان‌ها — ۲ مورد نمونه" icon="bell" badge="۲" onClick={() => toast('اعلان‌های نمونه: ۱ تماس مجدد امروز، ۱ فاکتور رد شده. (ساختگی)')} />
                <HelpCenterButton
                  subtitle="راهنمای پیش‌نمایش پنل فروشنده (ساختگی)."
                  topics={[
                    { id: 'keys', title: 'میانبرهای صفحه‌کلید', summary: 'Ctrl+K یا / پالت فرمان · ۰–۶ صف · F جستجو · J/K مشتری بعدی/قبلی · ۱–۵ نتیجه در کشو · Esc بستن', icon: 'key', onOpen: () => toast('میانبرها: Ctrl+K یا / پالت · ۰–۶ صف · F جستجو · J/K بعدی/قبلی · ۱–۵ نتیجه · Esc بستن') },
                    { id: 'data', title: 'درباره داده‌ها', summary: 'همه داده‌ها ساختگی‌اند و هیچ اقدامی ثبت یا ارسال نمی‌شود.', icon: 'info', onOpen: () => toast('همه داده‌ها ساختگی‌اند؛ پیش‌نمایش به هیچ سرویس تجاری وصل نیست.') },
                  ]}
                />
                <MenuButton label="منوی کاربر، پوسته و تراکم" className="ds-user" entries={userMenu}>
                  <span className="ds-avatar" aria-hidden="true">{PREVIEW_USER.initials}</span>
                  <span className="ds-user-meta"><b>{PREVIEW_USER.name}</b><span>{PREVIEW_USER.role} · {PREVIEW_USER.id}</span></span>
                  <DsIcon name="chev" />
                </MenuButton>
              </>
            }
          />
        }
        nav={<RoleNavigation items={nav} currentId={current} label="بخش‌های پنل فروشنده" onNavigate={onNavigate} />}
        bottomNav={<MobileBottomNavigation items={nav} currentId={current} onNavigate={onNavigate} />}
        overlay={<><CommandPalette open={palette} onClose={() => setPalette(false)} items={paletteItems} /><Toasts /></>}
      >
        {sim === 'offline' ? (
          <Banner kind="offline" action={<button type="button" className="btn btn-sm" onClick={() => { setSim('none'); toast('حالت نمایشی آفلاین خاموش شد.') }}><DsIcon name="refresh" />تلاش مجدد</button>}>
            <b>اتصال اینترنت قطع شده است (نمایشی).</b> اقدام‌ها تا برقراری اتصال متوقف‌اند.
          </Banner>
        ) : null}
        <Section key={current} section={current} />
      </DsAppShell>
    </div>
  )
}

function Section({ section }: { section: SellerSection }) {
  switch (section) {
    case 'leads': return <LeadsPage />
    case 'conversions': return <ConversionsPage />
    case 'repeat': return <RepeatPage />
    case 'invoices': return <InvoicesPage />
    case 'behavior': return <BehaviorPage />
    case 'request': return <RequestPage />
    case 'wallet': return <WalletPage />
  }
}
