package liquidity

import (
	"sort"
	"strings"

	"github.com/shopspring/decimal"

	liqerr "github.com/kaimo-no/usdc-liquidity-orchestrator/pkg/errors"
)

// PlanConsolidate plans multi-chain full-balance Circle Gateway deposits (no pay_to/fee).
// Eligible: native USDC on GatewayOK chains with a known Gateway Wallet.
// Optional source allowlist; AllowCircleGateway=false → invalid_query.
func PlanConsolidate(inv Inventory, o *Orchestration, g *Guard) (Plan, error) {
	if o != nil && !o.gatewayAllowed() {
		return Plan{}, liqerr.New(liqerr.CodeInvalidQuery,
			"liquidity: allow_circle_gateway=false refuses consolidate")
	}
	agent := strings.TrimSpace(inv.AgentAddress)
	if agent == "" {
		return Plan{}, liqerr.New(liqerr.CodeInvalidQuery,
			"liquidity: agent_address required to plan consolidate (agent_self recipient)")
	}
	if err := g.CheckAgent(inv); err != nil {
		return Plan{}, err
	}

	sources := sourceAllowlist(o)
	// chain → sum of eligible native USDC rows
	sums := map[string]decimal.Decimal{}
	assets := map[string]string{}
	canonical := map[string]string{} // lower key → registry CAIP-2

	for _, b := range inv.Balances {
		if normalizeLoc(b.Location) != LocationNative {
			continue
		}
		chainIn := strings.TrimSpace(b.ChainCAIP2)
		if chainIn == "" {
			continue
		}
		info, ok := LookupChain(chainIn)
		if !ok || !info.GatewayOK {
			continue
		}
		if _, ok := GatewayWalletAddress(info.CAIP2); !ok {
			continue
		}
		if len(sources) > 0 && !chainInList(info.CAIP2, sources) {
			continue
		}
		if !nativeUSDCOnChain(b.Asset, info.CAIP2) {
			continue
		}
		if !b.AmountAtomic.IsPositive() {
			continue
		}
		key := strings.ToLower(info.CAIP2)
		canonical[key] = info.CAIP2
		sums[key] = sums[key].Add(b.AmountAtomic)
		if _, set := assets[key]; !set {
			assets[key] = stepAssetForChain(b.Asset, info.CAIP2, info.USDC)
		}
	}

	base := Plan{
		InventoryAsserted:   true,
		InventoryUnverified: true,
		Executed:            false,
		DryRun:              true,
		agentAddress:        agent,
		RecipientRole:       RecipientRoleAgentSelf,
	}

	if len(sums) == 0 {
		base.Action = ActionNoop
		base.Reason = "noop: no eligible native USDC on GatewayOK chains to consolidate"
		return base, nil
	}

	keys := make([]string, 0, len(sums))
	for k := range sums {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	steps := make([]PlanStep, 0, len(keys))
	for _, k := range keys {
		amt := sums[k]
		if !amt.IsPositive() {
			continue
		}
		chain := canonical[k]
		asset := assets[k]
		if u, ok := DefaultUSDC(chain); ok && (asset == "" || strings.EqualFold(asset, "USDC")) {
			asset = u
		}
		steps = append(steps, PlanStep{
			Kind:           StepKindCircleGatewayDeposit,
			FromChainCAIP2: chain,
			Asset:          asset,
			AmountAtomic:   amt,
			Recipient:      agent,
			RecipientRole:  RecipientRoleAgentSelf,
		})
	}
	if len(steps) == 0 {
		base.Action = ActionNoop
		base.Reason = "noop: no eligible native USDC on GatewayOK chains to consolidate"
		return base, nil
	}

	base.Action = ActionCircleGatewayConsolidate
	base.Reason = "deposit full native USDC balances into circle_gateway (agent_self)"
	base.Steps = steps

	if err := attachDepositPrepareCallsOnPlan(&base); err != nil {
		return Plan{}, err
	}
	if err := g.CheckPlan(base); err != nil {
		return Plan{}, err
	}
	return base, nil
}

func nativeUSDCOnChain(asset, chainCAIP2 string) bool {
	usdc, ok := DefaultUSDC(chainCAIP2)
	if !ok {
		return false
	}
	return sameChainUSDCMatch(asset, usdc, chainCAIP2)
}
