package liquidity

import (
	"context"

	liqerr "github.com/kaimo-no/usdc-liquidity-orchestrator/pkg/errors"
)

// Receipt is a future execute receipt (unused while unconfigured).
type Receipt struct {
	TxHashes []string
	Note     string
}

// Executor runs a plan (circle_gateway / cctp). This cut ships the fail-closed stub.
type Executor interface {
	Execute(ctx context.Context, p Plan) (Receipt, error)
}

// UnconfiguredExecutor always fails closed after validating the plan.
// No live Circle Gateway / CCTP SDK in this cut — safe for demos and CI.
type UnconfiguredExecutor struct {
	Guard *Guard
}

// Execute validates plan first, then returns a typed error. Never succeeds.
func (e UnconfiguredExecutor) Execute(ctx context.Context, p Plan) (Receipt, error) {
	_ = ctx
	g := e.Guard
	if err := g.CheckPlan(p); err != nil {
		return Receipt{}, err
	}
	switch p.Action {
	case ActionInsufficient, ActionCorridorUnsupported:
		return Receipt{}, liqerr.New(liqerr.CodeInsufficientLiquidity,
			"liquidity execute: plan action %s — %s", p.Action, p.Reason)
	case ActionNoop:
		return Receipt{}, liqerr.New(liqerr.CodeLiquidityRailUnavailable,
			"liquidity execute: not configured (noop dry plan is not an on-chain success)")
	default:
		return Receipt{}, liqerr.New(liqerr.CodeLiquidityRailUnavailable,
			"liquidity execute: circle_gateway/cctp not configured — refuse %s", p.Action)
	}
}
