/**
 * RM-E02 role navigation contracts (presentation only; they grant nothing).
 *
 * Each role lists the destinations its product / design spec defines, in spec
 * order, with the status the R7 consolidated audit (§4, docs/audit/r7-consolidated-audit.md)
 * verified for `web/src`. Only sections mapped to an EXISTING route id carry
 * `destinations`; MISSING / BLOCKED sections are documentation for the
 * navigation contract and are never rendered as links (no fake screens).
 *
 * Section names are English contract identifiers, not UI copy: Persian labels of
 * screens that do not exist yet belong to the role's design freeze (RM-E03+).
 * The links themselves always show the existing page label.
 *
 * Role keys are the server's (`internal/identity/api`, matched_roles of
 * effective-access). The SPA never infers a role from anything else.
 */

export type RoleKey = 'seller' | 'supervisor' | 'senior_supervisor' | 'sales_manager' | 'sales_deputy' | 'mis' | 'hr' | 'finance' | 'invoice_reviewer'

/** Design authority status (AGENTS.md: READY_FOR_REVIEW is not frozen). */
export type DesignStatus = 'FROZEN' | 'FREEZE_UNVERIFIED' | 'READY_FOR_REVIEW' | 'NO_DESIGN'

/** R7 §4 status of a spec destination in web/src. */
export type SectionStatus = 'TECHNICAL_READ' | 'PARTIAL' | 'MISSING' | 'BLOCKED'

export interface RoleSection {
  section: string
  status: SectionStatus
  /** Existing destination ids (destinations.ts) that partially or technically serve this section. */
  destinations?: readonly string[]
  /** Decisions / gaps that block it (R7 §4, §9). */
  blockedBy?: readonly string[]
}

export interface RoleContract {
  key: RoleKey
  /** Role name shown as the header crumb (role names of ROLE-ACCESS-MAP.fa.md). */
  label: string
  productSpec: string
  design: { status: DesignStatus; ref: string }
  sections: readonly RoleSection[]
}

const REPORTS = ['reports.cases', 'reports.custody', 'reports.invoices', 'reports.payments'] as const

export const ROLE_CONTRACTS: Readonly<Record<RoleKey, RoleContract>> = {
  seller: {
    key: 'seller', label: 'فروشنده',
    productSpec: 'docs/source-of-truth/CRM-CROSS-ROLE-ARCHITECTURE-V1.md (Seller)',
    design: { status: 'FROZEN', ref: 'prototype/index.html (Seller Design V1)' },
    sections: [
      { section: 'leads queues + outcome drawer', status: 'PARTIAL', destinations: ['sales.cases'] },
      { section: 'conversions', status: 'MISSING' },
      { section: 'repeat action', status: 'MISSING' },
      { section: 'invoices', status: 'TECHNICAL_READ', destinations: ['billing.invoices'] },
      { section: 'customer behaviour', status: 'MISSING' },
      { section: 'extra-number request', status: 'MISSING', blockedBy: ['OPD-10'] },
      { section: 'wallet', status: 'MISSING' },
    ],
  },
  supervisor: {
    key: 'supervisor', label: 'سرپرست',
    productSpec: 'prototype/supervisor/SUPERVISOR-DESIGN-SPEC.md',
    design: { status: 'FREEZE_UNVERIFIED', ref: 'prototype/supervisor/ (own spec: DESIGN PROTOTYPE; "Frozen" only asserted by SENIOR-SUPERVISOR-PRODUCT-SPEC.md)' },
    sections: [
      { section: 'team view', status: 'MISSING' },
      { section: 'assign and return', status: 'MISSING' },
      { section: 'conversion', status: 'MISSING' },
      { section: 'invoices and exceptions', status: 'MISSING' },
      { section: 'customer history', status: 'TECHNICAL_READ', destinations: ['customers.list'] },
      { section: 'HR requests', status: 'PARTIAL', destinations: ['hr.requests'] },
      { section: 'reports', status: 'MISSING' },
      { section: 'wallet', status: 'MISSING' },
    ],
  },
  senior_supervisor: {
    key: 'senior_supervisor', label: 'سرپرست ارشد',
    productSpec: 'docs/source-of-truth/SENIOR-SUPERVISOR-PRODUCT-SPEC.md',
    design: { status: 'READY_FOR_REVIEW', ref: 'prototype/senior/SENIOR-DESIGN-SPEC.md' },
    sections: [
      { section: 'overview', status: 'MISSING' },
      { section: 'performance explorer', status: 'MISSING' },
      { section: 'distribution and return', status: 'MISSING' },
      { section: 'ready team view', status: 'MISSING' },
      { section: 'invoices and exceptions', status: 'MISSING' },
      { section: 'HR', status: 'PARTIAL', destinations: ['hr.requests'] },
      { section: 'reports', status: 'TECHNICAL_READ', destinations: REPORTS },
      { section: 'case explorer', status: 'TECHNICAL_READ', destinations: ['sales.cases'] },
      { section: 'wallet', status: 'MISSING' },
    ],
  },
  sales_manager: {
    key: 'sales_manager', label: 'مدیر فروش',
    productSpec: 'docs/source-of-truth/SALES-MANAGER-PRODUCT-SPEC.md',
    design: { status: 'READY_FOR_REVIEW', ref: 'prototype/manager/MANAGER-DESIGN-SPEC.md' },
    sections: [
      { section: 'operations and exceptions', status: 'MISSING' },
      { section: 'distribution', status: 'MISSING' },
      { section: 'return', status: 'MISSING' },
      { section: 'performance and reconciliation', status: 'MISSING' },
      { section: 'invoices', status: 'MISSING' },
      { section: 'archive explorer', status: 'BLOCKED', blockedBy: ['OP-20'] },
      { section: 'extra-number requests', status: 'BLOCKED', blockedBy: ['OPD-10'] },
      { section: 'HR', status: 'PARTIAL', destinations: ['hr.requests'] },
      { section: 'reports', status: 'TECHNICAL_READ', destinations: REPORTS },
      { section: 'customer behaviour', status: 'MISSING' },
      { section: 'wallet', status: 'MISSING' },
    ],
  },
  sales_deputy: {
    key: 'sales_deputy', label: 'معاون فروش',
    productSpec: 'docs/source-of-truth/SALES-DEPUTY-PRODUCT-SPEC.md',
    design: { status: 'READY_FOR_REVIEW', ref: 'prototype/deputy/DEPUTY-DESIGN-SPEC.md' },
    sections: [
      { section: 'leadership overview', status: 'MISSING' },
      { section: 'hierarchy and performance', status: 'MISSING' },
      { section: 'exceptions (11 categories)', status: 'MISSING' },
      { section: 'conditional allocation', status: 'MISSING' },
      { section: 'return', status: 'MISSING' },
      { section: 'scoped invoices', status: 'MISSING' },
      { section: 'HR context', status: 'MISSING' },
      { section: 'archive / extra-number monitor', status: 'MISSING' },
      { section: 'diagnostics', status: 'PARTIAL', destinations: ['mis.reconciliation'] },
      { section: 'reports', status: 'MISSING' },
      { section: 'wallet', status: 'MISSING' },
    ],
  },
  mis: {
    key: 'mis', label: 'MIS',
    productSpec: 'docs/source-of-truth/MIS-PRODUCT-SPEC.md',
    design: { status: 'READY_FOR_REVIEW', ref: 'prototype/mis/MIS-DESIGN-SPEC.md' },
    sections: [
      { section: 'today', status: 'TECHNICAL_READ', destinations: ['mis.today'] },
      { section: 'import and data quality', status: 'MISSING' },
      { section: 'source rows / lineage', status: 'MISSING' },
      { section: 'custody and delivery', status: 'TECHNICAL_READ', destinations: ['sales.custody'] },
      { section: 'reports and reconciliation', status: 'PARTIAL', destinations: [...REPORTS, 'mis.reconciliation', 'mis.quality'] },
      { section: 'planning', status: 'MISSING' },
      { section: 'quick delivery', status: 'MISSING' },
      { section: 'maintenance', status: 'MISSING' },
      { section: 'diagnostics', status: 'MISSING' },
    ],
  },
  hr: {
    key: 'hr', label: 'منابع انسانی',
    productSpec: 'docs/source-of-truth/HR-PRODUCT-SPEC.md',
    design: { status: 'READY_FOR_REVIEW', ref: 'prototype/hr/HR-DESIGN-SPEC.md' },
    sections: [
      { section: 'workforce explorer', status: 'PARTIAL', destinations: ['hr.workforce'] },
      { section: 'employment / structure timeline', status: 'TECHNICAL_READ', destinations: ['hr.structure'] },
      { section: 'onboarding and transfer', status: 'BLOCKED', blockedBy: ['OPD-05'] },
      { section: 'termination and handover', status: 'BLOCKED', blockedBy: ['OPD-05', 'SD-04'] },
      { section: 'requests', status: 'PARTIAL', destinations: ['hr.requests'] },
      { section: 'structure and access', status: 'BLOCKED', blockedBy: ['D-34'] },
      { section: 'compensation', status: 'MISSING' },
      { section: 'credentials (restricted)', status: 'BLOCKED', blockedBy: ['D-34', 'R5-SEC-01'] },
      { section: 'exceptions and audit', status: 'PARTIAL', destinations: ['hr.exceptions'] },
    ],
  },
  finance: {
    key: 'finance', label: 'مالی',
    productSpec: 'docs/source-of-truth/FINANCE-PRODUCT-SPEC.md',
    design: { status: 'READY_FOR_REVIEW', ref: 'prototype/finance/FINANCE-DESIGN-SPEC.md' },
    sections: [
      { section: 'review queue', status: 'TECHNICAL_READ', destinations: ['finance.review'] },
      { section: 'reconciliation and refund', status: 'TECHNICAL_READ', destinations: ['finance.reconciliation'], blockedBy: ['OPD-04 (refund)'] },
      { section: 'ledger', status: 'TECHNICAL_READ', destinations: ['finance.dashboard', 'finance.ledger', 'finance.balances'] },
      { section: 'rules, runs and posting', status: 'BLOCKED', blockedBy: ['D-37', 'D-38'] },
      { section: 'reports', status: 'MISSING' },
      { section: 'audit', status: 'MISSING' },
      { section: 'configuration', status: 'MISSING' },
      { section: 'maintenance', status: 'MISSING' },
    ],
  },
  invoice_reviewer: {
    key: 'invoice_reviewer', label: 'بازبین فاکتور',
    productSpec: 'docs/source-of-truth/ROLE-ACCESS-MAP.fa.md',
    design: { status: 'NO_DESIGN', ref: '—' },
    sections: [
      { section: 'invoice review', status: 'BLOCKED', blockedBy: ['D-24 (API only, no page)'] },
    ],
  },
}

export function isRoleKey(v: string): v is RoleKey {
  return Object.prototype.hasOwnProperty.call(ROLE_CONTRACTS, v)
}

/** Existing destination ids a role contract maps, in spec order, without duplicates. */
export function roleDestinationIds(role: RoleKey): string[] {
  const out: string[] = []
  for (const s of ROLE_CONTRACTS[role].sections) for (const id of s.destinations ?? []) if (!out.includes(id)) out.push(id)
  return out
}
