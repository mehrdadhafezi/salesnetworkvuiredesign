import { Field } from '../../shared/forms/Field'
import type { Outcome } from '../../shared/api'
import { outcomePresentation } from '../../shared/ui/outcome'
import { Button, Card, PageHeader, StateBlock, StatusPill } from '../../shared/ui/primitives'

/** DEV-ONLY demonstration of the shell primitives. Never mounted in production builds. */
export function PrimitivesPage() {
  const outcomes = Object.keys(outcomePresentation) as Outcome[]
  return (
    <>
      <PageHeader title="نمونه اجزای پوسته" subtitle="فقط محیط توسعه — داده‌ها ساختگی‌اند." />
      <Card title="نتیجه‌ها (هفت حالت ثابت)">
        <div className="grid">
          {outcomes.map(o => {
            const p = outcomePresentation[o]
            return <div key={o}><StatusPill tone={p.tone} icon={p.icon} label={p.label} /> <span className="num muted">{o}</span></div>
          })}
        </div>
      </Card>
      <Card title="دکمه‌ها">
        <Button variant="primary">اقدام اصلی</Button> <Button>اقدام عادی</Button> <Button variant="ghost">اقدام کم‌رنگ</Button> <Button disabled>غیرفعال</Button>
      </Card>
      <Card title="فیلد">
        <Field label="نمونه فیلد" help="متن راهنما" placeholder="ساختگی" />
        <Field label="فیلد با خطا" error="مقدار معتبر نیست" defaultValue="—" />
      </Card>
      <Card title="حالت‌ها">
        <StateBlock kind="loading" title="بارگذاری" />
        <StateBlock kind="empty" title="خالی" />
        <StateBlock kind="unknown" title="نامعلوم" detail="ممکن است انجام شده باشد — پیش از تلاش دوباره تطبیق دهید." />
        <StateBlock kind="locked" title="مجاز نیست" />
        <StateBlock kind="error" title="خطا" />
      </Card>
    </>
  )
}
