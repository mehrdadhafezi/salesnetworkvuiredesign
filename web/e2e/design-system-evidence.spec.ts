import { writeFileSync } from 'node:fs'
import { expect, test, type Page } from '@playwright/test'

// RM-E01 visual comparison evidence: the frozen prototype (prototype/index.html, Seller V1 + SN-207)
// against the React showcase (/dev/design-system), same viewport and theme.
// Runs only when DS_EVIDENCE_DIR is set:  DS_EVIDENCE_DIR=../docs/evidence/rm-e01 npx playwright test -c playwright.ds.config.ts
// External requests of the prototype (Google Fonts) are blocked so both sides render with the same
// local fallback font (Tahoma); the React app ships no web font (see the RM-E01 report).

const OUT = process.env.DS_EVIDENCE_DIR
// Prototype location from the project's testDir (web/e2e); no node:path dependency in the e2e typecheck.
const protoUrl = () => encodeURI(`file:///${test.info().project.testDir.split(String.fromCharCode(92)).join('/').replace(/^[/]/, '')}/../../prototype/index.html`)

async function proto(page: Page, query: string) {
  await page.route(/^https?:\/\//, route => route.abort())
  await page.goto(`${protoUrl()}?${query}`)
  await page.waitForFunction(() => (document.querySelector('#ws')?.children.length ?? 0) > 0)
  await page.evaluate(() => document.querySelector('.demo-switch')?.remove()) // demo-only chrome, not product UI
  await page.waitForTimeout(300)
}

async function react(page: Page, query: string) {
  await page.goto(`/dev/design-system?${query}`)
  await expect(page.locator('h1')).toBeVisible()
  await page.waitForTimeout(300)
}

const PAIRS: [string, string, string][] = [
  // [prototype selector, React selector, label]
  ['.hdr', '.ds-hdr', 'header'],
  ['.nav', '.ds-nav', 'role navigation'],
  ['.nav a[aria-current="page"]', '.ds-nav-link[aria-current="page"]', 'current nav item'],
  ['.nav a:not([aria-current]):not([hidden])', '.ds-nav-link:not([aria-current]):not([hidden])', 'nav item'],
  ['.queues', '.ds-queues', 'queue bar'],
  ['.q[aria-pressed="true"]', '.ds-q[aria-pressed="true"]', 'pressed queue'],
  ['.q[aria-pressed="false"]', '.ds-q[aria-pressed="false"]', 'queue'],
  ['.kpis', '.ds-kpis', 'metric strip'],
  ['.kpi', '.ds-kpi', 'metric'],
  ['.page-head h1', '.ds-page-head h1', 'page title'],
  ['.brand-mark', '.ds-brand-mark', 'brand mark'],
  ['.search', '.ds-search', 'search trigger'],
  ['.bottom-nav', '.ds-bottom-nav', 'bottom nav'],
  ['.bottom-nav a[aria-current="page"]', '.ds-bn-link[aria-current="page"]', 'bottom nav current'],
]
const DRAWER_PAIRS: [string, string, string][] = [
  ['.drawer', '.ds-drawer', 'drawer'],
  ['.dr-top', '.ds-dr-top', 'drawer top bar'],
  ['.dr-foot', '.ds-dr-foot', 'drawer footer'],
  ['.sheet-grip', '.ds-sheet-grip', 'sheet grip'],
]
const PROPS = ['height', 'backgroundColor', 'color', 'fontSize', 'fontWeight', 'borderRadius', 'paddingLeft', 'paddingRight', 'borderBottomColor'] as const

async function styles(page: Page, selector: string, withWidth = false) {
  return page.evaluate(([sel, props, ww]) => {
    const el = document.querySelector(sel as string)
    if (!el || !(el as HTMLElement).offsetParent && getComputedStyle(el).position !== 'fixed') return null
    const cs = getComputedStyle(el)
    const out: Record<string, string> = {}
    for (const p of props as readonly string[]) out[p] = p === 'height' ? `${Math.round(el.getBoundingClientRect().height)}px` : cs.getPropertyValue(p.replace(/[A-Z]/g, m => `-${m.toLowerCase()}`))
    if (ww) out.width = `${Math.round(el.getBoundingClientRect().width)}px`
    return out
  }, [selector, PROPS, withWidth] as const)
}

test.describe('RM-E01 visual evidence', () => {
  test.skip(!OUT, 'set DS_EVIDENCE_DIR to capture evidence')

  for (const theme of ['light', 'dark'] as const) {
    test(`screenshots + computed-style parity (${theme})`, async ({ page, viewport }) => {
      const dir = OUT ?? '.'
      const w = viewport?.width ?? 0
      const tag = `${w}-${theme}`

      await proto(page, `view=leads&queue=none&theme=${theme}&welcome=0`)
      await page.screenshot({ path: `${dir}/prototype-${tag}.png` })
      const protoStyles: Record<string, unknown> = {}
      for (const [p, , label] of PAIRS) protoStyles[label] = await styles(page, p)
      await proto(page, `view=leads&queue=none&theme=${theme}&welcome=0&lead=9043346`)
      await page.waitForTimeout(400)
      await page.screenshot({ path: `${dir}/prototype-${tag}-drawer.png` })
      for (const [p, , label] of DRAWER_PAIRS) protoStyles[label] = await styles(page, p, true)

      await page.unrouteAll()
      await react(page, `theme=${theme}`)
      await page.screenshot({ path: `${dir}/react-${tag}.png` })
      const reactStyles: Record<string, unknown> = {}
      for (const [, r, label] of PAIRS) reactStyles[label] = await styles(page, r)
      await page.screenshot({ path: `${dir}/react-${tag}-full.png`, fullPage: true })
      await react(page, `theme=${theme}&drawer=open`)
      await page.waitForTimeout(400)
      await page.screenshot({ path: `${dir}/react-${tag}-drawer.png` })
      for (const [, rr, label] of DRAWER_PAIRS) reactStyles[label] = await styles(page, rr, true)

      const rows = [...PAIRS, ...DRAWER_PAIRS].map(([, , label]) => {
        const a = protoStyles[label] as Record<string, string> | null, b = reactStyles[label] as Record<string, string> | null
        const keys = a ? Object.keys(a) : []
        const diffs = a && b ? keys.filter(p => a[p] !== b[p]).map(p => `${p}: ${a[p]} → ${b[p]}`) : []
        return { label, prototype: a, react: b, comparable: Boolean(a && b), identical: Boolean(a && b) && diffs.length === 0, diffs }
      })
      writeFileSync(`${dir}/parity-${tag}.json`, JSON.stringify(rows, null, 2) + '\n')
    })
  }
})
