// Package execonchain is an optional testnet-only on-chain deposit executor.
package execonchain

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// ChainClient is the minimal RPC surface used by DepositExecutor (mockable).
type ChainClient interface {
	ChainID(ctx context.Context) (*big.Int, error)
	PendingNonceAt(ctx context.Context, account common.Address) (uint64, error)
	SuggestGasPrice(ctx context.Context) (*big.Int, error)
	EstimateGas(ctx context.Context, msg ethereum.CallMsg) (uint64, error)
	SendTransaction(ctx context.Context, tx *types.Transaction) error
	TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error)
	Close()
}

// ethClient wraps *ethclient.Client as ChainClient.
type ethClient struct {
	c *ethclient.Client
}

func (e *ethClient) ChainID(ctx context.Context) (*big.Int, error) {
	return e.c.ChainID(ctx)
}

func (e *ethClient) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	return e.c.PendingNonceAt(ctx, account)
}

func (e *ethClient) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	return e.c.SuggestGasPrice(ctx)
}

func (e *ethClient) EstimateGas(ctx context.Context, msg ethereum.CallMsg) (uint64, error) {
	return e.c.EstimateGas(ctx, msg)
}

func (e *ethClient) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	return e.c.SendTransaction(ctx, tx)
}

func (e *ethClient) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	return e.c.TransactionReceipt(ctx, txHash)
}

func (e *ethClient) Close() {
	e.c.Close()
}

// DefaultDial connects via go-ethereum ethclient.
func DefaultDial(ctx context.Context, rpcURL string) (ChainClient, error) {
	c, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, err
	}
	return &ethClient{c: c}, nil
}
