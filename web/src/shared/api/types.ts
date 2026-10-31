// Contract types, re-exported from the generated OpenAPI client. Never
// hand-write transport DTOs: regenerate with `npm run openapi:gen`.
import type { components, paths } from '../../generated/api/schema'

export type { components, paths }

type Schemas = components['schemas']

export type Outcome = Schemas['Outcome']
export type ReasonCode = Schemas['ReasonCode']
export type Problem = Schemas['Problem']
export type FieldError = Schemas['FieldError']
export type Money = Schemas['Money']
export type Unit = Schemas['Unit']
export type PageInfo = Schemas['PageInfo']
export type AllowedAction = Schemas['AllowedAction']
export type AllowedActions = Schemas['AllowedActions']
export type HealthLive = Schemas['HealthLive']
export type HealthReady = Schemas['HealthReady']
export type SessionContext = Schemas['SessionContext']
export type SessionEnvelope = Schemas['SessionEnvelope']

// Slice 4 read-only identity / organization / authorization contract types.
export type UserSummary = Schemas['UserSummary']
export type PersonSummary = Schemas['PersonSummary']
export type PersonDetail = Schemas['PersonDetail']
export type Employment = Schemas['Employment']
export type Assignment = Schemas['Assignment']
export type HistoryEntry = Schemas['HistoryEntry']
export type Finding = Schemas['Finding']
export type FindingKind = Schemas['FindingKind']
export type OrgUnit = Schemas['OrgUnit']
export type Lookup = Schemas['Lookup']
export type PersonRef = Schemas['PersonRef']
export type EmploymentStatus = Schemas['EmploymentStatus']
export type EffectiveAccess = Schemas['EffectiveAccess']

// Slice 5 read-only customer identity (shadow) contract types. There is no
// merge, resolve or edit type: the contract has none.
export type ResolutionState = Schemas['ResolutionState']
export type Strength = Schemas['Strength']
export type CaseReason = Schemas['CaseReason']
export type EvidenceKind = Schemas['EvidenceKind']
export type CustomerSummary = Schemas['CustomerSummary']
export type CustomerDetail = Schemas['CustomerDetail']
export type ContactPoint = Schemas['ContactPoint']
export type CustomerIdentifier = Schemas['Identifier']
export type IdentityEvidence = Schemas['Evidence']
export type MergeCase = Schemas['MergeCase']
export type ResolutionReport = Schemas['ResolutionReport']

// Slice 6 read-only sales (cases + custody) contract types. No action,
// assignment or transfer type exists: the contract has none.
export type WorkflowType = Schemas['WorkflowType']
export type CaseSummary = Schemas['CaseSummary']
export type CaseDetail = Schemas['CaseDetail']
export type CaseAlias = Schemas['CaseAlias']
export type StateHistoryEntry = Schemas['StateHistoryEntry']
export type AttributionSnapshot = Schemas['AttributionSnapshot']
export type CustodyAssignment = Schemas['CustodyAssignment']
export type CustodyEvent = Schemas['CustodyEvent']
export type CustodyFinding = Schemas['CustodyFinding']
export type CaseCustody = Schemas['CaseCustody']
export type Parity = Schemas['Parity']
export type Facet = Schemas['Facet']
export type HonestMetric = Schemas['HonestMetric']
export type SalesCoverageReport = Schemas['SalesCoverageReport']
export type CustodyParityReport = Schemas['CustodyParityReport']
export type UserRef = Schemas['UserRef']
