import '../writes/writes.css'
import './payments.css'
import { useState } from 'react'
import { Link } from 'react-router'
import { useApi, type PaymentBatchEnvelope, type PaymentUploadEnvelope, type SubmittedAttempt } from '../../shared/api'
import { useSession } from '../../shared/auth/currentSession'
import { actionHint } from '../../shared/permissions/allowedActions'
import { settle, useRead } from '../../shared/read/load'
import { ReadStates } from '../../shared/read/ReadStates'
import { formatInstant } from '../../shared/ui/time'
import { Button, Card, StateBlock, StatusPill } from '../../shared/ui/primitives'
import { newIdempotencyKey } from '../writes/CommandBanner'
import { modeLabel, reasonText } from '../writes/labels'
import {
  attemptStateLabel, batchWritebackLabel, blockedPaymentLabel, codes, evidenceStateLabel, evidenceStateTone, money, receiptStateLabel, receiptStateTone,
  unitLabel, uploadStateLabel,
} from './labels'

const ACTION = 'payments.batch.submit'

// Stable client item identities (never the position): one per new item.
let itemSeq = 0
const nextItemKey = () => `item-${(itemSeq += 1)}-${crypto.randomUUID().slice(0, 8)}`

interface Staged { uploadId: string; sha: string; name: string; size: number }
interface ItemDraft { key: string; amount: string; receipts: string[] }
interface Intent { items: { item_key: string; amount: number; receipts: Staged[] }[]; total: number }

/** The honest batch result: committed in the target ≠ reviewed ≠ in legacy. */
export function BatchBanner({ env }: { env: PaymentBatchEnvelope }) {
  const d = env.data
  const items = d?.items ?? []
  const list = items.length > 0 && (
    <ul className="wr-list" aria-label="نتیجه هر ردیف">
      {items.map(it => (
        <li key={it.item_key}>
          <span className="wr-ltr">{it.item_key}</span> · {money(it.amount, it.unit)} ·{' '}
          {it.committed
            ? <StatusPill tone="green" icon="check" label="ثبت شد" />
            : <StatusPill tone={it.valid ? 'slate' : 'red'} icon={it.valid ? 'lock' : 'x'}
                label={it.valid ? 'معتبر اما ثبت نشد (دسته کامل یا هیچ)' : `رد شد: ${reasonText(it.reasons)}`} />}
          {it.blocked_by.length > 0 && <span className="muted"> · تصمیم باز: {codes(it.blocked_by)}</span>}
        </li>
      ))}
    </ul>
  )
  switch (env.outcome) {
    case 'SUCCESS':
      return (
        <div className="wr-banner" role="status" aria-label="نتیجه ثبت پرداخت">
          <strong>{d?.replayed ? 'تکرار همان درخواست؛ نتیجه قبلی' : 'پرداخت در سامانه جدید ثبت شد'} — در انتظار بررسی مالی؛ هنوز تایید یا ثبت مالی نشده است</strong>
          <span className="muted">{d ? batchWritebackLabel[d.writeback_state] : ''}</span>
          {d?.batch_id && <Link to={`/payments/batches/${d.batch_id}`}>جزئیات این دسته</Link>}
          {list}
        </div>
      )
    case 'BLOCKED':
      return (<><StateBlock kind="locked" title="این دسته ثبت نشد (هیچ ردیفی ثبت نشد)" detail={`${reasonText(env.reasons)}${d?.blocked_by.length ? ` · تصمیم‌های باز: ${codes(d.blocked_by)}` : ''}`} />{list}</>)
    case 'CONFLICT':
      return (<><StateBlock kind="error" title="تعارض: وضعیت مرحله تغییر کرده، کلید برای درخواست دیگری است یا فیش قبلاً متصل شده"
        detail="اطلاعات تازه بارگذاری شد؛ پیش از هر اقدام دوباره بررسی کنید. هیچ ردیفی ثبت نشد." />{list}</>)
    case 'UNKNOWN':
      return <StateBlock kind="unknown" title="نتیجه نامعلوم است و در حال تطبیق است" detail="این درخواست را دوباره ارسال نکنید؛ وضعیت پس از تطبیق نمایش داده می‌شود." />
    case 'UNAUTHORIZED':
      return <StateBlock kind="locked" title="اجازه ثبت پرداخت برای این فاکتور را ندارید" detail={reasonText(env.reasons)} />
    default:
      return <StateBlock kind="error" title="ثبت انجام نشد" detail={reasonText(env.reasons)} />
  }
}

export function AttemptList({ attempts, zone }: { attempts: readonly SubmittedAttempt[]; zone: string }) {
  if (attempts.length === 0) return <StateBlock kind="empty" title="هنوز پرداختی در سامانه جدید ثبت نشده است" />
  return (
    <ol className="wr-list" aria-label="پرداخت‌های ثبت‌شده">
      {attempts.map(a => (
        <li key={a.attempt_id}>
          {money(a.amount, a.unit)} · {attemptStateLabel[a.state] ?? a.state} · {formatInstant(a.submitted_at, zone)}{' '}
          <StatusPill tone={evidenceStateTone[a.evidence_state] ?? 'slate'} icon="inbox" label={evidenceStateLabel[a.evidence_state] ?? a.evidence_state} />
          <ul className="wr-list" aria-label="فیش‌ها">
            {a.receipts.map(r => (
              <li key={r.receipt_id}><StatusPill tone={receiptStateTone[r.state] ?? 'slate'} icon="inbox" label={receiptStateLabel[r.state] ?? r.state} />{' '}
                <span className="muted">{r.media_type}</span></li>
            ))}
          </ul>
        </li>
      ))}
    </ol>
  )
}

/**
 * The owning seller's manual (receipt-mode) payment submission for the
 * invoice's current stage (Slice 12). Files are staged first (bytes durable
 * before anything is recorded); the batch is reviewed in a confirmation
 * step and submitted once with one idempotency key per intent and the
 * collection version as If-Match. The server re-evaluates everything; the
 * submit action appears only when its allowed_actions says so. Nothing here
 * reviews, approves, posts or changes the stage.
 */
export function PaymentSubmissionPanel({ invoiceId }: { invoiceId: string }) {
  const api = useApi()
  const s = useSession()
  const zone = s.status === 'ready' ? s.session.display_zone : 'Asia/Tehran'
  const [reload, setReload] = useState(0)
  const view = useRead(`pay:${invoiceId}:${reload}`, async () => settle(await api.GET('/api/v1/payments/invoices/{invoice_id}/submission', {
    params: { path: { invoice_id: invoiceId } } }), e => e.data))
  const [staged, setStaged] = useState<Staged[]>([])
  const [uploadNote, setUploadNote] = useState<string | null>(null)
  const [drafts, setDrafts] = useState<ItemDraft[]>(() => [{ key: nextItemKey(), amount: '', receipts: [] }])
  const [intent, setIntent] = useState<Intent | null>(null)
  const [key, setKey] = useState('')
  const [result, setResult] = useState<PaymentBatchEnvelope | null>(null)
  const [busy, setBusy] = useState(false)
  const unknown = result?.outcome === 'UNKNOWN'

  async function stage(files: FileList | null) {
    if (!files) return
    setUploadNote(null)
    for (const file of Array.from(files)) {
      const r = await api.POST('/api/v1/payments/uploads', {
        body: file as unknown as string, bodySerializer: (b: unknown) => b as BodyInit, headers: { 'Content-Type': 'application/octet-stream' } })
      const env = (r.data ?? r.error) as PaymentUploadEnvelope | undefined
      const d = env && 'outcome' in env ? env.data : undefined
      if (env && env.outcome === 'SUCCESS' && d?.upload_id && d.content_sha256) {
        const s = { uploadId: d.upload_id, sha: d.content_sha256, name: file.name, size: file.size }
        setStaged(prev => (prev.some(p => p.uploadId === s.uploadId) ? prev : [...prev, s]))
      } else if (env && env.outcome === 'FAILED') {
        setUploadNote(`ذخیره «${file.name}» انجام نشد و چیزی ثبت نشد؛ دوباره بارگذاری کنید.`)
      } else {
        setUploadNote(`«${file.name}» پذیرفته نشد (نوع یا حجم فایل، یا دسترسی). ${env && 'reasons' in env ? reasonText(env.reasons) : ''}`)
      }
    }
  }

  function review() {
    const used = new Set<string>()
    const items: Intent['items'] = []
    for (const d of drafts) {
      const amount = Number(d.amount)
      if (!Number.isInteger(amount) || amount <= 0 || d.receipts.length === 0) return
      const receipts = staged.filter(x => d.receipts.includes(x.uploadId) && !used.has(x.uploadId))
      receipts.forEach(x => used.add(x.uploadId))
      items.push({ item_key: d.key, amount, receipts })
    }
    const next: Intent = { items, total: items.reduce((a, b) => a + b.amount, 0) }
    // A changed intent gets a new idempotency key; the same intent keeps it.
    if (JSON.stringify(next) !== JSON.stringify(intent)) setKey(newIdempotencyKey())
    setIntent(next)
    setResult(null)
  }

  async function submit(version: number, unit: string, stageNo: number) {
    if (!intent || unknown) return
    setBusy(true)
    const r = await api.POST('/api/v1/payments/invoices/{invoice_id}/batches', {
      params: { path: { invoice_id: invoiceId }, header: { 'Idempotency-Key': key, 'If-Match': String(version) } },
      body: { stage_no: stageNo, entry_mode: 'RECEIPT', items: intent.items.map(i => ({ item_key: i.item_key, amount: i.amount, unit: unit as 'IRT' | 'IRR',
        receipts: i.receipts.map(x => ({ upload_id: x.uploadId, content_sha256: x.sha })) })) } })
    const env = (r.data ?? r.error) as PaymentBatchEnvelope | undefined
    setBusy(false)
    if (env && 'outcome' in env) {
      setResult(env)
      if (env.outcome === 'SUCCESS' || env.outcome === 'CONFLICT') {
        setIntent(null)
        if (env.outcome === 'SUCCESS') { setStaged([]); setDrafts([{ key: nextItemKey(), amount: '', receipts: [] }]) }
        setReload(n => n + 1) // conflict: refresh server state, never overwrite
      }
    } else {
      setResult({ outcome: 'FAILED', reasons: [], correlation_id: '' } as PaymentBatchEnvelope)
    }
  }

  const setDraft = (i: number, d: Partial<ItemDraft>) => setDrafts(prev => prev.map((x, j) => (j === i ? { ...x, ...d } : x)))

  return (
    <Card title="ثبت پرداخت کارت‌به‌کارت با فیش (دستور کنترل‌شده)">
      <ReadStates state={view} what="وضعیت پرداخت" blockedDetail="ثبت پرداخت پشت پرچم payments.manual_write است." />
      {view.status === 'ok' && (() => {
        const c = view.data
        const allowed = actionHint(view.allowed, ACTION)
        return (
          <div className="wr-sections">
            <p className="muted">{modeLabel[c.rollout_mode]}</p>
            <dl className="wr-dl">
              <dt>مرحله جاری</dt><dd>{c.stage_no ?? 'نامعلوم'}</dd>
              <dt>مبلغ مرحله</dt><dd>{money(c.due_amount, c.unit)}</dd>
              <dt>مانده فاکتور</dt><dd>{money(c.remaining_amount, c.unit)}</dd>
              <dt>در انتظار / پذیرفته‌شده</dt><dd>{money(c.pending_or_accepted_amount, c.unit)}</dd>
              <dt>قابل ثبت</dt><dd>{money(c.available_amount, c.unit)}</dd>
              <dt>نسخه حساب وصول</dt><dd className="wr-ltr">{c.collection_version}</dd>
            </dl>
            {!c.eligible && <StateBlock kind="locked" title="ثبت پرداخت برای این فاکتور اکنون مجاز نیست"
              detail={`${reasonText(c.reasons)}${c.blocked_by.length ? ` · تصمیم‌های باز: ${codes(c.blocked_by)}` : ''}`} />}
            {result && <BatchBanner env={result} />}
            {allowed.allowed && !intent && !unknown && (
              <div className="wr-form" role="group" aria-label="ثبت پرداخت">
                <label className="lbl">فیش‌ها (jpg، png، webp یا pdf؛ هر فایل حداکثر ۵ مگابایت)
                  <input className="input" type="file" multiple accept="image/jpeg,image/png,image/webp,application/pdf"
                    onChange={e => { void stage(e.currentTarget.files) }} />
                </label>
                {uploadNote && <StateBlock kind="error" title={uploadNote} />}
                {staged.length > 0 && (
                  <ul className="wr-list" aria-label="فیش‌های بارگذاری‌شده">
                    {staged.map(x => <li key={x.uploadId}>{x.name} · <span className="muted">{uploadStateLabel.UNBOUND}</span></li>)}
                  </ul>
                )}
                <div className="pay-items">
                  {drafts.map((d, i) => (
                    <fieldset key={d.key} className="pay-item">
                      <legend>تراکنش {i + 1}</legend>
                      <label className="lbl">مبلغ ({c.unit ? unitLabel[c.unit] ?? c.unit : 'واحد نامعلوم'}؛ عدد صحیح، بدون تبدیل)
                        <input className="input pay-amount wr-ltr" inputMode="numeric" value={d.amount} onChange={e => setDraft(i, { amount: e.currentTarget.value })} />
                      </label>
                      <div className="pay-receipts" role="group" aria-label={`فیش‌های تراکنش ${i + 1}`}>
                        {staged.map(x => (
                          <label key={x.uploadId}>
                            <input type="checkbox" checked={d.receipts.includes(x.uploadId)}
                              onChange={e => setDraft(i, { receipts: e.currentTarget.checked ? [...d.receipts, x.uploadId] : d.receipts.filter(r => r !== x.uploadId) })} />
                            {x.name}
                          </label>
                        ))}
                      </div>
                    </fieldset>
                  ))}
                </div>
                <div className="wr-actions">
                  <Button onClick={() => setDrafts(prev => [...prev, { key: nextItemKey(), amount: '', receipts: [] }])}>افزودن تراکنش</Button>
                  <Button variant="primary" onClick={review}>بررسی پیش از ثبت</Button>
                </div>
              </div>
            )}
            {intent && !unknown && c.stage_no != null && c.unit != null && (
              <div className="wr-confirm" role="dialog" aria-label="تایید ثبت پرداخت">
                <dl className="wr-dl">
                  <dt>فاکتور</dt><dd className="wr-ltr">{invoiceId}</dd>
                  <dt>مرحله</dt><dd>{c.stage_no}</dd>
                  <dt>تراکنش‌ها</dt><dd>{intent.items.length} ردیف · جمع {money(intent.total, c.unit)}</dd>
                  <dt>فیش‌ها</dt><dd>{intent.items.reduce((a, b) => a + b.receipts.length, 0)} فایل</dd>
                  <dt>اثر</dt><dd>فقط ثبت تلاش پرداخت در انتظار بررسی مالی؛ تایید مالی، ثبت در دفتر، تغییر مرحله، کمیسیون و اشتراک انجام نمی‌شود. دسته کامل ثبت می‌شود یا هیچ. سامانه قدیمی همچنان مرجع است.</dd>
                  <dt>نسخه مورد انتظار</dt><dd className="wr-ltr">{c.collection_version}</dd>
                </dl>
                <div className="wr-actions">
                  <Button variant="primary" disabled={busy} onClick={() => { void submit(c.collection_version, c.unit as string, c.stage_no as number) }}>تایید و ثبت</Button>
                  <Button onClick={() => setIntent(null)}>انصراف</Button>
                </div>
              </div>
            )}
            {!allowed.allowed && (
              <p className="muted">{`ثبت پرداخت برای شما در دسترس نیست${allowed.reason_code ? ` (${allowed.reason_code})` : ''}.`}</p>
            )}
            <AttemptList attempts={c.attempts} zone={zone} />
            <div>
              <strong>اقدام‌های در دسترس نیستند (تصمیم باز)</strong>
              <ul className="wr-list" aria-label="اقدام‌های مسدود پرداخت">
                {c.blocked_actions.map(b => (
                  <li key={b.action}><StatusPill tone="slate" icon="lock" label={blockedPaymentLabel[b.action] ?? b.action} /> <span className="muted">{b.blocked_by.join('، ')}</span></li>
                ))}
              </ul>
            </div>
          </div>
        )
      })()}
    </Card>
  )
}
