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
}

// PlanStep is one fund-movement or note. Fund steps use recipient_role=agent_self.
type PlanStep struct {
	Kind           string `json:"kind"`
	FromChainCAIP2 string `json:"from_chain_caip2,omitempty"`
	ToChainCAIP2   string `json:"to_chain_caip2,omitempty"`
	Asset          string `json:"asset,omitempty"`
	AmountAtomic   string `json:"amount_atomic,omitempty"`
	Recipient      string `json:"recipient,omitempty"`
	RecipientRole  string `json:"recipient_role,omitempty"`
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
	Required       Required  `json:"required"`
	Inventory      Inventory `json:"inventory"`
	AmountOverride string    `json:"amount_override,omitempty"`
	Execute        bool      `json:"execute,omitempty"`
}

// PlanResponse is the HTTP plan output.
type PlanResponse struct {
	Plan  Plan      `json:"plan"`
	Error *APIError `json:"error,omitempty"`
}

// APIError is the JSON error body.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
