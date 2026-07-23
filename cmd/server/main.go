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
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/kaimo-no/usdc-liquidity-orchestrator/internal/httpserver"
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
	if os.Getenv("ENABLE_TESTNET_EXECUTE") != "1" {
		return liquidity.UnconfiguredExecutor{}, nil
	}
	if !isLoopbackListen(listenAddr) {
		return nil, fmt.Errorf("ENABLE_TESTNET_EXECUTE requires loopback LISTEN_ADDR (127.0.0.1, ::1, or localhost)")
	}
	key := strings.TrimSpace(os.Getenv("AGENT_PRIVATE_KEY"))
	if key == "" {
		return nil, fmt.Errorf("AGENT_PRIVATE_KEY required when ENABLE_TESTNET_EXECUTE=1")
	}
	rpcs, err := loadRPCsFromEnv()
	if err != nil {
		return nil, err
	}
	if len(rpcs) == 0 {
		return nil, fmt.Errorf("RPC_URLS_JSON or RPC_URL_eip155_* required when ENABLE_TESTNET_EXECUTE=1")
	}
	ex, err := execonchain.NewDepositExecutor(execonchain.Config{
		PrivateKeyHex: key,
		RPCs:          rpcs,
	})
	if err != nil {
		return nil, err
	}
	log.Printf("testnet deposit execute enabled (loopback only)")
	return ex, nil
}

// loadRPCsFromEnv reads RPC_URLS_JSON and/or RPC_URL_eip155_N env vars.
// Never logs values.
func loadRPCsFromEnv() (map[string]string, error) {
	out := map[string]string{}
	if raw := strings.TrimSpace(os.Getenv("RPC_URLS_JSON")); raw != "" {
		var m map[string]string
		if err := json.Unmarshal([]byte(raw), &m); err != nil {
			return nil, fmt.Errorf("RPC_URLS_JSON: invalid JSON object")
		}
		for k, v := range m {
			k, v = strings.TrimSpace(k), strings.TrimSpace(v)
			if k == "" || v == "" {
				return nil, fmt.Errorf("RPC_URLS_JSON: empty key or value refused")
			}
			out[k] = v
		}
	}
	const prefix = "RPC_URL_"
	for _, e := range os.Environ() {
		// KEY=VALUE
		eq := strings.IndexByte(e, '=')
		if eq <= 0 {
			continue
		}
		name, val := e[:eq], e[eq+1:]
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		// RPC_URL_eip155_84532 → eip155:84532
		rest := strings.TrimPrefix(name, prefix)
		if rest == "" {
			continue
		}
		caip := strings.ReplaceAll(rest, "_", ":")
		// Prefer first underscore → colon only for eip155_N form.
		// eip155_84532 → eip155:84532 (single replacement of first _)
		if strings.HasPrefix(rest, "eip155_") {
			caip = "eip155:" + strings.TrimPrefix(rest, "eip155_")
		}
		val = strings.TrimSpace(val)
		if val == "" {
			return nil, fmt.Errorf("%s: empty value refused", name)
		}
		out[caip] = val
	}
	return out, nil
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
