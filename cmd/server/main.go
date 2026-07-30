// Command server is a thin HTTP microservice wrapping the pure liquidity planner.
//
//	POST /v1/plan         — dry plan (execute defaults false; true fails closed unless testnet execute enabled)
//	POST /v1/consolidate  — Gateway deposit plan (+ optional testnet execute)
//	GET  /v1/chains       — registered corridors (discovery for agents)
//	GET  /healthz
//
// Non-custodial: never holds product funds. Optional testnet Gateway execute requires
// ENABLE_TESTNET_EXECUTE=1 + AGENT_PRIVATE_KEY + RPCs and loopback LISTEN_ADDR only.
// Never log keys, agent addresses, balances, calldata, or RPC URLs.
package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/kaimo-no/usdc-liquidity-orchestrator/internal/envfile"
	"github.com/kaimo-no/usdc-liquidity-orchestrator/internal/execenv"
	"github.com/kaimo-no/usdc-liquidity-orchestrator/internal/httpserver"
)

func main() {
	if err := envfile.Load(".env"); err != nil {
		log.Fatal(err)
	}

	addr := envOr("LISTEN_ADDR", ":8088")
	ex, err := execenv.BuildExecutor(execenv.Options{RequireLoopbackListen: addr})
	if err != nil {
		log.Fatal(err)
	}
	if os.Getenv("ENABLE_TESTNET_EXECUTE") == "1" {
		log.Printf("testnet gateway execute enabled (consolidate + deposit_withdraw; loopback only)")
	}
	s := &http.Server{
		Addr:              addr,
		Handler:           httpserver.LogRequests(httpserver.NewMuxWithOptions(httpserver.MuxOptions{Executor: ex})),
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("usdc-liquidity-orchestrator listening on %s", addr)
	log.Fatal(s.ListenAndServe())
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
