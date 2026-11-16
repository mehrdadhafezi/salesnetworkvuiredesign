import { expect, test, type Page } from '@playwright/test'

// RM-E01 — real-browser checks of the shared design-system components on the DEV-only
// showcase (/dev/design-system). Synthetic content only; no API, no database, no business action.
// Run: npx playwright test -c playwright.ds.config.ts

const URL = '/dev/design-system'

async function open(page: Page) {
  await page.goto(URL)
  await expect(page.locator('h1')).toBeVisible() // the route is lazy-loaded
}

/** Wait until the drawer's slide transition has settled. */
async function settledBox(page: Page, name: string) {
  const drawer = page.getByRole('dialog', { name })
  await expect(drawer).toBeVisible()
  await drawer.evaluate(el => Promise.all(el.getAnimations().map(a => a.finished)))
  return drawer.boundingBox()
}

async function noPageHorizontalScroll(page: Page) {
  const overflow = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)
  expect(overflow).toBeLessThanOrEqual(0)
}

test.describe('desktop', () => {
  test.skip(({ viewport }) => (viewport?.width ?? 0) < 761, 'desktop only')

  test('landmarks, one h1, RTL, skip link moves focus to main', async ({ page }) => {
    await open(page)
    await expect(page.locator('.ds-showcase-root')).toHaveAttribute('dir', 'rtl')
    await expect(page.locator('h1')).toHaveCount(1)
    await expect(page.getByRole('banner')).toBeVisible()
    await expect(page.getByRole('navigation', { name: 'بخش‌های پنل نمونه' })).toBeVisible()
    await expect(page.getByRole('navigation', { name: 'ناوبری موبایل' })).toBeHidden()
    await page.keyboard.press('Tab')
    const skip = page.getByRole('link', { name: 'پرش به محتوای اصلی' })
    await expect(skip).toBeFocused()
    await expect(skip).toBeInViewport()
    await page.keyboard.press('Enter')
    await expect(page.locator('main#ws')).toBeFocused()
    await noPageHorizontalScroll(page)
  })

  test('RoleNavigation: aria-current follows selection; unknown count is «—», not 0', async ({ page }) => {
    await open(page)
    const nav = page.getByRole('navigation', { name: 'بخش‌های پنل نمونه' })
    await expect(nav.locator('[aria-current="page"]')).toHaveCount(1)
    await expect(nav.locator('[aria-current="page"]')).toContainText('شماره‌های من')
    await nav.getByRole('link', { name: /فاکتورها/ }).click()
    await expect(nav.locator('[aria-current="page"]')).toContainText('فاکتورها')
    await expect(nav.getByRole('link', { name: /فاکتورها/ })).toContainText('—')
    await expect(nav.getByRole('link', { name: /فاکتورها/ })).toHaveAccessibleName(/تعداد نامعلوم/)
    // A known zero shows no badge (prototype) and never stands in for unknown.
    await expect(nav.getByRole('link', { name: 'نیاز به اقدام مجدد' })).not.toContainText('۰')
  })

  test('RoleNavigation: priority+ overflow keeps the current item and lists the rest under «بیشتر»', async ({ page }) => {
    await page.setViewportSize({ width: 820, height: 768 })
    await open(page)
    const nav = page.getByRole('navigation', { name: 'بخش‌های پنل نمونه' })
    const more = nav.getByRole('button', { name: /بیشتر/ })
    await expect(more).toBeVisible()
    await expect(nav.locator('[aria-current="page"]')).toBeVisible()
    const scroll = await nav.evaluate(n => n.scrollWidth - n.clientWidth)
    expect(scroll).toBeLessThanOrEqual(1)
    await more.focus()
    await page.keyboard.press('Enter')
    const menu = page.getByRole('menu', { name: 'بیشتر' })
    await expect(menu).toBeVisible()
    await expect(menu.getByRole('menuitem').first()).toBeFocused()
    await page.keyboard.press('Escape')
    await expect(menu).toBeHidden()
    await expect(more).toBeFocused()
    await noPageHorizontalScroll(page)
  })

  test('QueueTabs: pressed state, RTL arrow keys, unknown count', async ({ page }) => {
    await open(page)
    const group = page.getByRole('group', { name: 'صف‌ها' })
    const first = group.getByRole('button', { name: /بدون وضعیت/ })
    await expect(first).toHaveAttribute('aria-pressed', 'true')
    await first.focus()
    await page.keyboard.press('ArrowLeft') // RTL: visual next
    await expect(group.getByRole('button', { name: /جواب نداده/ })).toBeFocused()
    await page.keyboard.press('ArrowRight')
    await expect(first).toBeFocused()
    await page.keyboard.press('End')
    const all = group.getByRole('button', { name: /همه/ })
    await expect(all).toBeFocused()
    await page.keyboard.press('Enter')
    await expect(all).toHaveAttribute('aria-pressed', 'true')
    await expect(first).toHaveAttribute('aria-pressed', 'false')
    await expect(all).toContainText('—')
    await expect(group.getByRole('button', { name: /عدم خرید/ })).toContainText('۰')
    // RTL: the first queue sits to the right of the second.
    const a = await first.boundingBox(), b = await group.getByRole('button', { name: /جواب نداده/ }).boundingBox()
    expect(a && b && a.x > b.x).toBe(true)
  })

  test('Drawer: opens at the inline-end edge, takes focus, Esc closes and restores focus', async ({ page }) => {
    await open(page)
    const opener = page.getByRole('button', { name: 'باز کردن کشو' })
    await opener.focus()
    await page.keyboard.press('Enter')
    const drawer = page.getByRole('dialog', { name: 'پرونده نمونه ۱۰۴ (ساختگی)' })
    await expect(drawer).toBeVisible()
    await expect(drawer).toHaveAttribute('aria-modal', 'false')
    await expect(drawer).toBeFocused()
    const box = await settledBox(page, 'پرونده نمونه ۱۰۴ (ساختگی)')
    expect(box?.x).toBe(0) // RTL: the drawer sits at the left (inline-end) edge
    expect(Math.round(box?.width ?? 0)).toBe(560)
    await page.keyboard.press('Escape')
    await expect(drawer).toBeHidden()
    await expect(opener).toBeFocused()
    // Wide drawer (660px) and close button.
    await page.getByRole('button', { name: 'کشوی پهن' }).click()
    expect(Math.round((await settledBox(page, 'پرونده نمونه ۱۰۴ (ساختگی)'))?.width ?? 0)).toBe(660)
    await drawer.getByRole('button', { name: 'بستن' }).first().click()
    await expect(drawer).toBeHidden()
  })

  test('Menu: keyboard roving focus; a disabled item stays visible with its reason and does nothing', async ({ page }) => {
    await open(page)
    const trigger = page.getByRole('button', { name: 'اقدام‌های ردیف نمونه' })
    await trigger.focus()
    await page.keyboard.press('ArrowDown')
    const menu = page.getByRole('menu', { name: 'اقدام‌های ردیف نمونه' })
    await expect(menu).toBeVisible()
    await expect(menu.getByRole('menuitem', { name: /مشاهده جزئیات/ })).toBeFocused()
    await page.keyboard.press('End')
    const denied = menu.getByRole('menuitem', { name: /اقدام بدون مجوز/ })
    await expect(denied).toBeFocused()
    await expect(denied).toHaveAttribute('aria-disabled', 'true')
    await expect(denied).toHaveAccessibleDescription(/پنهان نمی‌شود/)
    await page.keyboard.press('Enter')
    await expect(menu).toBeVisible()
    await expect(page.getByRole('status').filter({ hasText: 'هنوز اقدامی انجام نشده است.' })).toBeVisible()
    await page.keyboard.press('Escape')
    await expect(menu).toBeHidden()
    await expect(trigger).toBeFocused()
  })

  test('Popover and help: Esc closes and returns focus; outside click closes', async ({ page }) => {
    await open(page)
    const help = page.getByRole('button', { name: 'راهنما', exact: true })
    await help.click()
    const hc = page.getByRole('dialog', { name: 'مرکز راهنما' })
    await expect(hc).toBeVisible()
    await page.keyboard.press('Escape')
    await expect(hc).toBeHidden()
    await expect(help).toBeFocused()
    const pop = page.getByRole('button', { name: 'پاپ‌اور توضیح' })
    await pop.click()
    await expect(page.getByRole('dialog', { name: 'توضیح نمونه' })).toBeVisible()
    await page.locator('h1').click()
    await expect(page.getByRole('dialog', { name: 'توضیح نمونه' })).toBeHidden()
  })

  test('visible focus indicator uses the --ring token', async ({ page }) => {
    await open(page)
    const btn = page.getByRole('button', { name: 'باز کردن کشو' })
    await btn.focus()
    await page.keyboard.press('Shift+Tab')
    await page.keyboard.press('Tab')
    const outline = await btn.evaluate(el => getComputedStyle(el).outlineStyle + ' ' + getComputedStyle(el).outlineWidth)
    expect(outline).toBe('solid 2px')
  })
})

test.describe('mobile', () => {
  test.skip(({ viewport }) => (viewport?.width ?? 9999) > 760, 'mobile only')

  test('bottom navigation replaces the tab row; «بیشتر» opens the remaining destinations', async ({ page }) => {
    await open(page)
    await expect(page.getByRole('navigation', { name: 'بخش‌های پنل نمونه' })).toBeHidden()
    const bottom = page.getByRole('navigation', { name: 'ناوبری موبایل' })
    await expect(bottom).toBeVisible()
    const box = await bottom.boundingBox()
    expect(Math.round((box?.y ?? 0) + (box?.height ?? 0))).toBe(844)
    await expect(bottom.locator('[aria-current="page"]')).toContainText('شماره‌ها')
    const more = bottom.getByRole('button', { name: 'بیشتر' })
    await more.click()
    const menu = page.getByRole('menu', { name: 'بیشتر' })
    await expect(menu).toBeVisible()
    await menu.getByRole('menuitem', { name: /رفتار مشتریان/ }).click()
    await expect(menu).toBeHidden()
    await expect(more).toHaveAttribute('aria-current', 'page')
    await expect(page.locator('.ds-m-title')).toHaveText('رفتار مشتریان')
    await noPageHorizontalScroll(page)
  })

  test('drawer becomes a bottom sheet with a grip; Esc closes', async ({ page }) => {
    await open(page)
    await page.getByRole('button', { name: 'باز کردن کشو' }).click()
    const drawer = page.getByRole('dialog', { name: 'پرونده نمونه ۱۰۴ (ساختگی)' })
    const box = await settledBox(page, 'پرونده نمونه ۱۰۴ (ساختگی)')
    expect(Math.round(box?.width ?? 0)).toBe(390)
    expect(Math.round((box?.y ?? 0) + (box?.height ?? 0))).toBe(844)
    await expect(drawer.locator('.ds-sheet-grip')).toBeVisible()
    await page.keyboard.press('Escape')
    await expect(drawer).toBeHidden()
  })

  test('touch targets are at least 40px tall and the page never scrolls sideways', async ({ page }) => {
    await open(page)
    const heights = await page.locator('.ds-q, .ds-ws .btn:not(.btn-sm)').evaluateAll(els => els.filter(e => (e as HTMLElement).offsetParent).map(e => e.getBoundingClientRect().height))
    expect(heights.length).toBeGreaterThan(0)
    for (const h of heights) expect(h).toBeGreaterThanOrEqual(40)
    await noPageHorizontalScroll(page)
  })
})
