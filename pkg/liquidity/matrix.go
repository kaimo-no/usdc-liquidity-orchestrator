package liquidity

import "strings"

// Corridor hints only (no live Circle SDK). EVM-first; Solana = unsupported.

func chainIsEVM(caip2 string) bool {
	return strings.HasPrefix(strings.TrimSpace(caip2), "eip155:")
}

func chainIsSolana(caip2 string) bool {
	return strings.HasPrefix(strings.TrimSpace(caip2), "solana:")
}

// corridorEligible reports whether circle_gateway / cctp plans may target dest.
// Registry-driven; unknown EVM dests are unsupported this cut.
func corridorEligible(destCAIP2 string) (circleGateway, cctp bool) {
	dest := strings.TrimSpace(destCAIP2)
	if chainIsSolana(dest) || !chainIsEVM(dest) {
		return false, false
	}
	info, ok := LookupChain(dest)
	if !ok {
		return false, false
	}
	return info.GatewayOK, info.CCTPOK
}
