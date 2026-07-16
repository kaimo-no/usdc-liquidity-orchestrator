package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kaimo-no/usdc-liquidity-orchestrator/internal/httpserver"
	liqerr "github.com/kaimo-no/usdc-liquidity-orchestrator/pkg/errors"
	"github.com/kaimo-no/usdc-liquidity-orchestrator/pkg/types"
)

const (
	arcCAIP2      = "eip155:5042002"
	arcUSDC       = "0x3600000000000000000000000000000000000000"
	merchantPayTo = "0xMerchantPayTo000000000000000000000001"
	agentAddr     = "0xAgentSelf000000000000000000000000000001"
)

func TestGETChains_IncludesArc(t *testing.T) {
	mux := httpserver.NewMux()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/chains", nil)
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var body types.ChainsResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	var found bool
	for _, c := range body.Chains {
		if c.CAIP2 == arcCAIP2 {
			found = true
			assert.Equal(t, 26, c.GatewayDomain)
			assert.True(t, c.GatewayOK)
			assert.Equal(t, arcUSDC, c.USDC)
		}
	}
	assert.True(t, found, "Arc Testnet eip155:5042002 must be registered")
}

func TestPOSTPlan_DryStampsForced(t *testing.T) {
	mux := httpserver.NewMux()
	payload := planBody(false, merchantPayTo, true)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/plan", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp types.PlanResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Nil(t, resp.Error)
	assert.True(t, resp.Plan.DryRun)
	assert.False(t, resp.Plan.Executed)
	assert.True(t, resp.Plan.InventoryAsserted)
	assert.True(t, resp.Plan.InventoryUnverified)
	assert.Equal(t, "circle_gateway_withdraw", resp.Plan.Action)
}

func TestPOSTPlan_ExecuteTrue_ErrorPlusPlan(t *testing.T) {
	mux := httpserver.NewMux()
	payload := planBody(true, merchantPayTo, true)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/plan", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	var resp types.PlanResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.NotNil(t, resp.Error)
	assert.Equal(t, liqerr.CodeLiquidityRailUnavailable, resp.Error.Code)
	// Dry plan still returned for inspection.
	assert.Equal(t, "circle_gateway_withdraw", resp.Plan.Action)
	assert.True(t, resp.Plan.DryRun)
	assert.False(t, resp.Plan.Executed)
	require.NotEmpty(t, resp.Plan.Steps)
}

func TestPOSTPlan_EmptyPayTo_FailClosed(t *testing.T) {
	mux := httpserver.NewMux()
	payload := planBody(false, "", true)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/plan", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	var resp types.PlanResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.NotNil(t, resp.Error)
	assert.Equal(t, liqerr.CodeInsufficientLiquidity, resp.Error.Code)
	assert.Empty(t, resp.Plan.Action)
}

func TestPOSTPlan_ArcGatewayWithdraw(t *testing.T) {
	mux := httpserver.NewMux()
	payload := planBody(false, merchantPayTo, true)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/plan", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp types.PlanResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Nil(t, resp.Error)
	assert.Equal(t, "circle_gateway_withdraw", resp.Plan.Action)
	require.NotEmpty(t, resp.Plan.Steps)
	assert.Equal(t, "circle_gateway_withdraw", resp.Plan.Steps[0].Kind)
	assert.Equal(t, agentAddr, resp.Plan.Steps[0].Recipient)
	assert.Equal(t, "agent_self", resp.Plan.Steps[0].RecipientRole)
	assert.Equal(t, arcCAIP2, resp.Plan.Steps[0].ToChainCAIP2)
	// Fee must not appear as a step.
	for _, s := range resp.Plan.Steps {
		assert.NotEqual(t, "orchestrator_fee", s.Kind)
	}
}

func planBody(execute bool, payTo string, withGateway bool) []byte {
	balances := []map[string]string{}
	if withGateway {
		balances = append(balances, map[string]string{
			"asset":         "USDC",
			"amount_atomic": "100000000",
			"location":      "circle_gateway",
		})
	}
	balances = append(balances, map[string]string{
		"chain_caip2":   arcCAIP2,
		"asset":         arcUSDC,
		"amount_atomic": "20000000",
		"location":      "native",
	})
	body := map[string]any{
		"required": map[string]string{
			"protocol":      "x402",
			"chain_caip2":   arcCAIP2,
			"asset":         arcUSDC,
			"amount_atomic": "42000000",
			"pay_to":        payTo,
			"pay_to_role":   "merchant",
		},
		"inventory": map[string]any{
			"agent_address": agentAddr,
			"balances":      balances,
		},
		"orchestration": map[string]any{
			"target_chain_caip2": arcCAIP2,
			"prefer_rail":        "circle_gateway",
		},
		"execute": execute,
	}
	b, _ := json.Marshal(body)
	return b
}
