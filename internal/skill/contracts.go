package skill

import (
	appaction "dba_ai_assistant/internal/application/actionrequest"
	"dba_ai_assistant/internal/domain/order"
)

type RequestMySQLDatabaseCreateInput struct {
	TargetAssetID string `json:"target_asset_id"`
	DatabaseName  string `json:"database_name"`
	CharacterSet  string `json:"character_set,omitempty"`
	Collation     string `json:"collation,omitempty"`
}

type RequestMySQLDatabaseCreateOutput struct {
	OrderID          string       `json:"order_id"`
	OrderStatus      order.Status `json:"order_status"`
	RequiresApproval bool         `json:"requires_approval"`
}

type ExecuteAssistantOrderInput struct {
	OrderID string `json:"order_id"`
}

type ExecuteAssistantOrderOutput struct {
	TaskID     string `json:"task_id,omitempty"`
	TaskStatus string `json:"task_status"`
}

func MapRequestMySQLDatabaseCreateOutput(result appaction.ActionSubmissionResult) RequestMySQLDatabaseCreateOutput {
	return RequestMySQLDatabaseCreateOutput{
		OrderID:          result.OrderID,
		OrderStatus:      result.Status,
		RequiresApproval: result.ApprovalRequired,
	}
}

func MapExecuteAssistantOrderOutput(result appaction.ExecuteOrderResult) ExecuteAssistantOrderOutput {
	return ExecuteAssistantOrderOutput{
		TaskID:     result.TaskID,
		TaskStatus: string(result.Status),
	}
}
