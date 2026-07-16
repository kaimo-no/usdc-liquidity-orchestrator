package liquidity

import "strings"

// Corridor hints only (no live Circle SDK). EVM-first; Solana = unsupported.

func chainIsEVM(caip2 string) bool {
	return strings.HasPrefix(strings.TrimSpace(caip2), "eip155:")
}

func chainIsSolana(caip2 string) bool {
	return strings.HasPrefix(strings.TrimSpace(caip2), "solana:")
}

// knownEVMUSDCDest is the set of dest chains where circle_gateway / cctp plans
// are eligible (hints only; execute remains unconfigured by default).
var knownEVMUSDCDest = map[string]struct{}{
	"eip155:8453":   {}, // base
	"eip155:84532":  {}, // base-sepolia
	"eip155:42161":  {}, // arbitrum
	"eip155:421614": {}, // arbitrum-sepolia
}

func corridorEligible(destCAIP2 string) (circleGateway, cctp bool) {
	dest := strings.TrimSpace(destCAIP2)
	if chainIsSolana(dest) || !chainIsEVM(dest) {
		return false, false
	}
	if _, ok := knownEVMUSDCDest[dest]; !ok {
		return false, false
	}
	return true, true
}
