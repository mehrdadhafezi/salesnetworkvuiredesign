package apiserver

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

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
	"/health/live":    {http.MethodGet},
	"/health/ready":   {http.MethodGet},
	"/api/v1/session": {http.MethodGet},
}

// displayZone is the presentation default zone; never storage truth.
const displayZone = oapi.SessionContextDisplayZone("Asia/Tehran")

type api struct {
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
	return oapi.GetSession200JSONResponse{Body: oapi.SessionEnvelope{
		Outcome:       outcome,
		Reasons:       []oapi.ReasonCode{},
		CorrelationId: httpx.CorrelationID(ctx),
		Data:          data,
	}}, nil
}

// NewHandler builds the full handler chain:
// Correlation → Recover → LimitBody → (dev auth stub) → routes.
func NewHandler(cfg Config, clock timeutil.Clock, logger *slog.Logger) (http.Handler, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if clock == nil || logger == nil {
		return nil, ErrInvalidConfig
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", fallback)
	strict := oapi.NewStrictHandlerWithOptions(&api{clock: clock, devAuth: cfg.DevAuth}, nil, oapi.StrictHTTPServerOptions{
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

// fallback answers every request no contract operation matched.
func fallback(w http.ResponseWriter, r *http.Request) {
	if methods, ok := ContractPaths[r.URL.Path]; ok {
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
	h, err := NewHandler(cfg, clock, logger)
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
