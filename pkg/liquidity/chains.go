package liquidity

import "strings"

// ChainInfo describes a registered EVM corridor for plan / future execute.
// GatewayDomain is Circle's domain id (same numbering as CCTP/Gateway).
type ChainInfo struct {
	CAIP2         string
	Name          string
	GatewayDomain int
	USDC          string
	GatewayOK     bool
	CCTPOK        bool
	Testnet       bool
}

// Registered chains (hackathon + existing demos). Expand as corridors land.
var chainRegistry = []ChainInfo{
	{
		CAIP2: "eip155:5042002", Name: "arc-testnet", GatewayDomain: 26,
		USDC: "0x3600000000000000000000000000000000000000", GatewayOK: true, CCTPOK: true, Testnet: true,
	},
	{
		CAIP2: "eip155:84532", Name: "base-sepolia", GatewayDomain: 6,
		USDC: "0x036CbD53842c5426634e7929541eC2318f3dCF7e", GatewayOK: true, CCTPOK: true, Testnet: true,
	},
	{
		CAIP2: "eip155:421614", Name: "arbitrum-sepolia", GatewayDomain: 3,
		USDC: "0x75faf114eafb1BDbe2F0316DF893fd58CE46AA4d", GatewayOK: true, CCTPOK: true, Testnet: true,
	},
	{
		CAIP2: "eip155:8453", Name: "base", GatewayDomain: 6,
		USDC: "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913", GatewayOK: true, CCTPOK: true, Testnet: false,
	},
	{
		CAIP2: "eip155:42161", Name: "arbitrum", GatewayDomain: 3,
		USDC: "0xaf88d065e77c8cC2239327C5EDb3A432268e5831", GatewayOK: true, CCTPOK: true, Testnet: false,
	},
}

// ListChains returns a copy of the registry for discovery (e.g. GET /v1/chains).
func ListChains() []ChainInfo {
	out := make([]ChainInfo, len(chainRegistry))
	copy(out, chainRegistry)
	return out
}

// LookupChain finds a registered chain by CAIP-2 (case-insensitive).
func LookupChain(caip2 string) (ChainInfo, bool) {
	key := strings.TrimSpace(caip2)
	for _, c := range chainRegistry {
		if strings.EqualFold(c.CAIP2, key) {
			return c, true
		}
	}
	return ChainInfo{}, false
}

// DefaultUSDC returns the registered native USDC contract for a chain, if any.
func DefaultUSDC(caip2 string) (string, bool) {
	c, ok := LookupChain(caip2)
	if !ok || c.USDC == "" {
		return "", false
	}
	return c.USDC, true
}

// GatewayWalletAddress returns the Circle Gateway Wallet contract for a registered
// GatewayOK chain (testnet vs mainnet constant).
func GatewayWalletAddress(caip2 string) (addr string, ok bool) {
	c, ok := LookupChain(caip2)
	if !ok || !c.GatewayOK {
		return "", false
	}
	if c.Testnet {
		return GatewayWalletTestnet, true
	}
	return GatewayWalletMainnet, true
}

// IsKnownUSDCAsset reports whether asset matches any registered chain USDC
// or the case-insensitive token symbol "USDC" (gateway inventory convenience).
func IsKnownUSDCAsset(asset string) bool {
	a := strings.TrimSpace(asset)
	if a == "" {
		return false
	}
	if strings.EqualFold(a, "USDC") {
		return true
	}
	for _, c := range chainRegistry {
		if assetEqual(a, c.USDC, c.CAIP2) {
			return true
		}
	}
	return false
}
