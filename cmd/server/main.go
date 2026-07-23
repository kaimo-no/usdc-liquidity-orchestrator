// Command server is a thin HTTP microservice wrapping the pure liquidity planner.
//
//	POST /v1/plan         — dry plan (execute defaults false; true fails closed unless testnet execute enabled)
//	POST /v1/consolidate  — Gateway deposit plan (+ optional testnet execute)
//	GET  /v1/chains       — registered corridors (discovery for agents)
//	GET  /healthz
//
// Non-custodial: never holds product funds. Optional testnet deposit execute requires
// ENABLE_TESTNET_EXECUTE=1 + AGENT_PRIVATE_KEY + RPCs and loopback LISTEN_ADDR only.
// Never log keys, agent addresses, balances, calldata, or RPC URLs.
package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/kaimo-no/usdc-liquidity-orchestrator/internal/httpserver"
	"github.com/kaimo-no/usdc-liquidity-orchestrator/internal/rpcenv"
	"github.com/kaimo-no/usdc-liquidity-orchestrator/pkg/execonchain"
	"github.com/kaimo-no/usdc-liquidity-orchestrator/pkg/liquidity"
)

func main() {
	addr := envOr("LISTEN_ADDR", ":8088")
	ex, err := buildExecutor(addr)
	if err != nil {
		log.Fatal(err)
	}
	s := &http.Server{
		Addr:              addr,
		Handler:           httpserver.LogRequests(httpserver.NewMuxWithOptions(httpserver.MuxOptions{Executor: ex})),
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("usdc-liquidity-orchestrator listening on %s", addr)
	log.Fatal(s.ListenAndServe())
}

// buildExecutor returns UnconfiguredExecutor unless dual-gated testnet execute is on.
func buildExecutor(listenAddr string) (liquidity.Executor, error) {
	guard, err := guardFromEnv()
	if err != nil {
		return nil, err
	}
	if os.Getenv("ENABLE_TESTNET_EXECUTE") != "1" {
		return liquidity.UnconfiguredExecutor{Guard: guard}, nil
	}
	if !isLoopbackListen(listenAddr) {
		return nil, fmt.Errorf("ENABLE_TESTNET_EXECUTE requires loopback LISTEN_ADDR (127.0.0.1, ::1, or localhost)")
	}
	key := strings.TrimSpace(os.Getenv("AGENT_PRIVATE_KEY"))
	if key == "" {
		return nil, fmt.Errorf("AGENT_PRIVATE_KEY required when ENABLE_TESTNET_EXECUTE=1")
	}
	rpcs, err := rpcenv.LoadEVMTestnetExecuteRPCs()
	if err != nil {
		return nil, err
	}
	if len(rpcs) == 0 {
		return nil, fmt.Errorf("testnet EVM RPC required when ENABLE_TESTNET_EXECUTE=1 (RPC_URL_BASE_SEPOLIA / ARBITRUM_SEPOLIA / ARC_TESTNET, RPC_URL_eip155_*, or RPC_URLS_JSON)")
	}
	ex, err := execonchain.NewDepositExecutor(execonchain.Config{
		PrivateKeyHex: key,
		RPCs:          rpcs,
		Guard:         guard,
	})
	if err != nil {
		return nil, err
	}
	log.Printf("testnet deposit execute enabled (loopback only)")
	return ex, nil
}

// guardFromEnv builds a Guard with optional MAX_AMOUNT_ATOMIC cap.
func guardFromEnv() (*liquidity.Guard, error) {
	g := &liquidity.Guard{}
	if raw := strings.TrimSpace(os.Getenv("MAX_AMOUNT_ATOMIC")); raw != "" {
		d, err := decimal.NewFromString(raw)
		if err != nil || !d.IsPositive() {
			return nil, fmt.Errorf("MAX_AMOUNT_ATOMIC: must be a positive decimal integer (atomic units)")
		}
		g.MaxAmountAtomic = d
	}
	return g, nil
}

// isLoopbackListen accepts host parts that bind only to loopback.
// Bare ":port" is NOT loopback (binds all interfaces) — refuse for execute.
func isLoopbackListen(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		// Try as host only or bracketed IPv6.
		host = strings.Trim(addr, "[]")
		if i := strings.LastIndex(addr, ":"); i > 0 && !strings.Contains(addr, "]") {
			host = addr[:i]
		}
	}
	host = strings.TrimSpace(host)
	if host == "" {
		return false
	}
	switch strings.ToLower(host) {
	case "127.0.0.1", "localhost", "::1":
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
