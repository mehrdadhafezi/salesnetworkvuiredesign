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
