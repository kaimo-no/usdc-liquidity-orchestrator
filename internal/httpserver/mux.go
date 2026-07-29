// Package httpserver is the thin HTTP surface over pkg/liquidity.
// cmd/server wires ListenAndServe; black-box tests use NewMux / NewMuxWithOptions.
package httpserver

import (
	"embed"
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/shopspring/decimal"

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
		// Embed layout bug — fail closed at process start rather than silent 404 UI.
		panic("httpserver: embed static/: " + err.Error())
	}
	// GET / and static assets; more specific /v1/* and /healthz win on ServeMux.
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
	reg := liquidity.ListChains()
	out := make([]types.ChainInfo, 0, len(reg))
	for _, c := range reg {
		wallet, _ := liquidity.GatewayWalletAddress(c.CAIP2)
		out = append(out, types.ChainInfo{
			CAIP2: c.CAIP2, Name: c.Name, GatewayDomain: c.GatewayDomain,
			USDC: c.USDC, GatewayOK: c.GatewayOK, CCTPOK: c.CCTPOK,
			Testnet: c.Testnet, GatewayWallet: wallet,
		})
	}
	writeJSON(w, http.StatusOK, types.ChainsResponse{Chains: out})
}

func (s *server) handlePlan(w http.ResponseWriter, r *http.Request) {
	var req types.PlanRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeErr(w, liqerr.CodeInvalidQuery, "invalid JSON body")
		return
	}

	required, err := liquidity.RequiredFromWire(&req.Required, req.AmountOverride)
	if err != nil {
		writeCoded(w, err)
		return
	}
	inv, err := liquidity.InventoryFromWire(req.Inventory)
	if err != nil {
		writeCoded(w, err)
		return
	}

	orch := liquidity.OrchestrationFromWire(req.Orchestration)
	fee := liquidity.FeeConfigFromWire(req.FeeBps, req.FeeRecipient)

	plan, err := liquidity.PlanOrchestration(required, inv, orch, fee, nil)
	if err != nil {
		writeCoded(w, err)
		return
	}

	s.stampAndWrite(w, r, plan, req.Execute)
}

func (s *server) handleConsolidate(w http.ResponseWriter, r *http.Request) {
	var req types.ConsolidateRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeErr(w, liqerr.CodeInvalidQuery, "invalid JSON body")
		return
	}

	inv, err := liquidity.InventoryFromWire(req.Inventory)
	if err != nil {
		writeCoded(w, err)
		return
	}
	orch := liquidity.OrchestrationFromWire(req.Orchestration)

	plan, err := liquidity.PlanConsolidate(inv, orch, nil)
	if err != nil {
		writeCoded(w, err)
		return
	}

	s.stampAndWrite(w, r, plan, req.Execute)
}

// handlePaymentFunding plans scenario full-funding (hard-coded multi-source deposits + withdraw).
// Unlike /v1/plan (shortfall-only), this path moves explicit source amounts then full payment_real.
func (s *server) handlePaymentFunding(w http.ResponseWriter, r *http.Request) {
	var req types.PaymentFundingRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeErr(w, liqerr.CodeInvalidQuery, "invalid JSON body")
		return
	}

	required, err := liquidity.RequiredFromWire(&req.Required, "")
	if err != nil {
		writeCoded(w, err)
		return
	}
	// Optional logical / scale stamps from wire required envelope.
	if log := strings.TrimSpace(req.Required.AmountLogicalAtomic); log != "" {
		// RequiredFromWire already set AmountAtomic (real). Attach logical for PlanToWire.
		if d, err := decimal.NewFromString(log); err == nil && d.IsPositive() && d.Equal(d.Truncate(0)) {
			required.AmountLogicalAtomic = d.Truncate(0)
		}
	}
	if req.Required.ScaleFactor > 0 {
		required.ScaleFactor = req.Required.ScaleFactor
	}

	inv, err := liquidity.InventoryFromWire(req.Inventory)
	if err != nil {
		writeCoded(w, err)
		return
	}
	sources, err := liquidity.FundingSourcesFromWire(req.Sources)
	if err != nil {
		writeCoded(w, err)
		return
	}

	plan, err := liquidity.PlanPaymentFunding(required, inv, sources, nil)
	if err != nil {
		writeCoded(w, err)
		return
	}

	s.stampAndWrite(w, r, plan, req.Execute)
}

func (s *server) stampAndWrite(w http.ResponseWriter, r *http.Request, plan liquidity.Plan, execute bool) {
	wire := liquidity.PlanToWire(plan)
	var receipt liquidity.Receipt
	var execErr error
	if execute {
		receipt, execErr = s.ex.Execute(r.Context(), plan)
	}
	resp, status := stampPlanResponse(wire, execute, receipt, execErr)
	writeJSON(w, status, resp)
}

// stampPlanResponse applies dry/execute stamps and receipt according to design:
//
//	execute=false → forceDry, no receipt, 200
//	success all → dry_run=false executed=true + receipt hashes, 200
//	partial (hashes+err) → dry_run=false executed=false + receipt + error, 400
//	fail zero hashes → forceDry + error, 400
func stampPlanResponse(wire types.Plan, execute bool, receipt liquidity.Receipt, execErr error) (types.PlanResponse, int) {
	if !execute {
		return types.PlanResponse{Plan: forceDryStamps(wire)}, http.StatusOK
	}

	hashes := append([]string(nil), receipt.TxHashes...)
	apiErr := sanitizeAPIError(execErr)

	if execErr == nil {
		wire.DryRun = false
		wire.Executed = true
		wire.InventoryAsserted = true
		wire.InventoryUnverified = true
		resp := types.PlanResponse{Plan: wire}
		if len(hashes) > 0 {
			resp.Receipt = &types.ExecuteReceipt{TxHashes: hashes}
		}
		return resp, http.StatusOK
	}

	if len(hashes) > 0 {
		// Partial: some txs landed; never claim full success.
		wire.DryRun = false
		wire.Executed = false
		wire.InventoryAsserted = true
		wire.InventoryUnverified = true
		return types.PlanResponse{
			Plan:    wire,
			Receipt: &types.ExecuteReceipt{TxHashes: hashes},
			Error:   apiErr,
		}, http.StatusBadRequest
	}

	// Zero hashes: force dry stamps.
	return types.PlanResponse{
		Plan:  forceDryStamps(wire),
		Error: apiErr,
	}, http.StatusBadRequest
}

// sanitizeAPIError maps execute errors to stable code + fixed Message (no raw RPC).
func sanitizeAPIError(err error) *types.APIError {
	if err == nil {
		return nil
	}
	var e *liqerr.Error
	if errors.As(err, &e) && e != nil {
		code := e.Code
		if code == "" {
			code = liqerr.CodeLiquidityRailUnavailable
		}
		msg := e.Message
		if msg == "" {
			msg = "execute failed"
		}
		return &types.APIError{Code: code, Message: msg}
	}
	code := liqerr.CodeOf(err)
	if code == "" {
		code = liqerr.CodeLiquidityRailUnavailable
	}
	return &types.APIError{Code: code, Message: "execute failed"}
}

func forceDryStamps(wire types.Plan) types.Plan {
	wire.DryRun = true
	wire.Executed = false
	wire.InventoryAsserted = true
	wire.InventoryUnverified = true
	return wire
}

func writeCoded(w http.ResponseWriter, err error) {
	code := liqerr.CodeOf(err)
	if code == "" {
		code = liqerr.CodeInvalidQuery
	}
	msg := err.Error()
	var e *liqerr.Error
	if errors.As(err, &e) && e != nil && e.Message != "" {
		msg = e.Message
	}
	writeErr(w, code, msg)
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
