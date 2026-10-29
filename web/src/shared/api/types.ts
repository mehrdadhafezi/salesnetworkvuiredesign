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
