import { Link } from 'react-router'
import { PageHeader, StateBlock } from '../shared/ui/primitives'

export function NotFoundPage() {
  return (
    <>
      <PageHeader title="صفحه پیدا نشد" />
      <StateBlock kind="empty" title="این مسیر وجود ندارد"
        action={<Link className="btn" to="/">بازگشت به خانه</Link>} />
    </>
  )
}
