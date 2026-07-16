// Command demo runs the hackathon worked example:
// merchant wants 42 USDC on Base; agent has 30 on Arbitrum + 20 on Base.
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
		base  = "eip155:8453"
		arb   = "eip155:42161"
		usdc  = "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"
		payTo = "0xMerchantOnBase0000000000000000000001"
		agent = "0xAgentSelf000000000000000000000000000001"
		need  = "42000000" // 42 USDC (6 decimals)
	)

	req := liquidity.Required{
		Protocol:     "x402",
		ChainCAIP2:   base,
		Asset:        usdc,
		PayTo:        payTo,
		AmountAtomic: decimal.RequireFromString(need),
		AmountSource: liquidity.AmountSourceProbe,
	}
	inv := liquidity.Inventory{
		AgentAddress: agent,
		Balances: []liquidity.Balance{
			{ChainCAIP2: arb, Asset: usdc, AmountAtomic: decimal.RequireFromString("30000000"), Location: liquidity.LocationNative},
			{ChainCAIP2: base, Asset: usdc, AmountAtomic: decimal.RequireFromString("20000000"), Location: liquidity.LocationNative},
		},
	}

	plan, err := liquidity.PlanLiquidity(req, inv, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "plan error: %v\n", err)
		os.Exit(1)
	}
	wire := liquidity.PlanToWire(plan)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(wire)

	fmt.Fprintf(os.Stderr, "\n# action=%s dry_run=%v executed=%v\n", plan.Action, plan.DryRun, plan.Executed)
	fmt.Fprintf(os.Stderr, "# shortfall on Base was 22 USDC → planned deposit from Arbitrum via circle_gateway\n")
	fmt.Fprintf(os.Stderr, "# recipients are always agent_self (%s), never merchant pay_to\n", agent)
}
