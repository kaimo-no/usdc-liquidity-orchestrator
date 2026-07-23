// Command demo runs hackathon worked examples:
// 1) shortfall Gateway withdraw to agent_self on Arc Testnet
// 2) multi-chain native consolidate into Circle Gateway (unsigned prepare_calls)
// 3) optional live testnet consolidate execute when ENABLE_TESTNET_EXECUTE=1 + key + RPCs
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/kaimo-no/usdc-liquidity-orchestrator/pkg/execonchain"
	"github.com/kaimo-no/usdc-liquidity-orchestrator/pkg/liquidity"
)

func main() {
	demoShortfallPlan()
	fmt.Fprint(os.Stderr, "\n--- consolidate ---\n\n")
	demoConsolidate()
	if os.Getenv("ENABLE_TESTNET_EXECUTE") == "1" {
		fmt.Fprint(os.Stderr, "\n--- live testnet consolidate execute ---\n\n")
		demoLiveConsolidateExecute()
	}
}

func demoShortfallPlan() {
	const (
		arc     = "eip155:5042002"
		arcUSDC = "0x3600000000000000000000000000000000000000"
		payTo   = "0xMerchantOnArc000000000000000000000001"
		agent   = "0xAgentSelf000000000000000000000000000001"
		feeTo   = "0xKaimoFee000000000000000000000000000001"
		need    = "42000000" // 42 USDC (6 decimals)
	)

	req := liquidity.Required{
		Protocol:     "x402",
		ChainCAIP2:   arc,
		Asset:        arcUSDC,
		PayTo:        payTo,
		AmountAtomic: decimal.RequireFromString(need),
		AmountSource: liquidity.AmountSourceProbe,
	}
	inv := liquidity.Inventory{
		AgentAddress: agent,
		Balances: []liquidity.Balance{
			{Asset: "USDC", AmountAtomic: decimal.RequireFromString("100000000"), Location: liquidity.LocationCircleGateway},
			{ChainCAIP2: arc, Asset: arcUSDC, AmountAtomic: decimal.RequireFromString("20000000"), Location: liquidity.LocationNative},
		},
	}
	orch := &liquidity.Orchestration{
		TargetChainCAIP2: arc,
		PreferRail:       liquidity.PreferRailCircleGateway,
	}
	fee := &liquidity.FeeConfig{
		Bps:       25,
		Recipient: feeTo,
		SettleVia: liquidity.SettleViaX402,
	}

	plan, err := liquidity.PlanOrchestration(req, inv, orch, fee, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "plan error: %v\n", err)
		os.Exit(1)
	}
	wire := liquidity.PlanToWire(plan)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(wire)

	fmt.Fprintf(os.Stderr, "\n# action=%s dry_run=%v executed=%v\n", plan.Action, plan.DryRun, plan.Executed)
	fmt.Fprintf(os.Stderr, "# target=Arc Testnet (%s) source=circle_gateway shortfall=22 USDC → agent_self\n", arc)
	fmt.Fprintf(os.Stderr, "# recipients on fund rails are always agent_self (%s), never merchant pay_to\n", agent)
	if plan.Fee != nil {
		fmt.Fprintf(os.Stderr, "# kaimo fee %s bps amount_atomic=%s settle_via=%s → %s\n",
			fmt.Sprint(plan.Fee.Bps), plan.Fee.AmountAtomic.String(), plan.Fee.SettleVia, plan.Fee.Recipient)
	}
}

func demoConsolidate() {
	const (
		arc         = "eip155:5042002"
		arcUSDC     = "0x3600000000000000000000000000000000000000"
		baseSep     = "eip155:84532"
		baseSepUSDC = "0x036CbD53842c5426634e7929541eC2318f3dCF7e"
		agent       = "0xAgentSelf000000000000000000000000000001"
	)
	inv := liquidity.Inventory{
		AgentAddress: agent,
		Balances: []liquidity.Balance{
			{ChainCAIP2: baseSep, Asset: baseSepUSDC, AmountAtomic: decimal.RequireFromString("3000000"), Location: liquidity.LocationNative},
			{ChainCAIP2: arc, Asset: arcUSDC, AmountAtomic: decimal.RequireFromString("1000000"), Location: liquidity.LocationNative},
		},
	}
	plan, err := liquidity.PlanConsolidate(inv, nil, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "consolidate error: %v\n", err)
		os.Exit(1)
	}
	wire := liquidity.PlanToWire(plan)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(wire)

	fmt.Fprintf(os.Stderr, "\n# action=%s steps=%d dry_run=%v\n", plan.Action, len(plan.Steps), plan.DryRun)
	fmt.Fprintf(os.Stderr, "# full-balance circle_gateway deposits; prepare_calls are unsigned (agent signs)\n")
	if len(plan.Steps) > 0 && len(plan.Steps[0].PrepareCalls) > 0 {
		fmt.Fprintf(os.Stderr, "# first step prepare: approve → %s, deposit → gateway wallet\n",
			plan.Steps[0].PrepareCalls[0].To)
	}
}

// demoLiveConsolidateExecute optionally broadcasts re-derived deposit txs when env is set.
// Prints tx hashes to stderr only (never keys, balances, or calldata).
func demoLiveConsolidateExecute() {
	key := strings.TrimSpace(os.Getenv("AGENT_PRIVATE_KEY"))
	if key == "" {
		fmt.Fprintln(os.Stderr, "# skip live execute: AGENT_PRIVATE_KEY unset")
		return
	}
	rpcs, err := loadDemoRPCs()
	if err != nil || len(rpcs) == 0 {
		fmt.Fprintln(os.Stderr, "# skip live execute: no testnet RPCs (RPC_URLS_JSON or RPC_URL_eip155_*)")
		return
	}
	guard := &liquidity.Guard{}
	if raw := strings.TrimSpace(os.Getenv("MAX_AMOUNT_ATOMIC")); raw != "" {
		d, err := decimal.NewFromString(raw)
		if err != nil || !d.IsPositive() {
			fmt.Fprintln(os.Stderr, "# skip live execute: MAX_AMOUNT_ATOMIC invalid")
			return
		}
		guard.MaxAmountAtomic = d
	}
	ex, err := execonchain.NewDepositExecutor(execonchain.Config{
		PrivateKeyHex: key,
		RPCs:          rpcs,
		Guard:         guard,
		WaitTimeout:   3 * time.Minute,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "# live execute configure error: %v\n", err)
		return
	}
	agent := ex.Address().Hex()
	// Client-asserted inventory for chains with RPCs — operator supplies real balances.
	// Demo uses env DEMO_AMOUNT_ATOMIC (default 1 atomic unit) on first configured chain.
	amt := strings.TrimSpace(os.Getenv("DEMO_AMOUNT_ATOMIC"))
	if amt == "" {
		amt = "1"
	}
	var chain string
	for k := range rpcs {
		chain = k
		break
	}
	usdc, ok := liquidity.DefaultUSDC(chain)
	if !ok {
		fmt.Fprintln(os.Stderr, "# skip live execute: no USDC for chain")
		return
	}
	inv := liquidity.Inventory{
		AgentAddress: agent,
		Balances: []liquidity.Balance{{
			ChainCAIP2: chain, Asset: usdc,
			AmountAtomic: decimal.RequireFromString(amt),
			Location:     liquidity.LocationNative,
		}},
	}
	plan, err := liquidity.PlanConsolidate(inv, nil, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "# live plan error: %v\n", err)
		return
	}
	if plan.Action != liquidity.ActionCircleGatewayConsolidate {
		fmt.Fprintf(os.Stderr, "# live plan action=%s (need circle_gateway_consolidate)\n", plan.Action)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	rcpt, err := ex.Execute(ctx, plan)
	if len(rcpt.TxHashes) > 0 {
		fmt.Fprintf(os.Stderr, "# tx_hashes=%v\n", rcpt.TxHashes)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "# live execute error: %v\n", err)
		return
	}
	fmt.Fprintln(os.Stderr, "# live execute ok")
}

func loadDemoRPCs() (map[string]string, error) {
	out := map[string]string{}
	if raw := strings.TrimSpace(os.Getenv("RPC_URLS_JSON")); raw != "" {
		if err := json.Unmarshal([]byte(raw), &out); err != nil {
			return nil, err
		}
	}
	const prefix = "RPC_URL_"
	for _, e := range os.Environ() {
		eq := strings.IndexByte(e, '=')
		if eq <= 0 {
			continue
		}
		name, val := e[:eq], e[eq+1:]
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		rest := strings.TrimPrefix(name, prefix)
		var caip string
		if strings.HasPrefix(rest, "eip155_") {
			caip = "eip155:" + strings.TrimPrefix(rest, "eip155_")
		} else {
			caip = strings.ReplaceAll(rest, "_", ":")
		}
		if strings.TrimSpace(val) != "" {
			out[caip] = strings.TrimSpace(val)
		}
	}
	return out, nil
}
