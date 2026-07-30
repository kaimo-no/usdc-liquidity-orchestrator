// Package httpserver is the thin HTTP surface over pkg/liquidity via planio.
// cmd/server wires ListenAndServe; black-box tests use NewMux / NewMuxWithOptions.
package httpserver

import (
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"time"

	"github.com/kaimo-no/usdc-liquidity-orchestrator/internal/planio"
	liqerr "github.com/kaimo-no/usdc-liquidity-orchestrator/pkg/errors"
	"github.com/kaimo-no/usdc-liquidity-orchestrator/pkg/liquidity"
	"github.com/kaimo-no/usdc-liquidity-orchestrator/pkg/types"
)

//go:embed static/*
var staticEmbed embed.FS

// MuxOptions configures optional execute wiring. Zero value is plan-only default.
type MuxOptions struct {
	// Executor runs plans when request execute=true. Nil → UnconfiguredExecutor.
	Executor liquidity.Executor
}

// NewMux returns the service routes (plan-only default: UnconfiguredExecutor).
// Serves a small plan-only UI at GET / (embedded static/).
func NewMux() http.Handler {
	return NewMuxWithOptions(MuxOptions{})
}

// NewMuxWithOptions returns routes with an optional live Executor.
func NewMuxWithOptions(opts MuxOptions) http.Handler {
	ex := opts.Executor
	if ex == nil {
		ex = liquidity.UnconfiguredExecutor{}
	}
	s := &server{ex: ex}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("GET /v1/chains", handleChains)
	mux.HandleFunc("POST /v1/plan", s.handlePlan)
	mux.HandleFunc("POST /v1/consolidate", s.handleConsolidate)
	mux.HandleFunc("POST /v1/payment-funding", s.handlePaymentFunding)

	staticRoot, err := fs.Sub(staticEmbed, "static")
	if err != nil {
		panic("httpserver: embed static/: " + err.Error())
	}
	mux.Handle("GET /", http.FileServer(http.FS(staticRoot)))
	return mux
}

type server struct {
	ex liquidity.Executor
}

// LogRequests wraps next without logging request bodies (wallet inventory).
func LogRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
	})
}

func handleChains(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, planio.ListChains())
}

func (s *server) handlePlan(w http.ResponseWriter, r *http.Request) {
	var req types.PlanRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeErr(w, liqerr.CodeInvalidQuery, "invalid JSON body")
		return
	}
	resp, outcome := planio.RunPlan(r.Context(), s.ex, req)
	writeJSON(w, planio.HTTPStatus(outcome), resp)
}

func (s *server) handleConsolidate(w http.ResponseWriter, r *http.Request) {
	var req types.ConsolidateRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeErr(w, liqerr.CodeInvalidQuery, "invalid JSON body")
		return
	}
	resp, outcome := planio.RunConsolidate(r.Context(), s.ex, req)
	writeJSON(w, planio.HTTPStatus(outcome), resp)
}

func (s *server) handlePaymentFunding(w http.ResponseWriter, r *http.Request) {
	var req types.PaymentFundingRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeErr(w, liqerr.CodeInvalidQuery, "invalid JSON body")
		return
	}
	resp, outcome := planio.RunPaymentFunding(r.Context(), s.ex, req)
	writeJSON(w, planio.HTTPStatus(outcome), resp)
}

func writeErr(w http.ResponseWriter, code, msg string) {
	writeJSON(w, http.StatusBadRequest, types.PlanResponse{
		Error: &types.APIError{Code: code, Message: msg},
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}
