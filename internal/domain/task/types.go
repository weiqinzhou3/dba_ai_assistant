package task

import "time"

type Status string

const (
	StatusPending   Status = "PENDING"
	StatusRunning   Status = "RUNNING"
	StatusSucceeded Status = "SUCCEEDED"
	StatusFailed    Status = "FAILED"
	StatusTimeout   Status = "TIMEOUT"
	StatusCancelled Status = "CANCELLED"
)

type ExecutionTask struct {
	TaskID      string    `json:"task_id"`
	OrderID     string    `json:"order_id"`
	TraceID     string    `json:"trace_id"`
	ActionName  string    `json:"action_name"`
	Status      Status    `json:"status"`
	StartedAt   time.Time `json:"started_at,omitempty"`
	EndedAt     time.Time `json:"ended_at,omitempty"`
	HeartbeatAt time.Time `json:"heartbeat_at,omitempty"`
}
