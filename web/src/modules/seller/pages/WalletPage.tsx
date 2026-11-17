import { useMemo, useState } from 'react'
import { Banner, ContextHint, PageHead, Panel, StatePanel, StatusIndicator, TonePill, Toolbar, formatFaNumber, toFaDigits } from '../../../shared/ui/ds'
import { FilterChip, Money, TableFoot, Tag, useListState } from '../components/bits'
import { WALLET } from '../fixtures/sellerFixtures'
import { WALLET_STATE } from '../labels'
import { filterWallet, paginate, walletTotals } from '../state/selectors'
import { useSellerPreview } from '../state/SellerPreviewContext'
import type { WalletChannel, WalletTxState } from '../types'

/*
 * S7 — کیف پول من (prototype V.wallet). SYNTHETIC lines only.
 * The preview does not claim a settleable balance or entitlement: it shows confirmed and pending
 * sums separately, keeps unknown lines out of both (and says the total is incomplete), and offers no
 * payout, withdrawal, settlement or credit action. The «پروژه‌ها» wallet is not defined for Seller V1.
 */

const PER_PAGE = 10
const STATES: { id: WalletTxState; label: string }[] = [{ id: 'confirmed', label: 'تایید شده' }, { id: 'pending', label: 'در انتظار تایید' }, { id: 'unknown', label: 'نامعلوم' }]

export function WalletPage() {
  const { setSim } = useSellerPreview()
  const [wallet, setWallet] = useState<'sales' | 'projects'>('sales')
  const [channel, setChannel] = useState<'all' | WalletChannel>('all')
  const [state, setState] = useState<WalletTxState | null>(null)
  const [page, setPage] = useState(1)
  const totals = walletTotals(WALLET.tx)
  const list = useMemo(() => filterWallet(WALLET.tx, channel, state ?? 'all'), [channel, state])
  const pg = paginate(list, page, PER_PAGE)
  const stateEl = useListState({
    count: pg.items.length, noResult: channel !== 'all' || state !== null, onClear: () => { setChannel('all'); setState(null) }, onRetry: () => setSim('none'),
    emptyTitle: 'در این بازه تراکنشی ثبت نشده است', emptyDetail: 'بازه تاریخ یا نوع پرداخت را تغییر دهید.',
  })

  return (
    <>
      <PageHead title="کیف پول من" sub="فروش و پورسانت — همه ارقام ساختگی‌اند و ادعای مالی نیستند" />
      <div className="sl-wallet-sum">
        <div className="sl-wcard sl-hero">
          <span>پورسانت تایید‌شده (نمونه)</span>
          <b><span className="num">{formatFaNumber(totals.confirmed)}</span><small>تومان</small></b>
          <div className="sl-foot">
            <span>آخرین بستانکاری: {WALLET.lastCredit}</span>
            <span className="sl-seg" role="group" aria-label="کیف پول">
              <button type="button" aria-pressed={wallet === 'sales'} onClick={() => setWallet('sales')}>فروش و پورسانت</button>
              <button type="button" aria-pressed={wallet === 'projects'} onClick={() => setWallet('projects')}>پروژه‌ها</button>
            </span>
          </div>
        </div>
        <div className="sl-wcard">
          <span>در انتظار تایید</span>
          <b><span className="num">{formatFaNumber(totals.pending)}</span><small>تومان</small></b>
          <div className="sl-foot"><span>تا تایید مالی، قابل تسویه حساب نمی‌شود</span></div>
        </div>
        <div className="sl-wcard">
          <span>برداشت / تسویه</span>
          <b><span className="sl-na">—</span><small>تعریف نشده</small></b>
          <div className="sl-foot"><span>تسویه در Seller V1 تعریف نشده است؛ «—» به معنی صفر نیست</span></div>
        </div>
      </div>
      {totals.unknownCount > 0
        ? <Banner kind="incomplete">{toFaDigits(totals.unknownCount)} ردیف با وضعیت یا مبلغ نامعلوم در هیچ‌یک از جمع‌ها لحاظ نشده است؛ جمع‌ها نهایی نیستند.</Banner>
        : null}
      {wallet === 'projects' ? (
        <Panel label="کیف پول پروژه‌ها">
          <StatePanel kind="locked" title="کیف پول پروژه‌ها در Seller V1 تعریف نشده است" detail="نمایش‌ندادن داده به معنی نبودن داده نیست. این بخش در نسخه طراحی منجمد فقط به‌صورت گزینه دیده می‌شود." />
        </Panel>
      ) : (
        <Panel label="تراکنش‌های کیف پول">
          <Toolbar>
            <div className="sl-seg" role="group" aria-label="نوع پرداخت">
              {([['all', 'همه'], ['online', 'پرداخت آنلاین'], ['card', 'کارت‌به‌کارت']] as const).map(([id, label]) => (
                <button key={id} type="button" aria-pressed={channel === id} onClick={() => { setChannel(id); setPage(1) }}>{label}</button>
              ))}
            </div>
            <FilterChip label="وضعیت" value={state} options={STATES} onChange={s => { setState(s); setPage(1) }} />
            <span className="ds-grow" />
            <span className="muted sl-desk" role="status">{toFaDigits(list.length)} ردیف</span>
          </Toolbar>
          {stateEl ?? (
            <>
              <div className="sl-tbl-wrap">
                <table className="sl-tbl sl-stackable" aria-label="تراکنش‌های کیف پول">
                  <thead><tr><th scope="col">تاریخ رویداد</th><th scope="col">نوع</th><th scope="col">کانال پرداخت فروش</th><th scope="col">شرح</th><th scope="col">وضعیت</th><th scope="col" className="sl-end">مبلغ پورسانت <ContextHint text="مبلغ پورسانت است، نه مبلغ فروش فاکتور مبنا." /></th></tr></thead>
                  <tbody>
                    {pg.items.map(t => {
                      const p = WALLET_STATE[t.state]
                      return (
                        <tr key={t.id} className="sl-static">
                          <td data-l="تاریخ رویداد" className="muted"><span className="num">{t.date}</span></td>
                          <td data-l="نوع"><TonePill tone="green" icon="plus" label="بستانکار" /></td>
                          <td data-l="کانال پرداخت فروش"><Tag teal={t.channel === 'online'}>{t.channel === 'online' ? 'آنلاین' : 'کارت به کارت'}</Tag></td>
                          <td data-l="شرح">پورسانت فاکتور <span className="mono">{t.invoice}</span><div className="muted sl-micro">{t.rule}</div></td>
                          <td data-l="وضعیت">{t.state === 'unknown' ? <StatusIndicator state="unknown" /> : <TonePill tone={p.tone} icon={p.icon} label={p.label} />}</td>
                          <td data-l="مبلغ پورسانت" className="sl-end">{t.amount === null ? <Money value={null} /> : <span className={t.state === 'confirmed' ? 'sl-credit' : 'sl-pending'}>+<Money value={t.amount} /></span>}</td>
                        </tr>
                      )
                    })}
                  </tbody>
                </table>
              </div>
              <TableFoot page={pg} perPage={PER_PAGE} onPage={setPage} />
            </>
          )}
        </Panel>
      )}
    </>
  )
}
