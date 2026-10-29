import { Card, PageHeader, StateBlock } from '../shared/ui/primitives'

/** Synthetic placeholder: no business data, no KPIs exist yet. */
export function HomePage() {
  return (
    <>
      <PageHeader title="خانه" subtitle="پوسته سامانه CRM — بخش‌های کسب‌وکار در گام‌های بعدی افزوده می‌شوند." />
      <Card>
        <StateBlock kind="empty" title="هنوز بخشی فعال نیست"
          detail="این پوسته فقط زیرساخت نمایش را دارد. هیچ داده، شاخص یا اقدام تجاری در این نسخه وجود ندارد." />
      </Card>
    </>
  )
}
