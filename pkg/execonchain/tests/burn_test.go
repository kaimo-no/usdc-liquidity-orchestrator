package execonchain_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	liqerr "github.com/kaimo-no/usdc-liquidity-orchestrator/pkg/errors"
	"github.com/kaimo-no/usdc-liquidity-orchestrator/pkg/execonchain"
	"github.com/kaimo-no/usdc-liquidity-orchestrator/pkg/liquidity"
)

const (
	arbSepCAIP2 = "eip155:421614"
	arbSepUSDC  = "0x75faf114eafb1BDbe2F0316DF893fd58CE46AA4d"
)

func fixedSalt() (string, error) {
	return "0x" + strings.Repeat("ab", 32), nil
}

func depositWithdrawPlan(t *testing.T, agent string) liquidity.Plan {
	t.Helper()
	req := liquidity.Required{
		Protocol: "x402", ChainCAIP2: baseSepCAIP2, Asset: baseSepUSDC,
		PayTo:        "0xMerchant000000000000000000000000000001",
		AmountAtomic: decimal.RequireFromString("1000"),
		AmountSource: liquidity.AmountSourceProbe,
	}
	inv := liquidity.Inventory{
		AgentAddress: agent,
		Balances: []liquidity.Balance{{
			ChainCAIP2: arbSepCAIP2, Asset: arbSepUSDC,
			AmountAtomic: decimal.RequireFromString("1000"),
			Location:     liquidity.LocationNative,
		}},
	}
	sources := []liquidity.FundingSource{{
		ChainCAIP2: arbSepCAIP2, AmountAtomic: decimal.RequireFromString("1000"),
	}}
	p, err := liquidity.PlanPaymentFunding(req, inv, sources, nil)
	require.NoError(t, err)
	require.Equal(t, liquidity.ActionCircleGatewayDepositWithdraw, p.Action)
	return p
}

func TestDepositExecutor_DepositWithdraw_HappyPath(t *testing.T) {
	_, hex, agent := testKey(t)
	srcMock := newMock(421614)
	destMock := newMock(84532)
	// Use one dial that returns based on chain id verification path:
	// Dial is per-URL; map distinct URLs.
	rpcs := map[string]string{
		arbSepCAIP2:  "http://mock.local/arb",
		baseSepCAIP2: "http://mock.local/base",
	}
	var mu sync.Mutex
	var transferCalls int
	var lastBody []byte
	httpDo := func(req *http.Request) (*http.Response, error) {
		mu.Lock()
		transferCalls++
		mu.Unlock()
		assert.Equal(t, http.MethodPost, req.Method)
		assert.True(t, strings.HasSuffix(req.URL.Path, "/v1/transfer"))
		b, _ := io.ReadAll(req.Body)
		lastBody = b
		var items []map[string]any
		require.NoError(t, json.Unmarshal(b, &items))
		require.Len(t, items, 1)
		bi := items[0]["burnIntent"].(map[string]any)
		spec := bi["spec"].(map[string]any)
		// destinationRecipient must be agent (bytes32)
		rec, _ := spec["destinationRecipient"].(string)
		want := "0x" + strings.Repeat("0", 24) + strings.TrimPrefix(strings.ToLower(agent), "0x")
		assert.Equal(t, strings.ToLower(want), strings.ToLower(rec))
		// never pay_to
		assert.NotContains(t, strings.ToLower(string(b)), "merchant")

		raw, _ := json.Marshal(map[string]string{
			"attestation": "0x" + strings.Repeat("11", 64),
			"signature":   "0x" + strings.Repeat("22", 65),
		})
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewReader(raw)),
			Header:     make(http.Header),
		}, nil
	}
	dial := func(ctx context.Context, url string) (execonchain.ChainClient, error) {
		switch {
		case strings.Contains(url, "arb"):
			return srcMock, nil
		case strings.Contains(url, "base"):
			return destMock, nil
		default:
			return nil, assert.AnError
		}
	}
	ex, err := execonchain.NewDepositExecutor(execonchain.Config{
		PrivateKeyHex:      hex,
		RPCs:               rpcs,
		Dial:               dial,
		HTTPDo:             httpDo,
		GatewayAPI:         "https://gateway-api-testnet.circle.com",
		WaitTimeout:        time.Second,
		TransferRetries:    1,
		TransferRetryDelay: time.Millisecond,
		SaltFn:             fixedSalt,
	})
	require.NoError(t, err)

	plan := depositWithdrawPlan(t, agent)
	rcpt, err := ex.Execute(context.Background(), plan)
	require.NoError(t, err)
	// 2 deposit txs (approve+deposit) + 1 mint
	require.Len(t, rcpt.TxHashes, 3)
	assert.Equal(t, 2, srcMock.sends)
	assert.Equal(t, 1, destMock.sends)
	assert.Equal(t, 1, transferCalls)
	require.NotEmpty(t, lastBody)
	// mint goes to Gateway Minter
	require.NotEmpty(t, destMock.sent)
	assert.True(t, strings.EqualFold(destMock.sent[0].To().Hex(), liquidity.GatewayMinterTestnet))
}

func TestDepositExecutor_DestinationRecipientNotAgentRefused(t *testing.T) {
	_, hex, agent := testKey(t)
	mock := newMock(84532)
	ex, err := execonchain.NewDepositExecutor(execonchain.Config{
		PrivateKeyHex: hex,
		RPCs:          map[string]string{baseSepCAIP2: "http://mock.local"},
		Dial:          dialMock(mock),
		HTTPDo: func(req *http.Request) (*http.Response, error) {
			t.Fatal("transfer must not run")
			return nil, nil
		},
		TransferRetries: 1,
	})
	require.NoError(t, err)

	p := liquidity.Plan{
		Action: liquidity.ActionCircleGatewayWithdraw,
		Required: liquidity.Required{
			PayTo:      "0xMerchant000000000000000000000000000001",
			ChainCAIP2: baseSepCAIP2, Asset: baseSepUSDC,
			AmountAtomic: decimal.RequireFromString("1"),
		},
		Steps: []liquidity.PlanStep{{
			Kind:         liquidity.StepKindCircleGatewayWithdraw,
			ToChainCAIP2: baseSepCAIP2, Asset: baseSepUSDC,
			AmountAtomic:  decimal.RequireFromString("1"),
			Recipient:     "0x2222222222222222222222222222222222222222",
			RecipientRole: liquidity.RecipientRoleAgentSelf, // spoofed role; address wrong
		}},
	}
	p.BindAgent(agent)
	// Guard should catch recipient != agent first
	_, err = ex.Execute(context.Background(), p)
	require.Error(t, err)
	assert.Zero(t, mock.sends)
}

func TestDepositExecutor_AgentEqualsPayToRefusedByGuard(t *testing.T) {
	_, hex, agent := testKey(t)
	mock := newMock(84532)
	ex, err := execonchain.NewDepositExecutor(execonchain.Config{
		PrivateKeyHex: hex,
		RPCs:          map[string]string{baseSepCAIP2: "http://mock.local"},
		Dial:          dialMock(mock),
	})
	require.NoError(t, err)
	p := liquidity.Plan{
		Action: liquidity.ActionCircleGatewayWithdraw,
		Required: liquidity.Required{
			PayTo:      agent,
			ChainCAIP2: baseSepCAIP2, Asset: baseSepUSDC,
			AmountAtomic: decimal.RequireFromString("1"),
		},
		Steps: []liquidity.PlanStep{{
			Kind:         liquidity.StepKindCircleGatewayWithdraw,
			ToChainCAIP2: baseSepCAIP2, Asset: baseSepUSDC,
			AmountAtomic:  decimal.RequireFromString("1"),
			Recipient:     agent,
			RecipientRole: liquidity.RecipientRoleAgentSelf,
		}},
	}
	p.BindAgent(agent)
	_, err = ex.Execute(context.Background(), p)
	require.Error(t, err)
	assert.Zero(t, mock.sends)
}

func TestDepositExecutor_WithdrawOnly_BurnMint(t *testing.T) {
	_, hex, agent := testKey(t)
	mock := newMock(84532)
	var transferN, balancesN int
	httpDo := func(req *http.Request) (*http.Response, error) {
		if strings.HasSuffix(req.URL.Path, "/v1/balances") {
			balancesN++
			// 500 atomic = 0.0005 human USDC
			raw, _ := json.Marshal(map[string]any{
				"token": "USDC",
				"balances": []map[string]any{{
					"domain": 6, "depositor": agent, "balance": "0.000500",
				}},
			})
			return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(raw)), Header: make(http.Header)}, nil
		}
		transferN++
		raw, _ := json.Marshal(map[string]string{
			"attestation": "0xaa",
			"signature":   "0xbb",
		})
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(raw)), Header: make(http.Header)}, nil
	}
	lowFee := big.NewInt(10)
	ex, err := execonchain.NewDepositExecutor(execonchain.Config{
		PrivateKeyHex:      hex,
		RPCs:               map[string]string{baseSepCAIP2: "http://mock.local"},
		Dial:               dialMock(mock),
		HTTPDo:             httpDo,
		MaxFeeAtomic:       lowFee,
		WaitTimeout:        time.Second,
		TransferRetries:    1,
		TransferRetryDelay: time.Millisecond,
		SaltFn:             fixedSalt,
	})
	require.NoError(t, err)
	p := liquidity.Plan{
		Action: liquidity.ActionCircleGatewayWithdraw,
		Required: liquidity.Required{
			PayTo:      "0xMerchant000000000000000000000000000001",
			ChainCAIP2: baseSepCAIP2, Asset: baseSepUSDC,
			AmountAtomic: decimal.RequireFromString("500"),
		},
		Steps: []liquidity.PlanStep{{
			Kind:         liquidity.StepKindCircleGatewayWithdraw,
			ToChainCAIP2: baseSepCAIP2, Asset: baseSepUSDC,
			AmountAtomic:  decimal.RequireFromString("500"),
			Recipient:     agent,
			RecipientRole: liquidity.RecipientRoleAgentSelf,
		}},
	}
	p.BindAgent(agent)
	rcpt, err := ex.Execute(context.Background(), p)
	require.NoError(t, err)
	require.Len(t, rcpt.TxHashes, 1)
	assert.Equal(t, 1, balancesN)
	assert.Equal(t, 1, transferN)
	assert.Equal(t, 1, mock.sends)
}

func TestDepositExecutor_WithdrawOnly_MultiDomainFromBalances(t *testing.T) {
	// 1 USDC on Arb is below default fee floor (~2.01 USDC); only Base 5 USDC is burnable.
	_, hex, agent := testKey(t)
	arcMock := newMock(5042002)
	var transferBodies [][]byte
	httpDo := func(req *http.Request) (*http.Response, error) {
		if strings.HasSuffix(req.URL.Path, "/v1/balances") {
			raw, _ := json.Marshal(map[string]any{
				"token": "USDC",
				"balances": []map[string]any{
					{"domain": 3, "depositor": agent, "balance": "1.000000"},
					{"domain": 6, "depositor": agent, "balance": "5.000000"},
					{"domain": 26, "depositor": agent, "balance": "0"},
				},
			})
			return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(raw)), Header: make(http.Header)}, nil
		}
		b, _ := io.ReadAll(req.Body)
		transferBodies = append(transferBodies, b)
		raw, _ := json.Marshal(map[string]string{
			"attestation": "0xaa",
			"signature":   "0xbb",
		})
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(raw)), Header: make(http.Header)}, nil
	}
	ex, err := execonchain.NewDepositExecutor(execonchain.Config{
		PrivateKeyHex:      hex,
		RPCs:               map[string]string{"eip155:5042002": "http://mock.local/arc"},
		Dial:               dialMock(arcMock),
		HTTPDo:             httpDo,
		WaitTimeout:        time.Second,
		TransferRetries:    1,
		TransferRetryDelay: time.Millisecond,
		SaltFn:             fixedSalt,
	})
	require.NoError(t, err)
	const arcCAIP2 = "eip155:5042002"
	const arcUSDC = "0x3600000000000000000000000000000000000000"
	p := liquidity.Plan{
		Action: liquidity.ActionCircleGatewayWithdraw,
		Required: liquidity.Required{
			PayTo:      "0xMerchant000000000000000000000000000001",
			ChainCAIP2: arcCAIP2, Asset: arcUSDC,
			AmountAtomic: decimal.RequireFromString("5000000"),
		},
		Steps: []liquidity.PlanStep{{
			Kind:         liquidity.StepKindCircleGatewayWithdraw,
			ToChainCAIP2: arcCAIP2, Asset: arcUSDC,
			AmountAtomic:  decimal.RequireFromString("5000000"),
			Recipient:     agent,
			RecipientRole: liquidity.RecipientRoleAgentSelf,
		}},
	}
	p.BindAgent(agent)
	rcpt, err := ex.Execute(context.Background(), p)
	require.NoError(t, err)
	require.Len(t, rcpt.TxHashes, 1)
	require.Len(t, transferBodies, 1)
	var items []map[string]any
	require.NoError(t, json.Unmarshal(transferBodies[0], &items))
	spec := items[0]["burnIntent"].(map[string]any)["spec"].(map[string]any)
	assert.Equal(t, float64(6), spec["sourceDomain"])
	assert.Equal(t, "5000000", spec["value"])
	assert.Equal(t, 1, arcMock.sends)
}

func TestDepositExecutor_WithdrawOnly_SixUSDCFailsFeeFloorDust(t *testing.T) {
	_, hex, agent := testKey(t)
	httpDo := func(req *http.Request) (*http.Response, error) {
		if strings.HasSuffix(req.URL.Path, "/v1/balances") {
			raw, _ := json.Marshal(map[string]any{
				"balances": []map[string]any{
					{"domain": 3, "balance": "1.000000"},
					{"domain": 6, "balance": "5.000000"},
				},
			})
			return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(raw)), Header: make(http.Header)}, nil
		}
		t.Fatal("should not transfer")
		return nil, nil
	}
	ex, err := execonchain.NewDepositExecutor(execonchain.Config{
		PrivateKeyHex: hex,
		RPCs:          map[string]string{"eip155:5042002": "http://mock.local"},
		Dial:          dialMock(newMock(5042002)),
		HTTPDo:        httpDo,
	})
	require.NoError(t, err)
	const arcCAIP2 = "eip155:5042002"
	const arcUSDC = "0x3600000000000000000000000000000000000000"
	p := liquidity.Plan{
		Action: liquidity.ActionCircleGatewayWithdraw,
		Required: liquidity.Required{
			PayTo:      "0xMerchant000000000000000000000000000001",
			ChainCAIP2: arcCAIP2, Asset: arcUSDC,
			AmountAtomic: decimal.RequireFromString("6000000"),
		},
		Steps: []liquidity.PlanStep{{
			Kind: liquidity.StepKindCircleGatewayWithdraw, ToChainCAIP2: arcCAIP2, Asset: arcUSDC,
			AmountAtomic: decimal.RequireFromString("6000000"), Recipient: agent, RecipientRole: liquidity.RecipientRoleAgentSelf,
		}},
	}
	p.BindAgent(agent)
	_, err = ex.Execute(context.Background(), p)
	require.Error(t, err)
	assert.Equal(t, liqerr.CodeInsufficientLiquidity, liqerr.CodeOf(err))
	assert.Contains(t, err.Error(), "fee floor")
}

func TestDepositExecutor_TransferRetryThenSuccess(t *testing.T) {
	_, hex, agent := testKey(t)
	mock := newMock(84532)
	var transferN int
	httpDo := func(req *http.Request) (*http.Response, error) {
		if strings.HasSuffix(req.URL.Path, "/v1/balances") {
			raw, _ := json.Marshal(map[string]any{
				"balances": []map[string]any{{"domain": 6, "balance": "5.000000"}},
			})
			return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(raw)), Header: make(http.Header)}, nil
		}
		transferN++
		if transferN < 3 {
			return &http.Response{
				StatusCode: 400,
				Body:       io.NopCloser(strings.NewReader(`{"error":"not finalized leak"}`)),
				Header:     make(http.Header),
			}, nil
		}
		raw, _ := json.Marshal(map[string]string{"attestation": "0x11", "signature": "0x22"})
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(raw)), Header: make(http.Header)}, nil
	}
	ex, err := execonchain.NewDepositExecutor(execonchain.Config{
		PrivateKeyHex:      hex,
		RPCs:               map[string]string{baseSepCAIP2: "http://mock.local"},
		Dial:               dialMock(mock),
		HTTPDo:             httpDo,
		MaxFeeAtomic:       big.NewInt(100),
		WaitTimeout:        time.Second,
		TransferRetries:    5,
		TransferRetryDelay: time.Millisecond,
		SaltFn:             fixedSalt,
	})
	require.NoError(t, err)
	p := liquidity.Plan{
		Action: liquidity.ActionCircleGatewayWithdraw,
		Required: liquidity.Required{
			PayTo:      "0xMerchant000000000000000000000000000001",
			ChainCAIP2: baseSepCAIP2, Asset: baseSepUSDC,
			AmountAtomic: decimal.RequireFromString("1000000"),
		},
		Steps: []liquidity.PlanStep{{
			Kind: liquidity.StepKindCircleGatewayWithdraw, ToChainCAIP2: baseSepCAIP2, Asset: baseSepUSDC,
			AmountAtomic: decimal.RequireFromString("1000000"), Recipient: agent, RecipientRole: liquidity.RecipientRoleAgentSelf,
		}},
	}
	p.BindAgent(agent)
	_, err = ex.Execute(context.Background(), p)
	require.NoError(t, err)
	assert.Equal(t, 3, transferN)
}

func TestDepositExecutor_TransferErrorSanitized(t *testing.T) {
	_, hex, agent := testKey(t)
	mock := newMock(84532)
	httpDo := func(req *http.Request) (*http.Response, error) {
		if strings.HasSuffix(req.URL.Path, "/v1/balances") {
			raw, _ := json.Marshal(map[string]any{
				"balances": []map[string]any{{"domain": 6, "balance": "5.000000"}},
			})
			return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(raw)), Header: make(http.Header)}, nil
		}
		return &http.Response{
			StatusCode: 500,
			Body:       io.NopCloser(strings.NewReader(`{"success":false,"message":"server boom","detail":"raw-secret-body"}`)),
			Header:     make(http.Header),
		}, nil
	}
	ex, err := execonchain.NewDepositExecutor(execonchain.Config{
		PrivateKeyHex:      hex,
		RPCs:               map[string]string{baseSepCAIP2: "http://mock.local"},
		Dial:               dialMock(mock),
		HTTPDo:             httpDo,
		MaxFeeAtomic:       big.NewInt(100),
		TransferRetries:    1,
		TransferRetryDelay: time.Millisecond,
		SaltFn:             fixedSalt,
	})
	require.NoError(t, err)
	p := liquidity.Plan{
		Action: liquidity.ActionCircleGatewayWithdraw,
		Required: liquidity.Required{
			PayTo:      "0xMerchant000000000000000000000000000001",
			ChainCAIP2: baseSepCAIP2, Asset: baseSepUSDC,
			AmountAtomic: decimal.RequireFromString("1000000"),
		},
		Steps: []liquidity.PlanStep{{
			Kind: liquidity.StepKindCircleGatewayWithdraw, ToChainCAIP2: baseSepCAIP2, Asset: baseSepUSDC,
			AmountAtomic: decimal.RequireFromString("1000000"), Recipient: agent, RecipientRole: liquidity.RecipientRoleAgentSelf,
		}},
	}
	p.BindAgent(agent)
	_, err = ex.Execute(context.Background(), p)
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "raw-secret-body")
	assert.Contains(t, err.Error(), "transfer HTTP 500")
	assert.Contains(t, err.Error(), "server boom")
}

func TestDepositExecutor_MainnetStillRefused(t *testing.T) {
	_, hex, _ := testKey(t)
	_, err := execonchain.NewDepositExecutor(execonchain.Config{
		PrivateKeyHex: hex,
		RPCs:          map[string]string{baseMainnet: "https://example.invalid"},
	})
	require.Error(t, err)
	assert.Equal(t, liqerr.CodeLiquidityRailUnavailable, liqerr.CodeOf(err))
}

func TestDepositExecutor_DepositWithdraw_PrepareMismatch(t *testing.T) {
	_, hex, agent := testKey(t)
	srcMock := newMock(421614)
	destMock := newMock(84532)
	ex, err := execonchain.NewDepositExecutor(execonchain.Config{
		PrivateKeyHex: hex,
		RPCs: map[string]string{
			arbSepCAIP2: "http://mock.local/arb", baseSepCAIP2: "http://mock.local/base",
		},
		Dial: func(ctx context.Context, url string) (execonchain.ChainClient, error) {
			if strings.Contains(url, "arb") {
				return srcMock, nil
			}
			return destMock, nil
		},
		HTTPDo: func(req *http.Request) (*http.Response, error) {
			t.Fatal("no transfer after prepare mismatch")
			return nil, nil
		},
		TransferRetries: 1,
	})
	require.NoError(t, err)
	plan := depositWithdrawPlan(t, agent)
	require.NotEmpty(t, plan.Steps[0].PrepareCalls)
	plan.Steps[0].PrepareCalls[0].Data = "0xdeadbeef"
	_, err = ex.Execute(context.Background(), plan)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "prepare_calls mismatch")
	assert.Zero(t, srcMock.sends)
}
