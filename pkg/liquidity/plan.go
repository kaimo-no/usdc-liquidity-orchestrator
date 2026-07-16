// Package liquidity plans multi-chain USDC rebalancing for agentic commerce.
//
// Non-custodial: fund-movement steps always target the agent's own address
// (agent_self). Merchant pay_to is untrusted claim metadata for a later
// Payment-Signature — never a bridge/mint destination.
//
// Rails (naming avoids "gateway" alone to prevent MoR confusion):
//
//	circle_gateway_withdraw | circle_gateway_deposit_withdraw | cctp_fast
package liquidity

import (
	"strings"

	"github.com/shopspring/decimal"

	liqerr "github.com/kaimo-no/usdc-liquidity-orchestrator/pkg/errors"
	"github.com/kaimo-no/usdc-liquidity-orchestrator/pkg/types"
)

// PlanAction is the dry liquidity plan outcome.
type PlanAction string

const (
	ActionNoop                         PlanAction = "noop"
	ActionCircleGatewayWithdraw        PlanAction = "circle_gateway_withdraw"
	ActionCircleGatewayDepositWithdraw PlanAction = "circle_gateway_deposit_withdraw"
	ActionCCTPFast                     PlanAction = "cctp_fast"
	ActionInsufficient                 PlanAction = "insufficient"
	ActionCorridorUnsupported          PlanAction = "corridor_unsupported"
)

const (
	LocationNative         = "native"
	LocationCircleGateway  = "circle_gateway"
	RecipientRoleAgentSelf = "agent_self"
	RecipientRoleMerchant  = "merchant"
	AmountSourceProbe      = "probe"
	AmountSourceOverride   = "override"

	StepKindCircleGatewayWithdraw = "circle_gateway_withdraw"
	StepKindCircleGatewayDeposit  = "circle_gateway_deposit"
	StepKindCCTPBurn              = "cctp_burn"
	StepKindCCTPMint              = "cctp_mint"
	StepKindNote                  = "note"
)

// Required is the dest-chain USDC need (+ optional amount override rules).
type Required struct {
	Protocol     string
	ChainCAIP2   string
	Asset        string
	PayTo        string
	AmountAtomic decimal.Decimal
	AmountSource string
}

// Balance is one asserted inventory row.
type Balance struct {
	ChainCAIP2   string
	Asset        string
	AmountAtomic decimal.Decimal
	Location     string
}

// Inventory is agent-supplied balances (client-side only).
type Inventory struct {
	AgentAddress string
	Balances     []Balance
}

// PlanStep is one planned fund movement (agent_self recipients only).
type PlanStep struct {
	Kind           string
	FromChainCAIP2 string
	ToChainCAIP2   string
	Asset          string
	AmountAtomic   decimal.Decimal
	Recipient      string
	RecipientRole  string
}

// Plan is the pure planner output (Executed always false from PlanLiquidity).
type Plan struct {
	Action              PlanAction
	Required            Required
	Steps               []PlanStep
	Reason              string
	RecipientRole       string
	InventoryAsserted   bool
	InventoryUnverified bool
	Executed            bool
	DryRun              bool
	agentAddress        string
}

// RequiredFromWire builds Required from merchant-claim wire.
// amountOverride is accepted only when probe amount is empty.
func RequiredFromWire(lr *types.Required, amountOverride string) (Required, error) {
	if lr == nil {
		return Required{}, liqerr.New(liqerr.CodeInsufficientLiquidity,
			"liquidity: missing required (empty pay_to / no merchant claim)")
	}
	payTo := strings.TrimSpace(lr.PayTo)
	if payTo == "" {
		return Required{}, liqerr.New(liqerr.CodeInsufficientLiquidity,
			"liquidity: empty pay_to — refuse plan; never invent merchant recipient")
	}
	chain := strings.TrimSpace(lr.ChainCAIP2)
	asset := strings.TrimSpace(lr.Asset)
	amount, source, err := resolveAmount(strings.TrimSpace(lr.AmountAtomic), strings.TrimSpace(amountOverride))
	if err != nil {
		return Required{}, err
	}
	if chain == "" || asset == "" {
		return Required{}, liqerr.New(liqerr.CodeInvalidQuery,
			"liquidity: chain_caip2 and asset required (cannot invent; override only fills missing amount)")
	}
	if !amount.IsPositive() {
		return Required{}, liqerr.New(liqerr.CodeInvalidQuery,
			"liquidity: amount_atomic must be positive")
	}
	return Required{
		Protocol:     strings.TrimSpace(lr.Protocol),
		ChainCAIP2:   chain,
		Asset:        asset,
		PayTo:        payTo,
		AmountAtomic: amount,
		AmountSource: source,
	}, nil
}

func resolveAmount(probeAmt, override string) (decimal.Decimal, string, error) {
	switch {
	case override != "" && probeAmt != "":
		o, err := parseAtomic(override)
		if err != nil {
			return decimal.Zero, "", err
		}
		p, err := parseAtomic(probeAmt)
		if err != nil {
			return decimal.Zero, "", err
		}
		if !o.Equal(p) {
			return decimal.Zero, "", liqerr.New(liqerr.CodeInvalidQuery,
				"liquidity: amount_override refused — probe already has amount %s (cannot change)", probeAmt)
		}
		return p, AmountSourceProbe, nil
	case override != "" && probeAmt == "":
		o, err := parseAtomic(override)
		if err != nil {
			return decimal.Zero, "", err
		}
		return o, AmountSourceOverride, nil
	case probeAmt != "":
		p, err := parseAtomic(probeAmt)
		if err != nil {
			return decimal.Zero, "", err
		}
		return p, AmountSourceProbe, nil
	default:
		return decimal.Zero, "", liqerr.New(liqerr.CodeInvalidQuery,
			"liquidity: amount missing on probe and no amount_override — cannot plan")
	}
}

func parseAtomic(s string) (decimal.Decimal, error) {
	d, err := decimal.NewFromString(strings.TrimSpace(s))
	if err != nil {
		return decimal.Zero, liqerr.Wrap(liqerr.CodeInvalidQuery, err,
			"liquidity: amount_atomic %q is not a decimal", s)
	}
	if !d.Equal(d.Truncate(0)) {
		return decimal.Zero, liqerr.New(liqerr.CodeInvalidQuery,
			"liquidity: amount_atomic %q must be whole atomic units (no fractional)", s)
	}
	return d.Truncate(0), nil
}

// PlanLiquidity returns a pure dry plan. Never moves funds.
// Fund steps always target Inventory.AgentAddress with recipient_role=agent_self.
func PlanLiquidity(req Required, inv Inventory, g *Guard) (Plan, error) {
	if err := validateRequired(req); err != nil {
		return Plan{}, err
	}
	if err := g.CheckPrepare(req, inv); err != nil {
		return Plan{}, err
	}

	base := dryBase(req, inv)
	if hasNativeCover(req, inv) {
		base.Action = ActionNoop
		base.Reason = "dest native balance covers required amount"
		return base, nil
	}
	if early, ok := unsupportedOrEarly(req, base); ok {
		return early, nil
	}

	agent := strings.TrimSpace(inv.AgentAddress)
	if agent == "" {
		return Plan{}, liqerr.New(liqerr.CodeInvalidQuery,
			"liquidity: agent_address required to plan fund movement (agent_self recipient)")
	}

	// Only move the shortfall (required − dest native). Full required amount would
	// reject valid fragmented inventory (e.g. 20 Base + 30 Arb for 42 on Base).
	shortfall := req.AmountAtomic.Sub(sumMatching(req, inv, LocationNative, req.ChainCAIP2))
	if !shortfall.IsPositive() {
		base.Action = ActionNoop
		base.Reason = "dest native balance covers required amount"
		return base, nil
	}

	gwOK, cctpOK := corridorEligible(req.ChainCAIP2)
	if p, ok, err := tryBridgePlans(req, inv, agent, shortfall, gwOK, cctpOK, g, base); ok || err != nil {
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

func validateRequired(req Required) error {
	if strings.TrimSpace(req.PayTo) == "" {
		return liqerr.New(liqerr.CodeInsufficientLiquidity,
			"liquidity: empty pay_to — refuse plan")
	}
	if strings.TrimSpace(req.ChainCAIP2) == "" || strings.TrimSpace(req.Asset) == "" {
		return liqerr.New(liqerr.CodeInvalidQuery,
			"liquidity: chain_caip2 and asset required")
	}
	if !req.AmountAtomic.IsPositive() {
		return liqerr.New(liqerr.CodeInvalidQuery,
			"liquidity: amount_atomic must be positive")
	}
	return nil
}

func dryBase(req Required, inv Inventory) Plan {
	return Plan{
		Required:            req,
		InventoryAsserted:   true,
		InventoryUnverified: true,
		Executed:            false,
		DryRun:              true,
		agentAddress:        strings.TrimSpace(inv.AgentAddress),
	}
}

func unsupportedOrEarly(req Required, base Plan) (Plan, bool) {
	gwOK, cctpOK := corridorEligible(req.ChainCAIP2)
	if chainIsSolana(req.ChainCAIP2) || (!chainIsEVM(req.ChainCAIP2) && !gwOK && !cctpOK) {
		base.Action = ActionCorridorUnsupported
		base.Reason = "corridor_unsupported for dest chain (EVM-first; Solana/unknown not planned this cut)"
		return base, true
	}
	return base, false
}

func tryBridgePlans(req Required, inv Inventory, agent string, shortfall decimal.Decimal, gwOK, cctpOK bool, g *Guard, base Plan) (Plan, bool, error) {
	if gwOK && locationSum(req, inv, LocationCircleGateway).GreaterThanOrEqual(shortfall) {
		base.Action = ActionCircleGatewayWithdraw
		base.RecipientRole = RecipientRoleAgentSelf
		base.Reason = "circle_gateway balance covers shortfall; withdraw to agent_self on dest"
		base.Steps = []PlanStep{{
			Kind: StepKindCircleGatewayWithdraw, ToChainCAIP2: req.ChainCAIP2, Asset: req.Asset,
			AmountAtomic: shortfall, Recipient: agent, RecipientRole: RecipientRoleAgentSelf,
		}}
		p, err := finalizePlan(base, g)
		return p, true, err
	}
	if gwOK {
		if src, ok := findOtherNativeSource(req, inv, shortfall); ok {
			base.Action = ActionCircleGatewayDepositWithdraw
			base.RecipientRole = RecipientRoleAgentSelf
			base.Reason = "deposit source-chain native into circle_gateway then withdraw shortfall to dest agent_self"
			base.Steps = []PlanStep{
				{Kind: StepKindCircleGatewayDeposit, FromChainCAIP2: src.ChainCAIP2, Asset: req.Asset,
					AmountAtomic: shortfall, Recipient: agent, RecipientRole: RecipientRoleAgentSelf},
				{Kind: StepKindCircleGatewayWithdraw, ToChainCAIP2: req.ChainCAIP2, Asset: req.Asset,
					AmountAtomic: shortfall, Recipient: agent, RecipientRole: RecipientRoleAgentSelf},
			}
			p, err := finalizePlan(base, g)
			return p, true, err
		}
	}
	if cctpOK {
		if src, ok := findOtherNativeSource(req, inv, shortfall); ok {
			base.Action = ActionCCTPFast
			base.RecipientRole = RecipientRoleAgentSelf
			base.Reason = "cctp_fast burn shortfall on source, mint to agent_self on dest"
			base.Steps = []PlanStep{
				{Kind: StepKindCCTPBurn, FromChainCAIP2: src.ChainCAIP2, ToChainCAIP2: req.ChainCAIP2, Asset: req.Asset,
					AmountAtomic: shortfall, Recipient: agent, RecipientRole: RecipientRoleAgentSelf},
				{Kind: StepKindCCTPMint, ToChainCAIP2: req.ChainCAIP2, Asset: req.Asset,
					AmountAtomic: shortfall, Recipient: agent, RecipientRole: RecipientRoleAgentSelf},
			}
			p, err := finalizePlan(base, g)
			return p, true, err
		}
	}
	return base, false, nil
}

func finalizePlan(p Plan, g *Guard) (Plan, error) {
	if err := g.CheckPlan(p); err != nil {
		return Plan{}, err
	}
	return p, nil
}

func hasNativeCover(req Required, inv Inventory) bool {
	return sumMatching(req, inv, LocationNative, req.ChainCAIP2).GreaterThanOrEqual(req.AmountAtomic)
}

func locationSum(req Required, inv Inventory, loc string) decimal.Decimal {
	sum := decimal.Zero
	for _, b := range inv.Balances {
		if normalizeLoc(b.Location) != loc {
			continue
		}
		if !assetEqual(b.Asset, req.Asset, req.ChainCAIP2) {
			continue
		}
		sum = sum.Add(b.AmountAtomic)
	}
	return sum
}

func findOtherNativeSource(req Required, inv Inventory, need decimal.Decimal) (Balance, bool) {
	for _, b := range inv.Balances {
		if normalizeLoc(b.Location) != LocationNative {
			continue
		}
		if strings.TrimSpace(b.ChainCAIP2) == "" || strings.EqualFold(b.ChainCAIP2, req.ChainCAIP2) {
			continue
		}
		if chainIsSolana(b.ChainCAIP2) {
			continue
		}
		if !assetEqual(b.Asset, req.Asset, req.ChainCAIP2) && !looseUSDCAssetMatch(b.Asset, req.Asset) {
			continue
		}
		if b.AmountAtomic.GreaterThanOrEqual(need) {
			return b, true
		}
	}
	return Balance{}, false
}

func sumMatching(req Required, inv Inventory, loc, chain string) decimal.Decimal {
	sum := decimal.Zero
	for _, b := range inv.Balances {
		if normalizeLoc(b.Location) != loc {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(b.ChainCAIP2), strings.TrimSpace(chain)) {
			continue
		}
		if !assetEqual(b.Asset, req.Asset, req.ChainCAIP2) {
			continue
		}
		sum = sum.Add(b.AmountAtomic)
	}
	return sum
}

func normalizeLoc(loc string) string {
	loc = strings.TrimSpace(strings.ToLower(loc))
	if loc == "" {
		return LocationNative
	}
	// Bare "gateway" is forbidden (collides with HTTP gateway / MoR mental models).
	if loc == "gateway" {
		return "invalid_gateway_label"
	}
	return loc
}

func assetEqual(a, b, chainCAIP2 string) bool {
	a, b = strings.TrimSpace(a), strings.TrimSpace(b)
	if a == "" || b == "" {
		return false
	}
	if strings.HasPrefix(chainCAIP2, "solana:") {
		return a == b
	}
	return strings.EqualFold(a, b)
}

func looseUSDCAssetMatch(a, b string) bool {
	return assetEqual(a, b, "eip155:1")
}

// PlanToWire maps a Plan to the agent-facing types.Plan.
func PlanToWire(p Plan) types.Plan {
	steps := make([]types.PlanStep, 0, len(p.Steps))
	for _, s := range p.Steps {
		steps = append(steps, types.PlanStep{
			Kind:           s.Kind,
			FromChainCAIP2: s.FromChainCAIP2,
			ToChainCAIP2:   s.ToChainCAIP2,
			Asset:          s.Asset,
			AmountAtomic:   s.AmountAtomic.String(),
			Recipient:      s.Recipient,
			RecipientRole:  s.RecipientRole,
		})
	}
	var reqWire *types.Required
	if p.Required.PayTo != "" {
		reqWire = &types.Required{
			Protocol:     p.Required.Protocol,
			ChainCAIP2:   p.Required.ChainCAIP2,
			Asset:        p.Required.Asset,
			AmountAtomic: p.Required.AmountAtomic.String(),
			PayTo:        p.Required.PayTo,
			PayToRole:    RecipientRoleMerchant,
			Source:       "probe",
		}
	}
	return types.Plan{
		Action:              string(p.Action),
		Required:            reqWire,
		Steps:               steps,
		Reason:              p.Reason,
		RecipientRole:       p.RecipientRole,
		InventoryAsserted:   p.InventoryAsserted,
		InventoryUnverified: p.InventoryUnverified,
		Executed:            p.Executed,
		DryRun:              p.DryRun,
		AmountSource:        p.Required.AmountSource,
	}
}

// InventoryFromWire parses wire inventory into planner Inventory.
func InventoryFromWire(inv types.Inventory) (Inventory, error) {
	out := Inventory{AgentAddress: strings.TrimSpace(inv.AgentAddress)}
	for i, b := range inv.Balances {
		amt, err := parseAtomic(b.AmountAtomic)
		if err != nil {
			return Inventory{}, liqerr.Wrap(liqerr.CodeInvalidQuery, err,
				"liquidity: balances[%d].amount_atomic", i)
		}
		out.Balances = append(out.Balances, Balance{
			ChainCAIP2:   strings.TrimSpace(b.ChainCAIP2),
			Asset:        strings.TrimSpace(b.Asset),
			AmountAtomic: amt,
			Location:     b.Location,
		})
	}
	return out, nil
}
