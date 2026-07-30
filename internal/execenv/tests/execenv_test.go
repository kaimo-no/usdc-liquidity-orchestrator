package execenv_test

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kaimo-no/usdc-liquidity-orchestrator/internal/execenv"
	"github.com/kaimo-no/usdc-liquidity-orchestrator/pkg/liquidity"
)

func TestIsLoopbackListen(t *testing.T) {
	tests := []struct {
		addr string
		want bool
	}{
		{"127.0.0.1:8088", true},
		{"localhost:8088", true},
		{"[::1]:8088", true},
		{":8088", false},
		{"0.0.0.0:8088", false},
		{"192.168.1.1:8088", false},
	}
	for _, tc := range tests {
		t.Run(tc.addr, func(t *testing.T) {
			assert.Equal(t, tc.want, execenv.IsLoopbackListen(tc.addr))
		})
	}
}

func TestBuildExecutor_EnableOff_Unconfigured(t *testing.T) {
	t.Setenv("ENABLE_TESTNET_EXECUTE", "")
	t.Setenv("AGENT_PRIVATE_KEY", "")
	ex, err := execenv.BuildExecutor(execenv.Options{})
	require.NoError(t, err)
	_, ok := ex.(liquidity.UnconfiguredExecutor)
	assert.True(t, ok, "ENABLE≠1 must return UnconfiguredExecutor")
}

func TestBuildExecutor_EnableOn_IncompleteKey(t *testing.T) {
	t.Setenv("ENABLE_TESTNET_EXECUTE", "1")
	t.Setenv("AGENT_PRIVATE_KEY", "")
	t.Setenv("RPC_URL_BASE_SEPOLIA", "https://example.invalid")
	ex, err := execenv.BuildExecutor(execenv.Options{})
	assert.Nil(t, ex)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "AGENT_PRIVATE_KEY")
}

func TestBuildExecutor_EnableOn_NoRPC(t *testing.T) {
	t.Setenv("ENABLE_TESTNET_EXECUTE", "1")
	t.Setenv("AGENT_PRIVATE_KEY", "0x"+hex64("ab"))
	t.Setenv("RPC_URL_BASE_SEPOLIA", "")
	t.Setenv("RPC_URL_ARBITRUM_SEPOLIA", "")
	t.Setenv("RPC_URL_ARC_TESTNET", "")
	t.Setenv("RPC_URLS_JSON", "")
	// Best-effort clear alternate CAIP-2 env keys so residual process env does not supply RPCs.
	for _, e := range os.Environ() {
		eq := strings.IndexByte(e, '=')
		if eq <= 0 {
			continue
		}
		name := e[:eq]
		if strings.HasPrefix(name, "RPC_URL_eip155_") || strings.HasPrefix(name, "RPC_URL_solana_") {
			t.Setenv(name, "")
		}
	}
	ex, err := execenv.BuildExecutor(execenv.Options{})
	assert.Nil(t, ex)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "RPC")
}

func TestBuildExecutor_EnableOn_NonLoopbackHTTP(t *testing.T) {
	t.Setenv("ENABLE_TESTNET_EXECUTE", "1")
	t.Setenv("AGENT_PRIVATE_KEY", "0x"+hex64("cd"))
	t.Setenv("RPC_URL_BASE_SEPOLIA", "https://example.invalid")
	ex, err := execenv.BuildExecutor(execenv.Options{RequireLoopbackListen: ":8088"})
	assert.Nil(t, ex)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "loopback")
}

func TestBuildExecutor_EnableOn_CLINoLoopbackRequired(t *testing.T) {
	// CLI passes empty RequireLoopbackListen; still needs valid key + RPC.
	// Invalid key shape fails at DepositExecutor construction (sanitized path).
	t.Setenv("ENABLE_TESTNET_EXECUTE", "1")
	t.Setenv("AGENT_PRIVATE_KEY", "not-a-key")
	t.Setenv("RPC_URL_BASE_SEPOLIA", "https://example.invalid")
	ex, err := execenv.BuildExecutor(execenv.Options{})
	assert.Nil(t, ex)
	require.Error(t, err)
}

func TestGuardFromEnv_InvalidMax(t *testing.T) {
	t.Setenv("MAX_AMOUNT_ATOMIC", "not-a-number")
	_, err := execenv.GuardFromEnv()
	require.Error(t, err)
}

func TestGuardFromEnv_ValidMax(t *testing.T) {
	t.Setenv("MAX_AMOUNT_ATOMIC", "1000")
	g, err := execenv.GuardFromEnv()
	require.NoError(t, err)
	require.NotNil(t, g)
	assert.True(t, g.MaxAmountAtomic.IsPositive())
}

func hex64(seed string) string {
	// 64 hex chars for a plausible ECDSA private key shape (may still fail crypto parse).
	out := ""
	for len(out) < 64 {
		out += seed
	}
	return out[:64]
}
