package action

import (
	"time"

	"dba_ai_assistant/internal/domain/asset"
)

type Name string

const (
	NameMySQLDatabaseCreate Name = "mysql.database.create"
)

type RequestStatus string

const (
	RequestStatusAccepted RequestStatus = "ACCEPTED"
	RequestStatusRejected RequestStatus = "REJECTED"
)

type Request struct {
	RequestID        string         `json:"request_id"`
	TraceID          string         `json:"trace_id"`
	PrincipalID      string         `json:"principal_id"`
	ActionName       Name           `json:"action_name"`
	ResourceSelector asset.Selector `json:"resource_selector"`
	Parameters       map[string]any `json:"parameters"`
	RequestContext   map[string]any `json:"request_context"`
	Source           string         `json:"source"`
	Status           RequestStatus  `json:"status"`
	CreatedAt        time.Time      `json:"created_at"`
}
