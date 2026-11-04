// Package api is the catalog module's published contract: the service
// definition / component / scope read used by biawin and fulfillment
// through their composition roots (never a cross-schema join).
package api

import "context"

// Financial behaviour classifications. Slice 14 executes only NONE; any
// other classification is decision-blocked (D-35, D-38).
const (
	FinancialNone           = "NONE"
	FinancialConsumesCredit = "CONSUMES_CREDIT"
	FinancialChargesLedger  = "CHARGES_LEDGER"
)

// Integration providers.
const (
	ProviderNone   = "NONE"
	ProviderBiawin = "BIAWIN"
)

// Scope kinds.
const (
	ScopeSupervisor = "SUPERVISOR"
	ScopeExecutor   = "EXECUTOR"
)

// Workflow keys: versioned definitions select a pre-registered, tested Go
// workflow; there is no executable workflow language.
const WorkflowOperationalTaskV1 = "operational_task_v1"

// Completion outcomes a definition may approve.
const (
	OutcomeExecuted    = "EXECUTED"
	OutcomeNotExecuted = "NOT_EXECUTED"
)

// EvidenceRequirement is one evidence type of a service version.
type EvidenceRequirement struct {
	Type     string `json:"type"`
	Required bool   `json:"required"`
}

// ServiceVersion is one immutable service definition version.
type ServiceVersion struct {
	ServiceKey          string                `json:"service_key"`
	Version             int                   `json:"version"`
	Title               string                `json:"title"`
	WorkflowKey         string                `json:"workflow_key"`
	Evidence            []EvidenceRequirement `json:"evidence_requirements"`
	CompletionOutcomes  []string              `json:"completion_outcomes"`
	FinancialBehaviour  string                `json:"financial_behaviour"`
	IntegrationProvider string                `json:"integration_provider"`
	Status              string                `json:"status"`
	ApprovalRef         string                `json:"approval_ref"`
}

// Component is one service of a subscription product version.
type Component struct {
	Seq            int    `json:"seq"`
	ServiceKey     string `json:"service_key"`
	ServiceVersion int    `json:"service_version"`
}

// Reader is the catalog published read.
type Reader interface {
	// Service returns one version; ok=false when it does not exist.
	Service(ctx context.Context, key string, version int) (ServiceVersion, bool, error)
	// Components returns the services of a product version (seq order).
	Components(ctx context.Context, productKey string, productVersion int) ([]Component, error)
	// Supervisors returns the CURRENT supervisors of a service.
	Supervisors(ctx context.Context, serviceKey string) ([]string, error)
	// InScope reports whether user currently holds scope kind on service.
	InScope(ctx context.Context, serviceKey, userID, kind string) (bool, error)
	// ServicesInScope returns the services where user CURRENTLY holds any
	// scope (supervisor or executor); never every service.
	ServicesInScope(ctx context.Context, userID string) ([]string, error)
}
