package liquidity

import (
	"strings"

	"github.com/shopspring/decimal"

	liqerr "github.com/kaimo-no/usdc-liquidity-orchestrator/pkg/errors"
)

// PlanGatewayDeposit plans a fixed-N native USDC deposit into Circle Gateway (no pay_to/fee).
// MVP: single explicit source chain; hard error if native on source < amount.
func PlanGatewayDeposit(
	inv Inventory,
	sourceChainCAIP2 string,
	amountAtomic decimal.Decimal,
	o *Orchestration,
	g *Guard,
) (Plan, error) {
	agent, source, asset, err := validateGatewayDeposit(inv, sourceChainCAIP2, amountAtomic, o, g)
	if err != nil {
		return Plan{}, err
	}

	req := Required{
		ChainCAIP2:   source,
		Asset:        asset,
		AmountAtomic: amountAtomic,
	}
	native := sumMatchingNative(req, inv, source)
	if native.LessThan(amountAtomic) {
		return Plan{}, liqerr.New(liqerr.CodeInsufficientLiquidity,
			"liquidity: insufficient native USDC on source for deposit amount")
	}

	base := Plan{
		Action:              ActionCircleGatewayDeposit,
		RecipientRole:       RecipientRoleAgentSelf,
		InventoryAsserted:   true,
		InventoryUnverified: true,
		Executed:            false,
		DryRun:              true,
		agentAddress:        agent,
		Reason:              "deposit fixed amount native USDC into circle_gateway (agent_self)",
		Steps: []PlanStep{{
			Kind:           StepKindCircleGatewayDeposit,
			FromChainCAIP2: source,
			Asset:          asset,
			AmountAtomic:   amountAtomic,
			Recipient:      agent,
			RecipientRole:  RecipientRoleAgentSelf,
		}},
	}
	if err := attachDepositPrepareCallsOnPlan(&base); err != nil {
		return Plan{}, err
	}
	if err := g.CheckPlan(base); err != nil {
		return Plan{}, err
	}
	return base, nil
}

func validateGatewayDeposit(
	inv Inventory,
	sourceChainCAIP2 string,
	amountAtomic decimal.Decimal,
	o *Orchestration,
	g *Guard,
) (agent, source, asset string, err error) {
	if o != nil && !o.gatewayAllowed() {
		return "", "", "", liqerr.New(liqerr.CodeInvalidQuery,
			"liquidity: allow_circle_gateway=false refuses deposit")
	}
	agent = strings.TrimSpace(inv.AgentAddress)
	if agent == "" {
		return "", "", "", liqerr.New(liqerr.CodeInvalidQuery,
			"liquidity: agent_address required to plan deposit (agent_self recipient)")
	}
	if !amountAtomic.IsPositive() || !amountAtomic.Equal(amountAtomic.Truncate(0)) {
		return "", "", "", liqerr.New(liqerr.CodeInvalidQuery,
			"liquidity: deposit amount_atomic must be positive whole atomic units")
	}
	sourceIn := strings.TrimSpace(sourceChainCAIP2)
	if sourceIn == "" {
		return "", "", "", liqerr.New(liqerr.CodeInvalidQuery,
			"liquidity: source_chain_caip2 required for deposit")
	}
	info, ok := LookupChain(sourceIn)
	if !ok || !info.GatewayOK {
		return "", "", "", liqerr.New(liqerr.CodeInvalidQuery,
			"liquidity: source chain is not GatewayOK for deposit")
	}
	if _, ok := GatewayWalletAddress(info.CAIP2); !ok {
		return "", "", "", liqerr.New(liqerr.CodeInvalidQuery,
			"liquidity: no gateway wallet for source chain")
	}
	sources := sourceAllowlist(o)
	if len(sources) > 0 && !chainInList(info.CAIP2, sources) {
		return "", "", "", liqerr.New(liqerr.CodeInvalidQuery,
			"liquidity: source_chain_caip2 not in orchestration.source_chain_caip2s allowlist")
	}
	usdc, ok := DefaultUSDC(info.CAIP2)
	if !ok || usdc == "" {
		return "", "", "", liqerr.New(liqerr.CodeInvalidQuery,
			"liquidity: no registered USDC for source chain")
	}
	if err := g.CheckAgent(inv); err != nil {
		return "", "", "", err
	}
	return agent, info.CAIP2, usdc, nil
}
