package apiserver

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/result"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/kernel/timeutil"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/authstub"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx"
	"github.com/mehrdadhafezi/salesnetworkvuiredesign/internal/platform/httpx/oapi"
)

// ContractPaths are the paths of the OpenAPI contract with their methods.
// They drive the central 404/405 problem handling; TestRoutesMatchContract
// fails if they drift from the bundled contract.
var ContractPaths = map[string][]string{
	"/health/live":                             {http.MethodGet},
	"/health/ready":                            {http.MethodGet},
	"/api/v1/session":                          {http.MethodGet},
	"/api/v1/identity/users":                   {http.MethodGet},
	"/api/v1/identity/users/{user_id}":         {http.MethodGet},
	"/api/v1/organization/persons":             {http.MethodGet},
	"/api/v1/organization/persons/{person_id}": {http.MethodGet},
	"/api/v1/organization/positions":           {http.MethodGet},
	"/api/v1/organization/levels":              {http.MethodGet},
	"/api/v1/organization/org-units":           {http.MethodGet},
	"/api/v1/organization/assignments":         {http.MethodGet},
	"/api/v1/organization/hierarchy-findings":  {http.MethodGet},
	"/api/v1/authorization/effective-access":   {http.MethodGet},
	// Slice 5 customer identity (read/shadow only).
	"/api/v1/customers":                                     {http.MethodGet},
	"/api/v1/customers/{customer_id}":                       {http.MethodGet},
	"/api/v1/customers/{customer_id}/contact-points":        {http.MethodGet},
	"/api/v1/customers/{customer_id}/evidence":              {http.MethodGet},
	"/api/v1/customer-identity/merge-cases":                 {http.MethodGet},
	"/api/v1/customer-identity/merge-cases/{merge_case_id}": {http.MethodGet},
	"/api/v1/customer-identity/resolution-report":           {http.MethodGet},
}

// ContractQuery is the allowlist of query parameters per Slice 4 path. A
// parameter outside it is a 400 CONTRACT_INVALID problem rather than being
// ignored (Architecture V1 §23). TestQueryAllowlistMatchesContract keeps it
// equal to the contract.
var ContractQuery = map[string][]string{
	"/api/v1/identity/users":                                {"q", "presence", "sort", "cursor", "limit"},
	"/api/v1/identity/users/{user_id}":                      {},
	"/api/v1/organization/persons":                          {"q", "employment_status", "position_id", "level_id", "org_unit_id", "is_active", "has_findings", "sort", "cursor", "limit"},
	"/api/v1/organization/persons/{person_id}":              {},
	"/api/v1/organization/positions":                        {"cursor", "limit"},
	"/api/v1/organization/levels":                           {"cursor", "limit"},
	"/api/v1/organization/org-units":                        {"kind", "cursor", "limit"},
	"/api/v1/organization/assignments":                      {"person_id", "relation", "is_current", "relationship_kind", "has_findings", "cursor", "limit"},
	"/api/v1/organization/hierarchy-findings":               {"kind", "severity", "subject_type", "person_id", "cursor", "limit"},
	"/api/v1/authorization/effective-access":                {},
	"/api/v1/customers":                                     {"state", "has_conflict", "presence", "cursor", "limit"},
	"/api/v1/customers/{customer_id}":                       {},
	"/api/v1/customers/{customer_id}/contact-points":        {},
	"/api/v1/customers/{customer_id}/evidence":              {"strength", "cursor", "limit"},
	"/api/v1/customer-identity/merge-cases":                 {"state", "evidence_strength", "reason", "customer_id", "cursor", "limit"},
	"/api/v1/customer-identity/merge-cases/{merge_case_id}": {},
	"/api/v1/customer-identity/resolution-report":           {},
}

// CustomersAPI is the Slice 5 customer identity read surface (customers
// module, composed by cmd/crm-api). Without one, every operation answers
// BLOCKED / FEATURE_DISABLED.
type CustomersAPI interface {
	ListCustomers(ctx context.Context, request oapi.ListCustomersRequestObject) (oapi.ListCustomersResponseObject, error)
	GetCustomer(ctx context.Context, request oapi.GetCustomerRequestObject) (oapi.GetCustomerResponseObject, error)
	ListCustomerContactPoints(ctx context.Context, request oapi.ListCustomerContactPointsRequestObject) (oapi.ListCustomerContactPointsResponseObject, error)
	ListCustomerEvidence(ctx context.Context, request oapi.ListCustomerEvidenceRequestObject) (oapi.ListCustomerEvidenceResponseObject, error)
	ListMergeCases(ctx context.Context, request oapi.ListMergeCasesRequestObject) (oapi.ListMergeCasesResponseObject, error)
	GetMergeCase(ctx context.Context, request oapi.GetMergeCaseRequestObject) (oapi.GetMergeCaseResponseObject, error)
	GetCustomerResolutionReport(ctx context.Context, request oapi.GetCustomerResolutionReportRequestObject) (oapi.GetCustomerResolutionReportResponseObject, error)
	// CustomersEnabled reports the Slice 5 flag for the session navigation hint.
	CustomersEnabled(ctx context.Context) bool
}

// ReadAPI is the Slice 4 read surface (identity, organization,
// authorization). Business modules implement it and cmd/crm-api composes
// it; platform code never imports a business module. Without one, every
// Slice 4 operation answers BLOCKED / FEATURE_DISABLED.
type ReadAPI interface {
	GetEffectiveAccess(ctx context.Context, request oapi.GetEffectiveAccessRequestObject) (oapi.GetEffectiveAccessResponseObject, error)
	ListIdentityUsers(ctx context.Context, request oapi.ListIdentityUsersRequestObject) (oapi.ListIdentityUsersResponseObject, error)
	GetIdentityUser(ctx context.Context, request oapi.GetIdentityUserRequestObject) (oapi.GetIdentityUserResponseObject, error)
	ListOrganizationPersons(ctx context.Context, request oapi.ListOrganizationPersonsRequestObject) (oapi.ListOrganizationPersonsResponseObject, error)
	GetOrganizationPerson(ctx context.Context, request oapi.GetOrganizationPersonRequestObject) (oapi.GetOrganizationPersonResponseObject, error)
	ListOrganizationPositions(ctx context.Context, request oapi.ListOrganizationPositionsRequestObject) (oapi.ListOrganizationPositionsResponseObject, error)
	ListOrganizationLevels(ctx context.Context, request oapi.ListOrganizationLevelsRequestObject) (oapi.ListOrganizationLevelsResponseObject, error)
	ListOrganizationOrgUnits(ctx context.Context, request oapi.ListOrganizationOrgUnitsRequestObject) (oapi.ListOrganizationOrgUnitsResponseObject, error)
	ListOrganizationAssignments(ctx context.Context, request oapi.ListOrganizationAssignmentsRequestObject) (oapi.ListOrganizationAssignmentsResponseObject, error)
	ListOrganizationHierarchyFindings(ctx context.Context, request oapi.ListOrganizationHierarchyFindingsRequestObject) (oapi.ListOrganizationHierarchyFindingsResponseObject, error)
	// HRReadEnabled reports the Slice 4 flag for the session navigation hint.
	HRReadEnabled(ctx context.Context) bool
}

// displayZone is the presentation default zone; never storage truth.
const displayZone = oapi.SessionContextDisplayZone("Asia/Tehran")

type api struct {
	ReadAPI
	CustomersAPI
	clock   timeutil.Clock
	devAuth bool
}

var _ oapi.StrictServerInterface = (*api)(nil)

func (a *api) GetHealthLive(ctx context.Context, _ oapi.GetHealthLiveRequestObject) (oapi.GetHealthLiveResponseObject, error) {
	return oapi.GetHealthLive200JSONResponse{Body: oapi.HealthLive{Status: "live"}}, nil
}

// GetHealthReady: Slice 2 has no dependencies, so it is ready with no checks.
// Health is infrastructure status, deliberately not a business ResultEnvelope.
func (a *api) GetHealthReady(ctx context.Context, _ oapi.GetHealthReadyRequestObject) (oapi.GetHealthReadyResponseObject, error) {
	body := oapi.HealthReady{Status: "ready"}
	body.Checks = make([]struct {
		Name   string                       `json:"name"`
		Status oapi.HealthReadyChecksStatus `json:"status"`
	}, 0)
	return oapi.GetHealthReady200JSONResponse{Body: body}, nil
}

func (a *api) GetSession(ctx context.Context, _ oapi.GetSessionRequestObject) (oapi.GetSessionResponseObject, error) {
	outcome, err := httpx.Outcome(result.Success)
	if err != nil {
		return nil, err
	}
	data := oapi.SessionContext{
		Authenticated: false,
		AuthMode:      "none",
		DisplayZone:   displayZone,
		ServerTime:    a.clock.Now().Time(),
	}
	if a.devAuth {
		data.AuthMode = "dev_stub"
		if p, ok := authstub.FromContext(ctx); ok {
			hints := append([]string(nil), p.CapabilityHints...)
			data.Authenticated = true
			data.CapabilityHints = &hints
			data.Principal = &struct {
				DisplayName string                                `json:"display_name"`
				PrincipalId string                                `json:"principal_id"`
				Synthetic   oapi.SessionContextPrincipalSynthetic `json:"synthetic"`
			}{DisplayName: p.DisplayName, PrincipalId: p.ID, Synthetic: true}
		}
	}
	data.Features = &struct {
		CustomersIdentity bool `json:"customers_identity"`
		HrRead            bool `json:"hr_read"`
	}{CustomersIdentity: a.CustomersAPI.CustomersEnabled(ctx), HrRead: a.ReadAPI.HRReadEnabled(ctx)}
	return oapi.GetSession200JSONResponse{Body: oapi.SessionEnvelope{
		Outcome:       outcome,
		Reasons:       []oapi.ReasonCode{},
		CorrelationId: httpx.CorrelationID(ctx),
		Data:          data,
	}}, nil
}

// NewHandler builds the full handler chain without a Slice 4 read surface.
func NewHandler(cfg Config, clock timeutil.Clock, logger *slog.Logger) (http.Handler, error) {
	return NewHandlerWith(cfg, clock, logger, nil, nil)
}

// NewHandlerWith builds the full handler chain:
// Correlation → Recover → LimitBody → (dev auth stub) → query allowlist → routes.
// read and customers may be nil (their operations are then BLOCKED /
// FEATURE_DISABLED).
func NewHandlerWith(cfg Config, clock timeutil.Clock, logger *slog.Logger, read ReadAPI, customers CustomersAPI) (http.Handler, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if clock == nil || logger == nil {
		return nil, ErrInvalidConfig
	}
	if read == nil {
		read = unconfigured{}
	}
	if customers == nil {
		customers = unconfiguredCustomers{}
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", fallback)
	strict := oapi.NewStrictHandlerWithOptions(&api{ReadAPI: read, CustomersAPI: customers, clock: clock, devAuth: cfg.DevAuth}, nil, oapi.StrictHTTPServerOptions{
		RequestErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			httpx.WriteProblem(w, r, httpx.ProblemBadRequest)
		},
		ResponseErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			logger.Error("response error", "correlation_id", httpx.CorrelationID(r.Context()), "error", err)
			httpx.WriteProblem(w, r, httpx.ProblemInternal)
		},
	})
	var h http.Handler = oapi.HandlerWithOptions(strict, oapi.StdHTTPServerOptions{
		BaseRouter: mux,
		ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			httpx.WriteProblem(w, r, httpx.ProblemBadRequest)
		},
	})
	h = queryAllowlist(h)
	if cfg.DevAuth {
		stub, err := authstub.New(cfg.Environment)
		if err != nil {
			return nil, err
		}
		h = stub.Middleware(func(w http.ResponseWriter, r *http.Request, err error) {
			p := httpx.ProblemBadRequest
			p.Detail = "Unknown " + authstub.Header + " synthetic identity."
			httpx.WriteProblem(w, r, p)
		})(h)
	}
	h = httpx.LimitBody(cfg.MaxBodyBytes)(h)
	h = httpx.Recover(logger)(h)
	h = httpx.Correlation(h)
	return h, nil
}

// template returns the contract path template matching path, if any. When
// several match, the one with the most literal segments wins (deterministic).
func template(path string) (string, bool) {
	if _, ok := ContractPaths[path]; ok {
		return path, true
	}
	segs := strings.Split(path, "/")
	best, bestLiterals := "", -1
	for t := range ContractPaths {
		ts := strings.Split(t, "/")
		if len(ts) != len(segs) {
			continue
		}
		match, literals := true, 0
		for i := 0; i < len(ts) && match; i++ {
			switch {
			case ts[i] == segs[i]:
				literals++
			case strings.HasPrefix(ts[i], "{") && segs[i] != "":
			default:
				match = false
			}
		}
		if match && (literals > bestLiterals || literals == bestLiterals && t < best) {
			best, bestLiterals = t, literals
		}
	}
	return best, best != ""
}

// queryAllowlist rejects query parameters a Slice 4 operation does not
// declare, instead of silently ignoring them.
func queryAllowlist(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if t, ok := template(r.URL.Path); ok {
			if allowed, guarded := ContractQuery[t]; guarded {
				// A query the standard parser would partially drop (e.g. a ";")
				// is refused, never silently reduced to a default.
				q, err := url.ParseQuery(r.URL.RawQuery)
				if err != nil {
					p := httpx.ProblemBadRequest
					p.Detail = "Malformed query string."
					httpx.WriteProblem(w, r, p)
					return
				}
				for k := range q {
					if !slices.Contains(allowed, k) {
						p := httpx.ProblemBadRequest
						p.Detail = "Unknown query parameter " + strconv.Quote(k) + "."
						httpx.WriteProblem(w, r, p)
						return
					}
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

// fallback answers every request no contract operation matched.
func fallback(w http.ResponseWriter, r *http.Request) {
	if t, ok := template(r.URL.Path); ok {
		methods := ContractPaths[t]
		for _, m := range methods {
			w.Header().Add("Allow", m)
		}
		httpx.WriteProblem(w, r, httpx.ProblemMethodNotAllowed)
		return
	}
	httpx.WriteProblem(w, r, httpx.ProblemNotFound)
}

// NewServer returns an *http.Server with the configured timeouts and limits.
func NewServer(cfg Config, clock timeutil.Clock, logger *slog.Logger) (*http.Server, error) {
	return NewServerWith(cfg, clock, logger, nil, nil)
}

// NewServerWith is NewServer with the Slice 4 and Slice 5 read surfaces (either may be nil).
func NewServerWith(cfg Config, clock timeutil.Clock, logger *slog.Logger, read ReadAPI, customers CustomersAPI) (*http.Server, error) {
	h, err := NewHandlerWith(cfg, clock, logger, read, customers)
	if err != nil {
		return nil, err
	}
	return &http.Server{
		Addr:              cfg.Addr,
		Handler:           h,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		MaxHeaderBytes:    cfg.MaxHeaderBytes,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelWarn),
	}, nil
}

// Serve runs srv until ctx is cancelled, then shuts down gracefully within
// cfg.ShutdownTimeout. It returns nil after a clean shutdown.
func Serve(ctx context.Context, srv *http.Server, cfg Config, logger *slog.Logger) error {
	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()
	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
	}
	logger.Info("shutting down", "timeout", cfg.ShutdownTimeout.String())
	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cfg.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}
	if err := <-errCh; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
