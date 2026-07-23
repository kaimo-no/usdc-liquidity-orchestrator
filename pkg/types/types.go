// Package types holds agent-facing wire shapes for liquidity plan I/O.
package types

// Required is the merchant-claim dest-chain need (untrusted pay_to metadata).
// Agents fund agent_self only during prepare; pay_to is for later merchant settle.
type Required struct {
	Protocol     string `json:"protocol,omitempty"`
	ChainCAIP2   string `json:"chain_caip2,omitempty"`
	Asset        string `json:"asset,omitempty"`
	AmountAtomic string `json:"amount_atomic,omitempty"`
	AmountHuman  string `json:"amount_human,omitempty"`
	PayTo        string `json:"pay_to"`
	PayToRole    string `json:"pay_to_role"`
	Source       string `json:"source,omitempty"`
	Incomplete   bool   `json:"incomplete,omitempty"`
}

// Orchestration is optional agent setup: target + allowed sources + rail preference.
type Orchestration struct {
	TargetChainCAIP2   string   `json:"target_chain_caip2,omitempty"`
	SourceChainCAIP2s  []string `json:"source_chain_caip2s,omitempty"`
	AllowCircleGateway *bool    `json:"allow_circle_gateway,omitempty"` // nil = true
	PreferRail         string   `json:"prefer_rail,omitempty"`          // auto | circle_gateway | cctp_fast
}

// Fee is the optional kaimo orchestration fee on a plan (not a fund-rail recipient).
type Fee struct {
	Bps           int64  `json:"bps,omitempty"`
	AmountAtomic  string `json:"amount_atomic,omitempty"`
	Recipient     string `json:"recipient,omitempty"`
	RecipientRole string `json:"recipient_role,omitempty"` // orchestrator
	SettleVia     string `json:"settle_via,omitempty"`     // x402
	ChainCAIP2    string `json:"chain_caip2,omitempty"`
	Asset         string `json:"asset,omitempty"`
}

// Plan is the dry/execute plan envelope returned to agents.
type Plan struct {
	Action              string     `json:"action"`
	Required            *Required  `json:"required,omitempty"`
	Steps               []PlanStep `json:"steps,omitempty"`
	Reason              string     `json:"reason,omitempty"`
	RecipientRole       string     `json:"recipient_role,omitempty"`
	InventoryAsserted   bool       `json:"inventory_asserted"`
	InventoryUnverified bool       `json:"inventory_unverified"`
	Executed            bool       `json:"executed"`
	DryRun              bool       `json:"dry_run"`
	AmountSource        string     `json:"amount_source,omitempty"`
	Fee                 *Fee       `json:"fee,omitempty"`
}

// PrepareCall is an unsigned EVM call for agent-side signing (advisory; server-generated).
type PrepareCall struct {
	ChainCAIP2  string `json:"chain_caip2,omitempty"`
	To          string `json:"to,omitempty"`
	Data        string `json:"data,omitempty"`
	Value       string `json:"value,omitempty"`
	Method      string `json:"method,omitempty"`
	Description string `json:"description,omitempty"`
}

// PlanStep is one fund-movement or note. Fund steps use recipient_role=agent_self.
type PlanStep struct {
	Kind           string        `json:"kind"`
	FromChainCAIP2 string        `json:"from_chain_caip2,omitempty"`
	ToChainCAIP2   string        `json:"to_chain_caip2,omitempty"`
	Asset          string        `json:"asset,omitempty"`
	AmountAtomic   string        `json:"amount_atomic,omitempty"`
	Recipient      string        `json:"recipient,omitempty"`
	RecipientRole  string        `json:"recipient_role,omitempty"`
	PrepareCalls   []PrepareCall `json:"prepare_calls,omitempty"`
}

// Inventory is client-asserted balances (never server-custodied product funds).
type Inventory struct {
	AgentAddress string    `json:"agent_address"`
	Balances     []Balance `json:"balances"`
}

// Balance is one inventory row (native wallet or circle_gateway unified balance).
type Balance struct {
	ChainCAIP2   string `json:"chain_caip2,omitempty"`
	Asset        string `json:"asset,omitempty"`
	AmountAtomic string `json:"amount_atomic"`
	Location     string `json:"location,omitempty"` // native | circle_gateway
}

// PlanRequest is the HTTP/MCP plan input.
type PlanRequest struct {
	Required       Required       `json:"required"`
	Inventory      Inventory      `json:"inventory"`
	Orchestration  *Orchestration `json:"orchestration,omitempty"`
	AmountOverride string         `json:"amount_override,omitempty"`
	FeeBps         int64          `json:"fee_bps,omitempty"`
	FeeRecipient   string         `json:"fee_recipient,omitempty"`
	Execute        bool           `json:"execute,omitempty"`
}

// ConsolidateRequest is POST /v1/consolidate input (no merchant claim / fee).
type ConsolidateRequest struct {
	Inventory     Inventory      `json:"inventory"`
	Orchestration *Orchestration `json:"orchestration,omitempty"`
	Execute       bool           `json:"execute,omitempty"`
}

// ExecuteReceipt is optional on-chain execute evidence (tx hashes only; no notes).
type ExecuteReceipt struct {
	TxHashes []string `json:"tx_hashes,omitempty"`
}

// PlanResponse is the HTTP plan output.
type PlanResponse struct {
	Plan    Plan            `json:"plan"`
	Receipt *ExecuteReceipt `json:"receipt,omitempty"`
	Error   *APIError       `json:"error,omitempty"`
}

// APIError is the JSON error body.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ChainInfo is a registry row for discovery (GET /v1/chains).
type ChainInfo struct {
	CAIP2         string `json:"chain_caip2"`
	Name          string `json:"name"`
	GatewayDomain int    `json:"gateway_domain"`
	USDC          string `json:"usdc"`
	GatewayOK     bool   `json:"gateway_ok"`
	CCTPOK        bool   `json:"cctp_ok"`
	Testnet       bool   `json:"testnet"`
	GatewayWallet string `json:"gateway_wallet,omitempty"`
}

// ChainsResponse is the GET /v1/chains body.
type ChainsResponse struct {
	Chains []ChainInfo `json:"chains"`
}
