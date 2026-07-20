// Package httpserver is the thin HTTP surface over pkg/liquidity.
// cmd/server wires ListenAndServe; black-box tests use NewMux.
package httpserver

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	liqerr "github.com/kaimo-no/usdc-liquidity-orchestrator/pkg/errors"
	"github.com/kaimo-no/usdc-liquidity-orchestrator/pkg/liquidity"
	"github.com/kaimo-no/usdc-liquidity-orchestrator/pkg/types"
)

// NewMux returns the service routes (no logging wrapper).
func NewMux() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("GET /v1/chains", handleChains)
	mux.HandleFunc("POST /v1/plan", handlePlan)
	mux.HandleFunc("POST /v1/consolidate", handleConsolidate)
	return mux
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

func handlePlan(w http.ResponseWriter, r *http.Request) {
	var req types.PlanRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, liqerr.CodeInvalidQuery, "invalid JSON body")
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

	wire := forceDryStamps(liquidity.PlanToWire(plan))

	if req.Execute {
		ex := liquidity.UnconfiguredExecutor{}
		if _, err := ex.Execute(r.Context(), plan); err != nil {
			// Still return the dry plan + error so agents can inspect steps.
			writeJSON(w, http.StatusBadRequest, types.PlanResponse{
				Plan:  wire,
				Error: &types.APIError{Code: liqerr.CodeOf(err), Message: err.Error()},
			})
			return
		}
	}

	writeJSON(w, http.StatusOK, types.PlanResponse{Plan: wire})
}

func handleConsolidate(w http.ResponseWriter, r *http.Request) {
	var req types.ConsolidateRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, liqerr.CodeInvalidQuery, "invalid JSON body")
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

	wire := forceDryStamps(liquidity.PlanToWire(plan))

	if req.Execute {
		ex := liquidity.UnconfiguredExecutor{}
		if _, err := ex.Execute(r.Context(), plan); err != nil {
			writeJSON(w, http.StatusBadRequest, types.PlanResponse{
				Plan:  wire,
				Error: &types.APIError{Code: liqerr.CodeOf(err), Message: err.Error()},
			})
			return
		}
	}

	writeJSON(w, http.StatusOK, types.PlanResponse{Plan: wire})
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
	writeErr(w, http.StatusBadRequest, code, err.Error())
}

func writeErr(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, types.PlanResponse{
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
