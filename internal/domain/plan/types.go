package plan

import "time"

type Status string

const (
	StatusDraft       Status = "DRAFT"
	StatusFrozen      Status = "FROZEN"
	StatusRevalidated Status = "REVALIDATED"
	StatusStale       Status = "STALE"
	StatusConsumed    Status = "CONSUMED"
)

type Step struct {
	StepID         string `json:"step_id"`
	Priority       int    `json:"priority"`
	AdapterType    string `json:"adapter_type"`
	Operation      string `json:"operation"`
	TimeoutSeconds int    `json:"timeout_seconds"`
}

type ExecutionPlan struct {
	PlanID              string    `json:"plan_id"`
	OrderID             string    `json:"order_id"`
	PlanVersion         int       `json:"plan_version"`
	PlanStatus          Status    `json:"plan_status"`
	SelectedRoute       string    `json:"selected_route,omitempty"`
	AdapterChain        []string  `json:"adapter_chain,omitempty"`
	Steps               []Step    `json:"steps,omitempty"`
	RollbackStrategy    string    `json:"rollback_strategy,omitempty"`
	IdempotencyStrategy string    `json:"idempotency_strategy,omitempty"`
	SnapshotFrozen      bool      `json:"snapshot_frozen"`
	ValidatedAt         time.Time `json:"validated_at,omitempty"`
	StaleReason         string    `json:"stale_reason,omitempty"`
}
