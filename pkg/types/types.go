// Package types holds agent-facing wire shapes for liquidity plan I/O.
package types

// Required is the dest-chain land amount (Phase B) or scenario stamp (Phase A).
// Agents fund agent_self only; residual pay_to is optional claim metadata (never fund dest).
// amount_atomic is always the real on-chain amount. Optional amount_logical_atomic +
// scale_factor stamp scenario scale (logical / scale → real).
type Required struct {
	Protocol            string `json:"protocol,omitempty"`
	ChainCAIP2          string `json:"chain_caip2,omitempty"`
	Asset               string `json:"asset,omitempty"`
	AmountAtomic        string `json:"amount_atomic,omitempty"`
	AmountLogicalAtomic string `json:"amount_logical_atomic,omitempty"`
	ScaleFactor         int64  `json:"scale_factor,omitempty"`
	PayTo               string `json:"pay_to,omitempty"`
	PayToRole           string `json:"pay_to_role,omitempty"`
	Source              string `json:"source,omitempty"`
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
// amount_atomic is always real; optional amount_logical_atomic + scale_factor for scenario stamps.
type PlanStep struct {
	Kind                string        `json:"kind"`
	FromChainCAIP2      string        `json:"from_chain_caip2,omitempty"`
	ToChainCAIP2        string        `json:"to_chain_caip2,omitempty"`
	Asset               string        `json:"asset,omitempty"`
	AmountAtomic        string        `json:"amount_atomic,omitempty"`
	AmountLogicalAtomic string        `json:"amount_logical_atomic,omitempty"`
	ScaleFactor         int64         `json:"scale_factor,omitempty"`
	Recipient           string        `json:"recipient,omitempty"`
	RecipientRole       string        `json:"recipient_role,omitempty"`
	PrepareCalls        []PrepareCall `json:"prepare_calls,omitempty"`
}

// Inventory is client-asserted balances (never server-custodied product funds).
// Live load via POST /v1/inventory still stamps inventory_unverified on plans.
type Inventory struct {
	AgentAddress string    `json:"agent_address"`
	Balances     []Balance `json:"balances"`
}

// InventoryRequest is POST /v1/inventory input (agent only; never invent from env/key).
type InventoryRequest struct {
	AgentAddress string `json:"agent_address"`
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

// DepositRequest is CLI JSON input for fixed-N Gateway deposit (no merchant claim / fee).
// Not exposed on HTTP this cut.
type DepositRequest struct {
	Inventory        Inventory      `json:"inventory"`
	SourceChainCAIP2 string         `json:"source_chain_caip2"`
	AmountAtomic     string         `json:"amount_atomic"`
	Orchestration    *Orchestration `json:"orchestration,omitempty"`
	Execute          bool           `json:"execute,omitempty"`
}

// MoveRequest is CLI JSON input for self-land rebalance (land N on dest agent_self).
// Not exposed on HTTP this cut.
type MoveRequest struct {
	DestChainCAIP2 string         `json:"dest_chain_caip2"`
	AmountAtomic   string         `json:"amount_atomic"`
	Inventory      Inventory      `json:"inventory"`
	Orchestration  *Orchestration `json:"orchestration,omitempty"`
	Execute        bool           `json:"execute,omitempty"`
}

// FundingSource is one hard-coded deposit source for scenario full-funding plans.
// amount_atomic is REAL (on-chain); amount_logical_atomic is optional stamp metadata.
type FundingSource struct {
	ChainCAIP2          string `json:"chain_caip2"`
	AmountAtomic        string `json:"amount_atomic"`                   // real
	AmountLogicalAtomic string `json:"amount_logical_atomic,omitempty"` // logical pre-scale
}

// PaymentFundingRequest is POST /v1/payment-funding input (full hard-coded funding, not shortfall).
type PaymentFundingRequest struct {
	Required  Required        `json:"required"` // amount_atomic = REAL payment amount
	Inventory Inventory       `json:"inventory"`
	Sources   []FundingSource `json:"sources"`
	Execute   bool            `json:"execute,omitempty"`
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
