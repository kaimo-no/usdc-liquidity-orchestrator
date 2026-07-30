package liquidity

import (
	"strings"

	"github.com/shopspring/decimal"

	liqerr "github.com/kaimo-no/usdc-liquidity-orchestrator/pkg/errors"
)

// PlanSelfRebalance plans shortfall-only rebalance to land N on dest agent_self (no pay_to/fee).
// Prefer Gateway rails; uncovered → ActionInsufficient plan (not hard error).
func PlanSelfRebalance(
	destChainCAIP2 string,
	amountAtomic decimal.Decimal,
	inv Inventory,
	o *Orchestration,
	g *Guard,
) (Plan, error) {
	agent, asset, dest, err := validateSelfLand(destChainCAIP2, amountAtomic, inv, o)
	if err != nil {
		return Plan{}, err
	}
	if err := g.CheckAgent(inv); err != nil {
		return Plan{}, err
	}

	req := Required{
		ChainCAIP2:   dest,
		Asset:        asset,
		AmountAtomic: amountAtomic,
		AmountSource: AmountSourceSelf,
	}
	base := dryBase(req, inv)
	base.selfRebalance = true
	base.RecipientRole = RecipientRoleAgentSelf

	if hasNativeCover(req, inv) {
		base.Action = ActionNoop
		base.Reason = "dest native balance covers land amount"
		return base, nil
	}
	if early, ok := unsupportedOrEarly(req, base); ok {
		return early, nil
	}

	shortfall := req.AmountAtomic.Sub(sumMatchingNative(req, inv, req.ChainCAIP2))
	if !shortfall.IsPositive() {
		base.Action = ActionNoop
		base.Reason = "dest native balance covers land amount"
		return base, nil
	}

	gwOK, cctpOK := corridorEligible(req.ChainCAIP2)
	if o != nil && !o.gatewayAllowed() {
		gwOK = false
	}
	prefer := PreferRailAuto
	if o != nil && strings.TrimSpace(o.PreferRail) != "" {
		prefer = strings.ToLower(strings.TrimSpace(o.PreferRail))
	}
	sources := sourceAllowlist(o)

	if p, ok, err := tryBridgePlans(req, inv, agent, shortfall, gwOK, cctpOK, prefer, sources, nil, g, base); ok || err != nil {
		return p, err
	}
	if !gwOK && !cctpOK {
		base.Action = ActionCorridorUnsupported
		base.Reason = "no circle_gateway/cctp corridor for dest and dest native insufficient"
		return base, nil
	}
	base.Action = ActionInsufficient
	base.Reason = "insufficient_liquidity: no dest native, circle_gateway, or cross-chain source covers shortfall"
	return base, nil
}

// validateSelfLand checks dest/amount/agent for self-land plans (no merchant Required).
func validateSelfLand(
	destChainCAIP2 string,
	amountAtomic decimal.Decimal,
	inv Inventory,
	o *Orchestration,
) (agent, asset, dest string, err error) {
	agent = strings.TrimSpace(inv.AgentAddress)
	if agent == "" {
		return "", "", "", liqerr.New(liqerr.CodeInvalidQuery,
			"liquidity: agent_address required to plan self-rebalance (agent_self recipient)")
	}
	if !amountAtomic.IsPositive() || !amountAtomic.Equal(amountAtomic.Truncate(0)) {
		return "", "", "", liqerr.New(liqerr.CodeInvalidQuery,
			"liquidity: land amount_atomic must be positive whole atomic units")
	}
	destIn := strings.TrimSpace(destChainCAIP2)
	if destIn == "" {
		return "", "", "", liqerr.New(liqerr.CodeInvalidQuery,
			"liquidity: dest_chain_caip2 required for self-rebalance")
	}
	info, ok := LookupChain(destIn)
	if ok {
		dest = info.CAIP2
	} else {
		dest = destIn
	}
	if err := validateOrchestration(Required{ChainCAIP2: dest}, o); err != nil {
		return "", "", "", err
	}
	usdc, ok := DefaultUSDC(dest)
	if !ok || usdc == "" {
		// Unknown dest: still dry-plan so unsupportedOrEarly can stamp corridor_unsupported.
		if !chainIsEVM(dest) && !chainIsSolana(dest) {
			return "", "", "", liqerr.New(liqerr.CodeInvalidQuery,
				"liquidity: dest_chain_caip2 must be eip155 or solana CAIP-2")
		}
		usdc = "USDC"
	}
	return agent, usdc, dest, nil
}
