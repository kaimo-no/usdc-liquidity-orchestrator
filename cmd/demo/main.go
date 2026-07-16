// Command demo runs the hackathon worked example:
// merchant wants 42 USDC on Arc Testnet via x402; agent funds from circle_gateway
// unified balance (shortfall-only to agent_self) with optional kaimo fee.
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/shopspring/decimal"

	"github.com/kaimo-no/usdc-liquidity-orchestrator/pkg/liquidity"
)

func main() {
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
			// Unified Gateway balance (preferred source).
			{Asset: "USDC", AmountAtomic: decimal.RequireFromString("100000000"), Location: liquidity.LocationCircleGateway},
			// Partial native on Arc — shortfall only is moved.
			{ChainCAIP2: arc, Asset: arcUSDC, AmountAtomic: decimal.RequireFromString("20000000"), Location: liquidity.LocationNative},
		},
	}
	orch := &liquidity.Orchestration{
		TargetChainCAIP2: arc,
		PreferRail:       liquidity.PreferRailCircleGateway,
	}
	// allow_circle_gateway defaults true when nil
	fee := &liquidity.FeeConfig{
		Bps:       25, // 0.25%
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
