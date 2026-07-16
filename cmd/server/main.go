// Command server is a thin HTTP microservice wrapping the pure liquidity planner.
//
//	POST /v1/plan   — dry plan (execute defaults false; true fails closed)
//	GET  /v1/chains — registered corridors (discovery for agents)
//	GET  /healthz
//
// Non-custodial: never holds keys or product funds. Inventory is request-scoped only.
package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/kaimo-no/usdc-liquidity-orchestrator/internal/httpserver"
)

func main() {
	addr := envOr("LISTEN_ADDR", ":8088")
	s := &http.Server{
		Addr:              addr,
		Handler:           httpserver.LogRequests(httpserver.NewMux()),
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
