/*
 * RM-E01 shared CRM design system (presentation only).
 * Source of truth: prototype/shared/CRM-DESIGN-SYSTEM-V1.md (SN-207) and the frozen
 * Seller V1 prototype. Traceability: docs/implementation/rm-e01-shared-design-system.md.
 * Importing this module loads ds.css (namespaced `ds-`); tokens come from tokens.css.
 */
import './ds.css'

export { DsIcon, dsIconNames, type DsIconName } from './icons'
export { toFaDigits, formatFaNumber, isKnownCount, countText, countLabel, type DsCount } from './format'
export {
  StatusIndicator, TonePill, StatePanel, Banner, CountBadge, Metric, MetricGroup, dataStatePresentation,
  type DataState, type PanelStateKind, type BannerKind, type MetricState,
} from './Status'
export { Menu, MenuButton, Popover, type MenuEntry } from './Menu'
export {
  RoleNavigation, MobileBottomNavigation, QueueTabs, computeNavOverflow,
  type DsNavItem, type DsQueue, type DsLinkRenderer,
} from './navigation'
export { Drawer, DrawerSection } from './Drawer'
export { DsAppShell, AppHeader, HeaderIconButton, PageHead, Panel, PanelHeader, Toolbar, Freshness } from './Layout'
export { ContextHint, Glossary, ShortcutList, HelpCenterButton, type GlossaryEntry, type ShortcutEntry, type HelpTopic } from './Help'
