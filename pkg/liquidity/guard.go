package liquidity

import (
	"strings"

	"github.com/shopspring/decimal"

	liqerr "github.com/kaimo-no/usdc-liquidity-orchestrator/pkg/errors"
)

// Guard governs client-side liquidity prepare/execute only.
// Prepare never transfers to merchant pay_to.
type Guard struct {
	MaxAmountAtomic       decimal.Decimal
	AllowedAgentAddresses []string
}

// CheckPrepare validates required + inventory before planning.
func (g *Guard) CheckPrepare(req Required, inv Inventory) error {
	if g == nil {
		return nil
	}
	if g.MaxAmountAtomic.IsPositive() && req.AmountAtomic.GreaterThan(g.MaxAmountAtomic) {
		return liqerr.New(liqerr.CodeInsufficientLiquidity,
			"liquidity: required amount %s exceeds MaxAmountAtomic %s",
			req.AmountAtomic.String(), g.MaxAmountAtomic.String())
	}
	if len(g.AllowedAgentAddresses) == 0 {
		return nil
	}
	if strings.TrimSpace(inv.AgentAddress) == "" {
		return liqerr.New(liqerr.CodeInvalidQuery,
			"liquidity: agent_address required when AllowedAgentAddresses is set")
	}
	if !agentAddrAllowed(g.AllowedAgentAddresses, inv.AgentAddress, req.ChainCAIP2) {
		return liqerr.New(liqerr.CodeInvalidQuery,
			"liquidity: agent_address %q is not in AllowedAgentAddresses", inv.AgentAddress)
	}
	return nil
}

// CheckPlan refuses non-agent_self fund steps and pay_to-as-recipient.
// Nil receiver is safe (used by bare UnconfiguredExecutor{}).
func (g *Guard) CheckPlan(p Plan) error {
	agent := resolvePlanAgent(p)
	for _, s := range p.Steps {
		if err := checkFundStep(s, p.Required, agent, g); err != nil {
			return err
		}
	}
	if g != nil && g.MaxAmountAtomic.IsPositive() && p.Required.AmountAtomic.GreaterThan(g.MaxAmountAtomic) {
		return liqerr.New(liqerr.CodeInsufficientLiquidity,
			"liquidity: plan amount %s exceeds MaxAmountAtomic %s",
			p.Required.AmountAtomic.String(), g.MaxAmountAtomic.String())
	}
	return nil
}

func resolvePlanAgent(p Plan) string {
	agent := strings.TrimSpace(p.agentAddress)
	if agent != "" {
		return agent
	}
	for _, s := range p.Steps {
		if r := strings.TrimSpace(s.Recipient); r != "" {
			return r
		}
	}
	return ""
}

func checkFundStep(s PlanStep, req Required, agent string, g *Guard) error {
	if !isFundMovingKind(s.Kind) {
		return nil
	}
	if s.RecipientRole != RecipientRoleAgentSelf {
		return liqerr.New(liqerr.CodeInvalidQuery,
			"liquidity: fund-moving step kind %q must use recipient_role=agent_self, got %q",
			s.Kind, s.RecipientRole)
	}
	rec := strings.TrimSpace(s.Recipient)
	if rec == "" {
		return liqerr.New(liqerr.CodeInvalidQuery,
			"liquidity: fund-moving step missing recipient (agent_self required)")
	}
	if payTo := strings.TrimSpace(req.PayTo); payTo != "" && addrEqual(rec, payTo, req.ChainCAIP2) {
		return liqerr.New(liqerr.CodeInvalidQuery,
			"liquidity: refuse transfer to merchant pay_to — prepare funds agent_self only")
	}
	if agent != "" && !addrEqual(rec, agent, req.ChainCAIP2) {
		return liqerr.New(liqerr.CodeInvalidQuery,
			"liquidity: step recipient %q must equal agent_address", rec)
	}
	if g != nil && len(g.AllowedAgentAddresses) > 0 && !agentAddrAllowed(g.AllowedAgentAddresses, rec, req.ChainCAIP2) {
		return liqerr.New(liqerr.CodeInvalidQuery,
			"liquidity: step recipient %q not in AllowedAgentAddresses", rec)
	}
	return nil
}

func isFundMovingKind(kind string) bool {
	switch kind {
	case StepKindCircleGatewayWithdraw, StepKindCircleGatewayDeposit, StepKindCCTPBurn, StepKindCCTPMint:
		return true
	default:
		return false
	}
}

func agentAddrAllowed(allowed []string, addr, chainCAIP2 string) bool {
	caseSensitive := strings.HasPrefix(chainCAIP2, "solana:")
	return addrInSet(allowed, addr, caseSensitive)
}

func addrEqual(a, b, chainCAIP2 string) bool {
	a, b = strings.TrimSpace(a), strings.TrimSpace(b)
	if a == "" || b == "" {
		return false
	}
	if strings.HasPrefix(chainCAIP2, "solana:") {
		return a == b
	}
	if strings.HasPrefix(a, "0x") || strings.HasPrefix(b, "0x") {
		return strings.EqualFold(a, b)
	}
	return strings.EqualFold(a, b)
}

func addrInSet(allowed []string, addr string, caseSensitive bool) bool {
	addr = strings.TrimSpace(addr)
	for _, a := range allowed {
		a = strings.TrimSpace(a)
		if caseSensitive {
			if a == addr {
				return true
			}
			continue
		}
		if strings.EqualFold(a, addr) {
			return true
		}
	}
	return false
}
