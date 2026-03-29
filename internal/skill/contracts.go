package skill

import (
	appaction "dba_ai_assistant/internal/application/actionrequest"
	"dba_ai_assistant/internal/domain/order"
)

type RequestMySQLDatabaseCreateInput struct {
	Project         string `json:"project"`
	Environment     string `json:"environment"`
	ServiceInstance string `json:"service_instance"`
	DatabaseName    string `json:"database_name"`
	CharacterSet    string `json:"character_set,omitempty"`
	Collation       string `json:"collation,omitempty"`
	ConversationID  string `json:"conversation_id,omitempty"`
	MessageID       string `json:"message_id,omitempty"`
	Reason          string `json:"reason,omitempty"`
	AutoExecute     bool   `json:"auto_execute,omitempty"`
}

type SkillOperationError struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	TraceID string         `json:"trace_id,omitempty"`
	Details map[string]any `json:"details,omitempty"`
}

type RequestMySQLDatabaseCreateOutput struct {
	RequestID            string                       `json:"request_id"`
	OrderID              string                       `json:"order_id"`
	ActionName           string                       `json:"action_name"`
	OrderStatus          order.Status                 `json:"order_status"`
	ApprovalRequired     bool                         `json:"approval_required"`
	TaskID               string                       `json:"task_id,omitempty"`
	TraceID              string                       `json:"trace_id"`
	UserMessage          string                       `json:"user_message"`
	NextPollURI          string                       `json:"next_poll_uri"`
	AutoExecuteTriggered bool                         `json:"auto_execute_triggered"`
	AutoExecuteResult    *ExecuteAssistantOrderOutput `json:"auto_execute_result,omitempty"`
	AutoExecuteError     *SkillOperationError         `json:"auto_execute_error,omitempty"`
}

type ExecuteAssistantOrderInput struct {
	OrderID string `json:"order_id"`
	Reason  string `json:"reason,omitempty"`
}

type ExecuteAssistantOrderOutput struct {
	OrderID     string       `json:"order_id"`
	OrderStatus order.Status `json:"order_status"`
	TaskID      string       `json:"task_id,omitempty"`
	ExecutorID  string       `json:"executor_id,omitempty"`
	TraceID     string       `json:"trace_id"`
	UserMessage string       `json:"user_message"`
}

func MapRequestMySQLDatabaseCreateOutput(result appaction.ActionSubmissionResult) RequestMySQLDatabaseCreateOutput {
	return RequestMySQLDatabaseCreateOutput{
		RequestID:        result.RequestID,
		OrderID:          result.OrderID,
		ActionName:       string(result.ActionName),
		OrderStatus:      result.Status,
		ApprovalRequired: result.ApprovalRequired,
		TaskID:           result.TaskID,
		TraceID:          result.TraceID,
		UserMessage:      result.UserMessage,
		NextPollURI:      result.NextPollURI,
	}
}

func MapExecuteAssistantOrderOutput(result appaction.ExecuteOrderResult) ExecuteAssistantOrderOutput {
	return ExecuteAssistantOrderOutput{
		OrderID:     result.OrderID,
		OrderStatus: result.Status,
		TaskID:      result.TaskID,
		ExecutorID:  result.ExecutorID,
		TraceID:     result.TraceID,
		UserMessage: executeUserMessage(result),
	}
}

func executeUserMessage(result appaction.ExecuteOrderResult) string {
	switch result.Status {
	case order.StatusSucceeded:
		return "工单已执行完成。"
	case order.StatusExecuting:
		if result.TaskID != "" {
			return "执行已触发，任务 " + result.TaskID + " 正在运行。"
		}
		return "执行已触发，任务正在运行。"
	default:
		return "execute 调用已受理，当前工单状态为 " + string(result.Status) + "。"
	}
}
