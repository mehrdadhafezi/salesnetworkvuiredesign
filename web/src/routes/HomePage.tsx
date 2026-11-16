import { Link } from 'react-router'
import type { NavAccess, NavEntry } from '../app/navigation/access'
import { MODULE_LABEL, SAFE_DESTINATIONS, type DestinationKind } from '../app/navigation/destinations'
import { useNavigationAccess } from '../app/navigation/NavigationAccess'
import { ROLE_CONTRACTS } from '../app/navigation/roles'
import { useSession, useSessionActions } from '../shared/auth/currentSession'
import { Banner, DsIcon, PageHead, Panel, PanelHeader, StatePanel, TonePill } from '../shared/ui/ds'
import './home.css'

/**
 * RM-E02 role entry. Not a dashboard: no metric, count or business row is
 * shown or fetched here. It lists exactly the destinations the fail-closed
 * resolver allowed, or explains honestly why there are none.
 */
export function HomePage() {
  const access = useNavigationAccess()
  return (
    <>
      <PageHead title="خانه" sub={subtitle(access)} />
      <Entry access={access} />
    </>
  )
}

function subtitle(access: NavAccess): string {
  if (access.status !== 'GRANTED') return 'ورود به بخش‌های CRM بر اساس دسترسی‌هایی که سرور برای این نشست تأیید می‌کند.'
  if (access.role.kind === 'SINGLE') return `بخش‌های مجاز برای نقش ${ROLE_CONTRACTS[access.role.role].label}.`
  return 'بخش‌هایی که سرور برای این نشست مجاز دانسته است.'
}

const KIND: Record<DestinationKind, { tone: 'slate' | 'violet' | 'blue'; label: string; icon: 'file' | 'activity' | 'check' }> = {
  READ: { tone: 'slate', label: 'فقط خواندنی', icon: 'file' },
  CONTROLLED_WRITE: { tone: 'violet', label: 'اقدام کنترل‌شده', icon: 'check' },
  DIAGNOSTIC: { tone: 'blue', label: 'عیب‌یابی', icon: 'activity' },
}

function SafeLinks() {
  return <Link className="btn btn-ghost" to={SAFE_DESTINATIONS.system.to}>{SAFE_DESTINATIONS.system.label}</Link>
}

function Retry() {
  const { refresh } = useSessionActions()
  return <button type="button" className="btn" onClick={refresh}>بررسی دوباره</button>
}

function Entry({ access }: { access: NavAccess }) {
  const session = useSession()
  switch (access.status) {
    case 'LOADING':
      return <StatePanel kind="loading" title="در حال بررسی دسترسی‌ها…" detail="تا پاسخ سرور نرسد هیچ بخشی نمایش داده نمی‌شود." />
    case 'ANONYMOUS':
      return (
        <StatePanel kind="locked" title={session.status === 'failed' ? 'وضعیت نشست مشخص نیست' : 'هویتی برای این نشست ثبت نشده است'}
          detail="بدون هویت تأییدشده، هیچ بخش کسب‌وکاری نمایش داده نمی‌شود." actions={<SafeLinks />} />
      )
    case 'FAILED':
      return (
        <StatePanel kind="error" title="ارزیابی دسترسی انجام نشد"
          detail="پاسخ سرور درباره دسترسی‌های این نشست دریافت نشد. تا روشن شدن آن، هیچ بخشی مجاز فرض نمی‌شود."
          actions={<><Retry /><SafeLinks /></>} />
      )
    case 'CONFLICT':
      return (
        <StatePanel kind="conflict" title="پاسخ‌های سرور درباره نشست ناسازگار است"
          detail="نشست شما واردشده گزارش شد، اما ارزیابی دسترسی آن را ناشناس دانست. هیچ بخشی نمایش داده نمی‌شود."
          actions={<><Retry /><SafeLinks /></>} />
      )
    case 'BLOCKED':
      return (
        <StatePanel kind="locked" title="ارزیابی دسترسی در سرور در دسترس نیست"
          detail={<>سرور ارزیابی همه دسترسی‌ها را مسدود گزارش کرد (<bdi dir="ltr">{access.reasons.join(', ')}</bdi>). تا رفع آن، هیچ بخش کسب‌وکاری در ناوبری نمایش داده نمی‌شود.</>}
          actions={<SafeLinks />} />
      )
    case 'NONE':
      return access.reason === 'NO_GRANT'
        ? <StatePanel kind="locked" title="دسترسی فعالی برای شما ثبت نشده است"
            detail="حساب شما وارد شده است، اما سرور هیچ مجوز فعالی برای بخش‌های کسب‌وکار گزارش نکرده است. برای دریافت دسترسی با مدیر سامانه هماهنگ کنید."
            actions={<SafeLinks />} />
        : <StatePanel kind="locked" title="بخش‌های مجاز شما هنوز فعال نشده‌اند"
            detail="سرور برای برخی بخش‌ها مجوز گزارش کرد، اما هیچ‌کدام از آن بخش‌ها هنوز در سامانه فعال نشده است."
            actions={<SafeLinks />} />
    case 'GRANTED':
      return <Granted access={access} />
  }
}

function Granted({ access }: { access: Extract<NavAccess, { status: 'GRANTED' }> }) {
  const groups: { title: string; entries: NavEntry[] }[] = access.role.kind === 'SINGLE'
    ? [
        { title: 'بخش‌های نقش شما', entries: access.entries.filter(e => e.placement === 'role') },
        { title: 'سایر بخش‌های مجاز', entries: access.entries.filter(e => e.placement === 'other') },
      ]
    : [...new Set(access.entries.map(e => e.module))].map(m => ({ title: MODULE_LABEL[m], entries: access.entries.filter(e => e.module === m) }))
  return (
    <div className="role-entry">
      {access.role.kind === 'MULTIPLE'
        ? <Banner kind="info">برای این نشست بیش از یک نقش یا مبنای دسترسی ثبت شده و سیاست انتخاب نقش هنوز تعریف نشده است. فهرست زیر دقیقاً همان بخش‌هایی است که سرور مجاز دانسته؛ هیچ نقشی به‌جای شما انتخاب نشده است.</Banner>
        : null}
      {access.role.kind === 'UNKNOWN'
        ? <Banner kind="unknown">نقش شما از پاسخ سرور مشخص نیست. فهرست زیر دقیقاً همان بخش‌هایی است که سرور مجاز دانسته است.</Banner>
        : null}
      <Banner kind="info">این صفحه داشبورد نیست و هیچ شاخص یا داده‌ای نشان نمی‌دهد. هر صفحه دسترسی را دوباره از سرور می‌پرسد.</Banner>
      {groups.filter(g => g.entries.length > 0).map(g => (
        <Panel key={g.title} label={g.title}>
          <PanelHeader title={g.title} />
          <ul className="role-entry-list">
            {g.entries.map(e => (
              <li key={e.id}>
                <Link to={e.to}><DsIcon name={e.icon} /><span>{e.label}</span></Link>
                <TonePill tone={KIND[e.kind].tone} icon={KIND[e.kind].icon} label={KIND[e.kind].label} />
              </li>
            ))}
          </ul>
        </Panel>
      ))}
    </div>
  )
}
