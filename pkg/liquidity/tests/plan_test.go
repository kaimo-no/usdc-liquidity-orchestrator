package liquidity_test

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	liqerr "github.com/kaimo-no/usdc-liquidity-orchestrator/pkg/errors"
	"github.com/kaimo-no/usdc-liquidity-orchestrator/pkg/liquidity"
	"github.com/kaimo-no/usdc-liquidity-orchestrator/pkg/types"
)

const (
	baseCAIP2     = "eip155:8453"
	arbCAIP2      = "eip155:42161"
	solCAIP2      = "solana:5eykt4UsFv8P8NJdTREpY1vzqKqZKvdp"
	baseUSDC      = "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"
	merchantPayTo = "0xMerchantPayTo000000000000000000000001"
	agentAddr     = "0xAgentSelf000000000000000000000000000001"
	platformMoR   = "0xPlatformMoRWallet000000000000000000001"
)

func baseRequired() liquidity.Required {
	return liquidity.Required{
		Protocol:     "x402",
		ChainCAIP2:   baseCAIP2,
		Asset:        baseUSDC,
		PayTo:        merchantPayTo,
		AmountAtomic: decimal.RequireFromString("1000000"),
		AmountSource: liquidity.AmountSourceProbe,
	}
}

func wireLR(payTo, chain, asset, amount string) *types.Required {
	return &types.Required{
		Protocol: "x402", PayTo: payTo, PayToRole: "merchant",
		ChainCAIP2: chain, Asset: asset, AmountAtomic: amount, Source: "probe",
	}
}

func TestRequiredFromWire_EmptyPayTo_FailClosed(t *testing.T) {
	_, err := liquidity.RequiredFromWire(wireLR("", baseCAIP2, baseUSDC, "1000000"), "")
	require.Error(t, err)
	assert.Equal(t, liqerr.CodeInsufficientLiquidity, liqerr.CodeOf(err))
}

func TestRequiredFromWire_AmountOverrideOnlyWhenMissing(t *testing.T) {
	_, err := liquidity.RequiredFromWire(wireLR(merchantPayTo, baseCAIP2, baseUSDC, "1000000"), "2000000")
	require.Error(t, err)
	assert.Equal(t, liqerr.CodeInvalidQuery, liqerr.CodeOf(err))

	req, err := liquidity.RequiredFromWire(wireLR(merchantPayTo, baseCAIP2, baseUSDC, ""), "500000")
require.NoError(t, err)
	assert.Equal(t, liquidity.AmountSourceOverride, req.AmountSource)
	assert.Equal(t, merchantPayTo, req.PayTo)
}

func TestPlan_NoopWhenDestNativeCovers(t *testing.T) {
	inv := liquidity.Inventory{
		AgentAddress: agentAddr,
		Balances: []liquidity.Balance{{
			ChainCAIP2: baseCAIP2, Asset: baseUSDC,
			AmountAtomic: decimal.RequireFromString("2000000"), Location: liquidity.LocationNative,
		}},
	}
	p, err := liquidity.PlanLiquidity(baseRequired(), inv, nil)
require.NoError(t, err)
	assert.Equal(t, liquidity.ActionNoop, p.Action)
	assert.True(t, p.DryRun)
	assert.False(t, p.Executed)
	assert.True(t, p.InventoryUnverified)
}

func TestPlan_CircleGatewayWithdraw_AgentSelfOnly(t *testing.T) {
	inv := liquidity.Inventory{
		AgentAddress: agentAddr,
		Balances: []liquidity.Balance{{
			Asset: baseUSDC, AmountAtomic: decimal.RequireFromString("5000000"),
			Location: liquidity.LocationCircleGateway,
		}},
	}
	p, err := liquidity.PlanLiquidity(baseRequired(), inv, nil)
require.NoError(t, err)
	assert.Equal(t, liquidity.ActionCircleGatewayWithdraw, p.Action)
	require.Len(t, p.Steps, 1)
	assert.Equal(t, agentAddr, p.Steps[0].Recipient)
	assert.Equal(t, liquidity.RecipientRoleAgentSelf, p.Steps[0].RecipientRole)
	assert.NotEqual(t, merchantPayTo, p.Steps[0].Recipient)
	assert.NotEqual(t, platformMoR, p.Steps[0].Recipient)
}

func TestPlan_CircleGatewayDepositWithdraw(t *testing.T) {
	// Fragmented: Arc/Arb native only, merchant wants Base.
	inv := liquidity.Inventory{
		AgentAddress: agentAddr,
		Balances: []liquidity.Balance{{
			ChainCAIP2: arbCAIP2, Asset: baseUSDC,
			AmountAtomic: decimal.RequireFromString("3000000"), Location: liquidity.LocationNative,
		}},
	}
	p, err := liquidity.PlanLiquidity(baseRequired(), inv, nil)
require.NoError(t, err)
	assert.Equal(t, liquidity.ActionCircleGatewayDepositWithdraw, p.Action)
	require.Len(t, p.Steps, 2)
	for _, s := range p.Steps {
		assert.Equal(t, agentAddr, s.Recipient)
		assert.NotEqual(t, merchantPayTo, s.Recipient)
	}
}

func TestPlan_ShortfallOnly_FragmentedBaseAndArb(t *testing.T) {
	// Worked example: need 42 USDC on Base; have 20 Base + 30 Arb → move shortfall 22 only.
	req := baseRequired()
	req.AmountAtomic = decimal.RequireFromString("42000000")
	inv := liquidity.Inventory{
		AgentAddress: agentAddr,
		Balances: []liquidity.Balance{
			{ChainCAIP2: arbCAIP2, Asset: baseUSDC, AmountAtomic: decimal.RequireFromString("30000000"), Location: liquidity.LocationNative},
			{ChainCAIP2: baseCAIP2, Asset: baseUSDC, AmountAtomic: decimal.RequireFromString("20000000"), Location: liquidity.LocationNative},
		},
	}
	p, err := liquidity.PlanLiquidity(req, inv, nil)
require.NoError(t, err)
	assert.Equal(t, liquidity.ActionCircleGatewayDepositWithdraw, p.Action)
	require.Len(t, p.Steps, 2)
	assert.True(t, p.Steps[0].AmountAtomic.Equal(decimal.RequireFromString("22000000")))
	assert.Equal(t, arbCAIP2, p.Steps[0].FromChainCAIP2)
	assert.Equal(t, agentAddr, p.Steps[0].Recipient)
}

func TestPlan_Insufficient(t *testing.T) {
	inv := liquidity.Inventory{
		AgentAddress: agentAddr,
		Balances: []liquidity.Balance{{
			ChainCAIP2: baseCAIP2, Asset: baseUSDC,
			AmountAtomic: decimal.RequireFromString("1"), Location: liquidity.LocationNative,
		}},
	}
	p, err := liquidity.PlanLiquidity(baseRequired(), inv, nil)
require.NoError(t, err)
	assert.Equal(t, liquidity.ActionInsufficient, p.Action)
}

func TestPlan_CorridorUnsupported_Solana(t *testing.T) {
	req := baseRequired()
	req.ChainCAIP2 = solCAIP2
	req.Asset = "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v"
	p, err := liquidity.PlanLiquidity(req, liquidity.Inventory{AgentAddress: agentAddr}, nil)
require.NoError(t, err)
	assert.Equal(t, liquidity.ActionCorridorUnsupported, p.Action)
}

func TestPlan_BareGatewayLocationIgnored(t *testing.T) {
	inv := liquidity.Inventory{
		AgentAddress: agentAddr,
		Balances: []liquidity.Balance{{
			Asset: baseUSDC, AmountAtomic: decimal.RequireFromString("9000000"),
			Location: "gateway",
		}},
	}
	p, err := liquidity.PlanLiquidity(baseRequired(), inv, nil)
require.NoError(t, err)
	assert.Equal(t, liquidity.ActionInsufficient, p.Action)
}

func TestGuard_RefusePayToAsRecipient(t *testing.T) {
	g := &liquidity.Guard{}
	p := liquidity.Plan{
		Action: liquidity.ActionCircleGatewayWithdraw,
		Required: liquidity.Required{
			PayTo: merchantPayTo, ChainCAIP2: baseCAIP2, Asset: baseUSDC,
			AmountAtomic: decimal.RequireFromString("1"),
		},
		Steps: []liquidity.PlanStep{{
			Kind: liquidity.StepKindCircleGatewayWithdraw, Recipient: merchantPayTo,
			RecipientRole: liquidity.RecipientRoleAgentSelf,
		}},
	}
	err := g.CheckPlan(p)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pay_to")
}

func TestPlan_AtomicDecimal_NoRound2(t *testing.T) {
	req := baseRequired()
	req.AmountAtomic = decimal.RequireFromString("1234567")
	inv := liquidity.Inventory{
		AgentAddress: agentAddr,
		Balances: []liquidity.Balance{{
			ChainCAIP2: baseCAIP2, Asset: baseUSDC,
			AmountAtomic: decimal.RequireFromString("1234567"), Location: liquidity.LocationNative,
		}},
	}
	p, err := liquidity.PlanLiquidity(req, inv, nil)
require.NoError(t, err)
	assert.Equal(t, "1234567", p.Required.AmountAtomic.String())
}

func TestPlanToWire_DryStamps(t *testing.T) {
	inv := liquidity.Inventory{
		AgentAddress: agentAddr,
		Balances: []liquidity.Balance{{
			ChainCAIP2: baseCAIP2, Asset: baseUSDC,
			AmountAtomic: decimal.RequireFromString("2000000"), Location: liquidity.LocationNative,
		}},
	}
	p, err := liquidity.PlanLiquidity(baseRequired(), inv, nil)
require.NoError(t, err)
	w := liquidity.PlanToWire(p)
	assert.True(t, w.DryRun)
	assert.False(t, w.Executed)
	assert.True(t, w.InventoryAsserted)
	assert.True(t, w.InventoryUnverified)
}
