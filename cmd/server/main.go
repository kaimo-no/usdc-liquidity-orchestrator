// Command server is a thin HTTP microservice wrapping the pure liquidity planner.
//
//	POST /v1/plan   — dry plan (execute defaults false; true fails closed)
//	GET  /healthz
//
// Non-custodial: never holds keys or product funds. Inventory is request-scoped only.
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	liqerr "github.com/kilian1103/usdc-liquidity-orchestrator/pkg/errors"
	"github.com/kilian1103/usdc-liquidity-orchestrator/pkg/liquidity"
	"github.com/kilian1103/usdc-liquidity-orchestrator/pkg/types"
)

func main() {
	addr := envOr("LISTEN_ADDR", ":8088")
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("POST /v1/plan", handlePlan)

	s := &http.Server{
		Addr:              addr,
		Handler:           logRequests(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("usdc-liquidity-orchestrator listening on %s", addr)
	log.Fatal(s.ListenAndServe())
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

	plan, err := liquidity.PlanLiquidity(required, inv, nil)
	if err != nil {
		writeCoded(w, err)
		return
	}

	wire := liquidity.PlanToWire(plan)
	// Force dry stamps on the HTTP surface (never claim funded).
	wire.DryRun = true
	wire.Executed = false
	wire.InventoryAsserted = true
	wire.InventoryUnverified = true

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

func writeCoded(w http.ResponseWriter, err error) {
	code := liqerr.CodeOf(err)
	if code == "" {
		code = liqerr.CodeInvalidQuery
	}
	status := http.StatusBadRequest
	writeErr(w, status, code, err.Error())
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

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		// Never log bodies (may contain wallet inventory).
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
	})
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
