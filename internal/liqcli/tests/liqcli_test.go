package liqcli_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kaimo-no/usdc-liquidity-orchestrator/internal/liqcli"
	liqerr "github.com/kaimo-no/usdc-liquidity-orchestrator/pkg/errors"
	"github.com/kaimo-no/usdc-liquidity-orchestrator/pkg/types"
)

const planJSON = `{
  "required": {
    "protocol": "x402",
    "chain_caip2": "eip155:5042002",
    "asset": "0x3600000000000000000000000000000000000000",
    "amount_atomic": "42000000",
    "pay_to": "0xMerchantPayTo000000000000000000000001",
    "pay_to_role": "merchant"
  },
  "inventory": {
    "agent_address": "0xAgentSelf000000000000000000000000000001",
    "balances": [
      {"asset": "USDC", "amount_atomic": "100000000", "location": "circle_gateway"},
      {"chain_caip2": "eip155:5042002", "asset": "0x3600000000000000000000000000000000000000",
       "amount_atomic": "20000000", "location": "native"}
    ]
  },
  "orchestration": {
    "target_chain_caip2": "eip155:5042002",
    "prefer_rail": "circle_gateway"
  },
  "execute": false
}`

func TestMain_UsageExit2(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := liqcli.Main(nil, strings.NewReader(""), &stdout, &stderr)
	assert.Equal(t, 2, code)
	assert.Contains(t, stderr.String(), "Usage")
}

func TestMain_UnknownCommandExit2(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := liqcli.Main([]string{"nope"}, strings.NewReader(""), &stdout, &stderr)
	assert.Equal(t, 2, code)
}

func TestMain_Version(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := liqcli.Main([]string{"version"}, strings.NewReader(""), &stdout, &stderr)
	assert.Equal(t, 0, code)
	assert.Contains(t, stdout.String(), liqcli.Version)
}

func TestMain_DryPlan(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := liqcli.Main([]string{"plan", "-f", "-"}, strings.NewReader(planJSON), &stdout, &stderr)
	assert.Equal(t, 0, code)
	var resp types.PlanResponse
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &resp))
	assert.Nil(t, resp.Error)
	assert.True(t, resp.Plan.DryRun)
	assert.Equal(t, "circle_gateway_withdraw", resp.Plan.Action)
	assert.Equal(t, "agent_self", resp.Plan.Steps[0].RecipientRole)
}

func TestMain_ExecuteUnconfigured(t *testing.T) {
	t.Setenv("ENABLE_TESTNET_EXECUTE", "")
	var stdout, stderr bytes.Buffer
	code := liqcli.Main([]string{"plan", "--execute", "-f", "-"}, strings.NewReader(planJSON), &stdout, &stderr)
	assert.Equal(t, 1, code)
	var resp types.PlanResponse
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &resp))
	require.NotNil(t, resp.Error)
	assert.Equal(t, liqerr.CodeLiquidityRailUnavailable, resp.Error.Code)
	assert.True(t, resp.Plan.DryRun)
	// Privacy: stderr must not contain agent_address or balances.
	note := stderr.String()
	assert.NotContains(t, note, "0xAgentSelf")
	assert.NotContains(t, note, "100000000")
}

func TestMain_BodyOver1MiB(t *testing.T) {
	var stdout, stderr bytes.Buffer
	big := strings.Repeat("a", (1<<20)+10)
	code := liqcli.Main([]string{"plan", "-f", "-"}, strings.NewReader(big), &stdout, &stderr)
	assert.Equal(t, 1, code)
	assert.Contains(t, stderr.String(), "1 MiB")
}

func TestMain_Chains(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := liqcli.Main([]string{"chains"}, strings.NewReader(""), &stdout, &stderr)
	assert.Equal(t, 0, code)
	var body types.ChainsResponse
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &body))
	assert.NotEmpty(t, body.Chains)
}

func TestValidateInventoryArgs(t *testing.T) {
	tests := []struct {
		name      string
		flagAgent string
		envAgent  string
		wantErr   bool
	}{
		{"flag_ok", "0x" + strings.Repeat("ab", 20), "", false},
		{"env_ok", "", "0x" + strings.Repeat("cd", 20), false},
		{"missing", "", "", true},
		{"short", "0xabc", "", true},
		{"nonhex", "0x" + strings.Repeat("zz", 20), "", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := liqcli.ValidateInventoryArgs(tc.flagAgent, tc.envAgent)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotEmpty(t, cfg.AgentAddress)
		})
	}
}

func TestValidateInventoryRPCs(t *testing.T) {
	require.Error(t, liqcli.ValidateInventoryRPCs(nil))
	require.Error(t, liqcli.ValidateInventoryRPCs(map[string]string{}))
	require.NoError(t, liqcli.ValidateInventoryRPCs(map[string]string{"eip155:84532": "https://x"}))
}

func TestMain_InventoryMissingAgent_NoNetwork(t *testing.T) {
	t.Setenv("AGENT_ADDRESS", "")
	var stdout, stderr bytes.Buffer
	code := liqcli.Main([]string{"inventory"}, strings.NewReader(""), &stdout, &stderr)
	assert.Equal(t, 1, code)
	assert.Contains(t, stderr.String(), "agent address")
}

func TestMain_InventoryMissingRPC_NoNetwork(t *testing.T) {
	t.Setenv("AGENT_ADDRESS", "0x"+strings.Repeat("11", 20))
	t.Setenv("RPC_URL_BASE_SEPOLIA", "")
	t.Setenv("RPC_URL_ARBITRUM_SEPOLIA", "")
	t.Setenv("RPC_URL_ARC_TESTNET", "")
	t.Setenv("RPC_URLS_JSON", "")
	var stdout, stderr bytes.Buffer
	code := liqcli.Main([]string{"inventory"}, strings.NewReader(""), &stdout, &stderr)
	assert.Equal(t, 1, code)
	assert.Contains(t, stderr.String(), "RPC")
}
