package planio

import (
	"context"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/kaimo-no/usdc-liquidity-orchestrator/pkg/liquidity"
	"github.com/kaimo-no/usdc-liquidity-orchestrator/pkg/types"
)

// RunPlan shortfall-plans and optionally executes. Pre-plan errors return StampFail
// without calling Execute. StampPlan uses SanitizeAPIError for execute failures.
func RunPlan(ctx context.Context, ex liquidity.Executor, req types.PlanRequest) (types.PlanResponse, StampOutcome) {
	if ex == nil {
		ex = liquidity.UnconfiguredExecutor{}
	}
	required, err := liquidity.RequiredFromWire(&req.Required, req.AmountOverride)
	if err != nil {
		return CodedPlanError(err)
	}
	inv, err := liquidity.InventoryFromWire(req.Inventory)
	if err != nil {
		return CodedPlanError(err)
	}
	orch := liquidity.OrchestrationFromWire(req.Orchestration)
	fee := liquidity.FeeConfigFromWire(req.FeeBps, req.FeeRecipient)

	plan, err := liquidity.PlanOrchestration(required, inv, orch, fee, nil)
	if err != nil {
		return CodedPlanError(err)
	}
	return stampAfterExecute(ctx, ex, plan, req.Execute)
}

// RunConsolidate plans Gateway deposits and optionally executes.
func RunConsolidate(ctx context.Context, ex liquidity.Executor, req types.ConsolidateRequest) (types.PlanResponse, StampOutcome) {
	if ex == nil {
		ex = liquidity.UnconfiguredExecutor{}
	}
	inv, err := liquidity.InventoryFromWire(req.Inventory)
	if err != nil {
		return CodedPlanError(err)
	}
	orch := liquidity.OrchestrationFromWire(req.Orchestration)

	plan, err := liquidity.PlanConsolidate(inv, orch, nil)
	if err != nil {
		return CodedPlanError(err)
	}
	return stampAfterExecute(ctx, ex, plan, req.Execute)
}

// RunPaymentFunding plans scenario full-funding and optionally executes.
func RunPaymentFunding(ctx context.Context, ex liquidity.Executor, req types.PaymentFundingRequest) (types.PlanResponse, StampOutcome) {
	if ex == nil {
		ex = liquidity.UnconfiguredExecutor{}
	}
	required, err := liquidity.RequiredFromWire(&req.Required, "")
	if err != nil {
		return CodedPlanError(err)
	}
	if log := strings.TrimSpace(req.Required.AmountLogicalAtomic); log != "" {
		if d, err := decimal.NewFromString(log); err == nil && d.IsPositive() && d.Equal(d.Truncate(0)) {
			required.AmountLogicalAtomic = d.Truncate(0)
		}
	}
	if req.Required.ScaleFactor > 0 {
		required.ScaleFactor = req.Required.ScaleFactor
	}

	inv, err := liquidity.InventoryFromWire(req.Inventory)
	if err != nil {
		return CodedPlanError(err)
	}
	sources, err := liquidity.FundingSourcesFromWire(req.Sources)
	if err != nil {
		return CodedPlanError(err)
	}

	plan, err := liquidity.PlanPaymentFunding(required, inv, sources, nil)
	if err != nil {
		return CodedPlanError(err)
	}
	return stampAfterExecute(ctx, ex, plan, req.Execute)
}

// ListChains returns the registered corridor discovery payload.
func ListChains() types.ChainsResponse {
	reg := liquidity.ListChains()
	out := make([]types.ChainInfo, 0, len(reg))
	for _, c := range reg {
		wallet, _ := liquidity.GatewayWalletAddress(c.CAIP2)
		out = append(out, types.ChainInfo{
			CAIP2: c.CAIP2, Name: c.Name, GatewayDomain: c.GatewayDomain,
			USDC: c.USDC, GatewayOK: c.GatewayOK, CCTPOK: c.CCTPOK,
			Testnet: c.Testnet, GatewayWallet: wallet,
		})
	}
	return types.ChainsResponse{Chains: out}
}

func stampAfterExecute(ctx context.Context, ex liquidity.Executor, plan liquidity.Plan, execute bool) (types.PlanResponse, StampOutcome) {
	wire := liquidity.PlanToWire(plan)
	var receipt liquidity.Receipt
	var execErr error
	if execute {
		receipt, execErr = ex.Execute(ctx, plan)
	}
	return StampPlan(wire, execute, receipt, execErr)
}
