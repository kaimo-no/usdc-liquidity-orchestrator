// Package liqcli implements the usdc-liq dual-surface CLI (plan parity with HTTP).
// Never logs keys, agent addresses, balances, prepare calldata, or RPC URLs.
package liqcli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/kaimo-no/usdc-liquidity-orchestrator/internal/demorun"
	"github.com/kaimo-no/usdc-liquidity-orchestrator/internal/execenv"
	"github.com/kaimo-no/usdc-liquidity-orchestrator/internal/inventory"
	"github.com/kaimo-no/usdc-liquidity-orchestrator/internal/planio"
	"github.com/kaimo-no/usdc-liquidity-orchestrator/internal/rpcenv"
	"github.com/kaimo-no/usdc-liquidity-orchestrator/pkg/liquidity"
	"github.com/kaimo-no/usdc-liquidity-orchestrator/pkg/types"
)

// Version is reported by the version subcommand.
const Version = "0.1.0"

const maxBody = 1 << 20 // 1 MiB

// Main runs the CLI with the given argv (without program name), I/O streams.
// Exit codes: 0 success, 1 plan/execute failure, 2 usage.
func Main(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) < 1 {
		printUsage(stderr)
		return 2
	}
	cmd := args[0]
	rest := args[1:]
	switch cmd {
	case "plan":
		return runPlanCmd(rest, stdin, stdout, stderr)
	case "consolidate":
		return runConsolidateCmd(rest, stdin, stdout, stderr)
	case "payment-funding":
		return runPaymentFundingCmd(rest, stdin, stdout, stderr)
	case "chains":
		return runChainsCmd(rest, stdout, stderr)
	case "inventory":
		return runInventoryCmd(rest, stdout, stderr)
	case "demo":
		return demorun.Run(stdout, stderr)
	case "version":
		fmt.Fprintln(stdout, Version)
		return 0
	case "help", "-h", "--help":
		printUsage(stderr)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n", cmd)
		printUsage(stderr)
		return 2
	}
}

func printUsage(w io.Writer) {
	fmt.Fprint(w, `usdc-liq — non-custodial multi-chain USDC liquidity CLI

Usage:
  usdc-liq <command> [flags]

Commands (HTTP parity):
  plan              shortfall-only rebalance plan (POST /v1/plan body)
  consolidate       Gateway deposit plan (POST /v1/consolidate body)
  payment-funding   scenario full-funding plan (POST /v1/payment-funding body)
  chains            registered corridors (GET /v1/chains)

CLI-only:
  inventory         load live native + optional Gateway balances (needs agent + RPCs)
  demo              worked scenario + shortfall + consolidate examples
  version           print version

Shared flags (plan | consolidate | payment-funding):
  -f file     request JSON file (default "-" = stdin)
  --execute   request execute=true (requires dual-gate env when live)

Exit codes: 0 success, 1 plan/execute failure, 2 usage.
JSON always on stdout; notes on stderr (no secrets/balances/RPC).
`)
}

func runPlanCmd(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("plan", flag.ContinueOnError)
	fs.SetOutput(stderr)
	file, execute := addSharedFlags(fs)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	raw, err := readInput(*file, stdin)
	if err != nil {
		fmt.Fprintf(stderr, "read input: %v\n", err)
		return 1
	}
	var req types.PlanRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		fmt.Fprintf(stderr, "invalid JSON body\n")
		return 1
	}
	if *execute {
		req.Execute = true
	}
	ex, code := resolveExecutor(req.Execute, stdout, stderr)
	if code != 0 {
		return code
	}
	resp, outcome := planio.RunPlan(context.Background(), ex, req)
	return writePlanResp(stdout, stderr, resp, outcome)
}

func runConsolidateCmd(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("consolidate", flag.ContinueOnError)
	fs.SetOutput(stderr)
	file, execute := addSharedFlags(fs)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	raw, err := readInput(*file, stdin)
	if err != nil {
		fmt.Fprintf(stderr, "read input: %v\n", err)
		return 1
	}
	var req types.ConsolidateRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		fmt.Fprintf(stderr, "invalid JSON body\n")
		return 1
	}
	if *execute {
		req.Execute = true
	}
	ex, code := resolveExecutor(req.Execute, stdout, stderr)
	if code != 0 {
		return code
	}
	resp, outcome := planio.RunConsolidate(context.Background(), ex, req)
	return writePlanResp(stdout, stderr, resp, outcome)
}

func runPaymentFundingCmd(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("payment-funding", flag.ContinueOnError)
	fs.SetOutput(stderr)
	file, execute := addSharedFlags(fs)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	raw, err := readInput(*file, stdin)
	if err != nil {
		fmt.Fprintf(stderr, "read input: %v\n", err)
		return 1
	}
	var req types.PaymentFundingRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		fmt.Fprintf(stderr, "invalid JSON body\n")
		return 1
	}
	if *execute {
		req.Execute = true
	}
	ex, code := resolveExecutor(req.Execute, stdout, stderr)
	if code != 0 {
		return code
	}
	resp, outcome := planio.RunPaymentFunding(context.Background(), ex, req)
	return writePlanResp(stdout, stderr, resp, outcome)
}

func runChainsCmd(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("chains", flag.ContinueOnError)
	fs.SetOutput(stderr)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	return writeJSON(stdout, planio.ListChains())
}

func runInventoryCmd(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("inventory", flag.ContinueOnError)
	fs.SetOutput(stderr)
	agent := fs.String("agent", "", "agent wallet address (or AGENT_ADDRESS env)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	cfg, err := ValidateInventoryArgs(*agent, os.Getenv("AGENT_ADDRESS"))
	if err != nil {
		fmt.Fprintf(stderr, "%s\n", err.Error())
		return 1
	}
	rpcs, err := rpcenv.LoadEVMTestnetExecuteRPCs()
	if err != nil {
		fmt.Fprintf(stderr, "inventory: RPC env invalid\n")
		return 1
	}
	if err := ValidateInventoryRPCs(rpcs); err != nil {
		fmt.Fprintf(stderr, "%s\n", err.Error())
		return 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	gw := strings.TrimSpace(os.Getenv("GATEWAY_API_BASE"))
	inv, err := inventory.Load(ctx, inventory.Config{
		AgentAddress: cfg.AgentAddress,
		RPCs:         rpcs,
		GatewayAPI:   gw,
	})
	if err != nil {
		// Sanitized fixed message only — no RPC / balance details.
		fmt.Fprintf(stderr, "inventory load failed\n")
		return 1
	}
	wire := liquidity.InventoryToWire(inv)
	return writeJSON(stdout, wire)
}

// InventoryConfig is validated inventory CLI input (no live fields).
type InventoryConfig struct {
	AgentAddress string
}

// ValidateInventoryArgs checks agent address from flag or env (unit-testable, no network).
func ValidateInventoryArgs(flagAgent, envAgent string) (InventoryConfig, error) {
	agent := strings.TrimSpace(flagAgent)
	if agent == "" {
		agent = strings.TrimSpace(envAgent)
	}
	if agent == "" {
		return InventoryConfig{}, fmt.Errorf("inventory: agent address required (-agent or AGENT_ADDRESS)")
	}
	// Hex address shape check without importing eth packages in callers' tests.
	a := strings.TrimPrefix(strings.TrimPrefix(agent, "0x"), "0X")
	if len(a) != 40 {
		return InventoryConfig{}, fmt.Errorf("inventory: agent_address required (valid EVM hex)")
	}
	for _, c := range a {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
			return InventoryConfig{}, fmt.Errorf("inventory: agent_address required (valid EVM hex)")
		}
	}
	return InventoryConfig{AgentAddress: "0x" + a}, nil
}

// ValidateInventoryRPCs checks that at least one RPC is configured (no network).
func ValidateInventoryRPCs(rpcs map[string]string) error {
	if len(rpcs) == 0 {
		return fmt.Errorf("inventory: RPC map required (set RPC_URL_BASE_SEPOLIA / ARBITRUM_SEPOLIA / ARC_TESTNET or RPC_URLS_JSON)")
	}
	return nil
}

func addSharedFlags(fs *flag.FlagSet) (file *string, execute *bool) {
	file = fs.String("f", "-", "request JSON file (\"-\" = stdin)")
	execute = fs.Bool("execute", false, "execute plan when dual-gate env is configured")
	return file, execute
}

// resolveExecutor: dry always UnconfiguredExecutor; execute uses BuildExecutor (no loopback).
// On configure failure returns nil executor and exit 1 after writing sanitized JSON to stdout.
func resolveExecutor(execute bool, stdout, stderr io.Writer) (liquidity.Executor, int) {
	if !execute {
		return liquidity.UnconfiguredExecutor{}, 0
	}
	ex, err := execenv.BuildExecutor(execenv.Options{})
	if err != nil {
		api := planio.SanitizeAPIError(err)
		if api == nil {
			api = &types.APIError{Code: "liquidity_rail_unavailable", Message: "execute failed"}
		}
		fmt.Fprintln(stderr, "# execute configure failed")
		_ = writeJSON(stdout, types.PlanResponse{Error: api})
		return nil, 1
	}
	return ex, 0
}

func writePlanResp(stdout, stderr io.Writer, resp types.PlanResponse, outcome planio.StampOutcome) int {
	if err := writeJSON(stdout, resp); err != 0 {
		return err
	}
	if outcome != planio.StampOK {
		if resp.Error != nil && resp.Error.Message != "" {
			fmt.Fprintf(stderr, "# %s: %s\n", resp.Error.Code, resp.Error.Message)
		} else {
			fmt.Fprintln(stderr, "# plan failed")
		}
	}
	return planio.ExitCode(outcome)
}

func writeJSON(w io.Writer, v any) int {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return 1
	}
	return 0
}

func readInput(path string, stdin io.Reader) ([]byte, error) {
	var r io.Reader
	var closer io.Closer
	if path == "" || path == "-" {
		r = stdin
	} else {
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		closer = f
		r = f
	}
	if closer != nil {
		defer closer.Close()
	}
	limited := io.LimitReader(r, maxBody+1)
	b, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if len(b) > maxBody {
		return nil, fmt.Errorf("body exceeds 1 MiB limit")
	}
	return b, nil
}
