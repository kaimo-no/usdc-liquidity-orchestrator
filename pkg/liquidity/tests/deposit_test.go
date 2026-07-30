package liquidity_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	liqerr "github.com/kaimo-no/usdc-liquidity-orchestrator/pkg/errors"
	"github.com/kaimo-no/usdc-liquidity-orchestrator/pkg/liquidity"
	"github.com/kaimo-no/usdc-liquidity-orchestrator/pkg/types"
)

func TestPlanGatewayDeposit_HappyExactN(t *testing.T) {
	inv := liquidity.Inventory{
		AgentAddress: agentAddr,
		Balances: []liquidity.Balance{{
			ChainCAIP2: arcCAIP2, Asset: arcUSDC,
			AmountAtomic: decimal.RequireFromString("5000"), Location: liquidity.LocationNative,
		}},
	}
	p, err := liquidity.PlanGatewayDeposit(inv, arcCAIP2, decimal.RequireFromString("1000"), nil, nil)
	require.NoError(t, err)
	assert.Equal(t, liquidity.ActionCircleGatewayDeposit, p.Action)
	assert.True(t, p.DryRun)
	assert.False(t, p.Executed)
	assert.Empty(t, p.Required.PayTo)
	assert.Nil(t, p.Fee)
	require.Len(t, p.Steps, 1)
	s := p.Steps[0]
	assert.Equal(t, liquidity.StepKindCircleGatewayDeposit, s.Kind)
	assert.Equal(t, arcCAIP2, s.FromChainCAIP2)
	assert.True(t, s.AmountAtomic.Equal(decimal.RequireFromString("1000")))
	assert.Equal(t, agentAddr, s.Recipient)
	assert.Equal(t, liquidity.RecipientRoleAgentSelf, s.RecipientRole)
	require.Len(t, s.PrepareCalls, 2)
}

func TestPlanGatewayDeposit_NativeEqualsN_OK(t *testing.T) {
	inv := liquidity.Inventory{
		AgentAddress: agentAddr,
		Balances: []liquidity.Balance{{
			ChainCAIP2: baseSepCAIP2, Asset: baseSepUSDC,
			AmountAtomic: decimal.RequireFromString("1000"), Location: liquidity.LocationNative,
		}},
	}
	p, err := liquidity.PlanGatewayDeposit(inv, baseSepCAIP2, decimal.RequireFromString("1000"), nil, nil)
	require.NoError(t, err)
	require.Len(t, p.Steps, 1)
	assert.True(t, p.Steps[0].AmountAtomic.Equal(decimal.RequireFromString("1000")))
}

func TestPlanGatewayDeposit_Underfunded_HardError(t *testing.T) {
	inv := liquidity.Inventory{
		AgentAddress: agentAddr,
		Balances: []liquidity.Balance{{
			ChainCAIP2: arcCAIP2, Asset: arcUSDC,
			AmountAtomic: decimal.RequireFromString("100"), Location: liquidity.LocationNative,
		}},
	}
	_, err := liquidity.PlanGatewayDeposit(inv, arcCAIP2, decimal.RequireFromString("1000"), nil, nil)
	require.Error(t, err)
	assert.Equal(t, liqerr.CodeInsufficientLiquidity, liqerr.CodeOf(err))
}

func TestPlanGatewayDeposit_MissingSourceAgentAmount(t *testing.T) {
	inv := liquidity.Inventory{
		AgentAddress: agentAddr,
		Balances: []liquidity.Balance{{
			ChainCAIP2: arcCAIP2, Asset: arcUSDC,
			AmountAtomic: decimal.RequireFromString("1000"), Location: liquidity.LocationNative,
		}},
	}
	_, err := liquidity.PlanGatewayDeposit(inv, "", decimal.RequireFromString("1000"), nil, nil)
	require.Error(t, err)
	assert.Equal(t, liqerr.CodeInvalidQuery, liqerr.CodeOf(err))

	_, err = liquidity.PlanGatewayDeposit(liquidity.Inventory{}, arcCAIP2, decimal.RequireFromString("1000"), nil, nil)
	require.Error(t, err)
	assert.Equal(t, liqerr.CodeInvalidQuery, liqerr.CodeOf(err))

	_, err = liquidity.PlanGatewayDeposit(inv, arcCAIP2, decimal.Zero, nil, nil)
	require.Error(t, err)
	assert.Equal(t, liqerr.CodeInvalidQuery, liqerr.CodeOf(err))
}

func TestPlanGatewayDeposit_NonGatewayOK_AndAllowFalse(t *testing.T) {
	inv := liquidity.Inventory{
		AgentAddress: agentAddr,
		Balances: []liquidity.Balance{{
			ChainCAIP2: "eip155:999999", Asset: "USDC",
			AmountAtomic: decimal.RequireFromString("1000"), Location: liquidity.LocationNative,
		}},
	}
	_, err := liquidity.PlanGatewayDeposit(inv, "eip155:999999", decimal.RequireFromString("1000"), nil, nil)
	require.Error(t, err)
	assert.Equal(t, liqerr.CodeInvalidQuery, liqerr.CodeOf(err))

	f := false
	orch := &liquidity.Orchestration{AllowCircleGateway: &f}
	inv2 := liquidity.Inventory{
		AgentAddress: agentAddr,
		Balances: []liquidity.Balance{{
			ChainCAIP2: arcCAIP2, Asset: arcUSDC,
			AmountAtomic: decimal.RequireFromString("1000"), Location: liquidity.LocationNative,
		}},
	}
	_, err = liquidity.PlanGatewayDeposit(inv2, arcCAIP2, decimal.RequireFromString("1000"), orch, nil)
	require.Error(t, err)
	assert.Equal(t, liqerr.CodeInvalidQuery, liqerr.CodeOf(err))
}

func TestPlanGatewayDeposit_SourceAllowlist(t *testing.T) {
	inv := liquidity.Inventory{
		AgentAddress: agentAddr,
		Balances: []liquidity.Balance{
			{ChainCAIP2: arcCAIP2, Asset: arcUSDC, AmountAtomic: decimal.RequireFromString("1000"), Location: liquidity.LocationNative},
			{ChainCAIP2: baseSepCAIP2, Asset: baseSepUSDC, AmountAtomic: decimal.RequireFromString("9000"), Location: liquidity.LocationNative},
		},
	}
	orch := &liquidity.Orchestration{SourceChainCAIP2s: []string{baseSepCAIP2}}
	_, err := liquidity.PlanGatewayDeposit(inv, arcCAIP2, decimal.RequireFromString("1000"), orch, nil)
	require.Error(t, err)
	assert.Equal(t, liqerr.CodeInvalidQuery, liqerr.CodeOf(err))

	p, err := liquidity.PlanGatewayDeposit(inv, baseSepCAIP2, decimal.RequireFromString("1000"), orch, nil)
	require.NoError(t, err)
	assert.Equal(t, baseSepCAIP2, p.Steps[0].FromChainCAIP2)
	// Exact N, not full multi-row balance on other chain.
	assert.True(t, p.Steps[0].AmountAtomic.Equal(decimal.RequireFromString("1000")))
}

func TestPlanGatewayDeposit_MaxAmountAtomic(t *testing.T) {
	inv := liquidity.Inventory{
		AgentAddress: agentAddr,
		Balances: []liquidity.Balance{{
			ChainCAIP2: arcCAIP2, Asset: arcUSDC,
			AmountAtomic: decimal.RequireFromString("5000"), Location: liquidity.LocationNative,
		}},
	}
	g := &liquidity.Guard{MaxAmountAtomic: decimal.RequireFromString("100")}
	_, err := liquidity.PlanGatewayDeposit(inv, arcCAIP2, decimal.RequireFromString("1000"), nil, g)
	require.Error(t, err)
	assert.Equal(t, liqerr.CodeInsufficientLiquidity, liqerr.CodeOf(err))
}

func TestPlanGatewayDeposit_PlanToWire_OmitsRequired(t *testing.T) {
	inv := liquidity.Inventory{
		AgentAddress: agentAddr,
		Balances: []liquidity.Balance{{
			ChainCAIP2: arcCAIP2, Asset: arcUSDC,
			AmountAtomic: decimal.RequireFromString("1000"), Location: liquidity.LocationNative,
		}},
	}
	p, err := liquidity.PlanGatewayDeposit(inv, arcCAIP2, decimal.RequireFromString("500"), nil, nil)
	require.NoError(t, err)
	w := liquidity.PlanToWire(p)
	assert.Nil(t, w.Required)
	assert.Empty(t, w.AmountSource)
	assert.Equal(t, string(liquidity.ActionCircleGatewayDeposit), w.Action)
}

func TestUnconfiguredExecutor_Deposit_RailUnavailable(t *testing.T) {
	inv := liquidity.Inventory{
		AgentAddress: agentAddr,
		Balances: []liquidity.Balance{{
			ChainCAIP2: arcCAIP2, Asset: arcUSDC,
			AmountAtomic: decimal.RequireFromString("1000"), Location: liquidity.LocationNative,
		}},
	}
	p, err := liquidity.PlanGatewayDeposit(inv, arcCAIP2, decimal.RequireFromString("500"), nil, nil)
	require.NoError(t, err)
	_, err = liquidity.UnconfiguredExecutor{}.Execute(context.Background(), p)
	require.Error(t, err)
	assert.Equal(t, liqerr.CodeLiquidityRailUnavailable, liqerr.CodeOf(err))
}

func TestTypes_LandRequired_OmitsPayToKeys(t *testing.T) {
	r := types.Required{
		ChainCAIP2:   arcCAIP2,
		Asset:        arcUSDC,
		AmountAtomic: "1000",
		Source:       "self",
	}
	b, err := json.Marshal(r)
	require.NoError(t, err)
	assert.NotContains(t, string(b), "pay_to")
	assert.Contains(t, string(b), "amount_atomic")
}
