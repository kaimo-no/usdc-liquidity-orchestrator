// Command demo runs hackathon worked examples:
// 1) shortfall Gateway withdraw to agent_self on Arc Testnet
// 2) multi-chain native consolidate into Circle Gateway (unsigned prepare_calls)
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/shopspring/decimal"

	"github.com/kaimo-no/usdc-liquidity-orchestrator/pkg/liquidity"
)

func main() {
	demoShortfallPlan()
	fmt.Fprint(os.Stderr, "\n--- consolidate ---\n\n")
	demoConsolidate()
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
