package skill

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	appaction "dba_ai_assistant/internal/application/actionrequest"
	appauth "dba_ai_assistant/internal/application/authorization"
	"dba_ai_assistant/internal/domain/asset"
	"dba_ai_assistant/internal/domain/common"
	"dba_ai_assistant/internal/domain/order"
)

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type Dependencies struct {
	BaseURL    string
	HTTPClient HTTPClient
}

type Service struct {
	baseURL    string
	httpClient HTTPClient
}

type apiErrorEnvelope struct {
	Error struct {
		Code    string         `json:"code"`
		Message string         `json:"message"`
		Details map[string]any `json:"details,omitempty"`
	} `json:"error"`
	TraceID string `json:"trace_id"`
}

func NewService(deps Dependencies) (*Service, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(deps.BaseURL), "/")
	if baseURL == "" {
		return nil, common.NewError(common.CodeRequestInvalid, "skill base URL is required", nil)
	}

	httpClient := deps.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &Service{
		baseURL:    baseURL,
		httpClient: httpClient,
	}, nil
}

func (s *Service) RequestMySQLDatabaseCreate(ctx context.Context, authCtx appauth.AuthContext, input RequestMySQLDatabaseCreateInput) (RequestMySQLDatabaseCreateOutput, error) {
	if strings.TrimSpace(authCtx.AuthenticatedPrincipalID) == "" {
		return RequestMySQLDatabaseCreateOutput{}, common.NewError(common.CodePrincipalNotFound, "authenticated principal is required", nil)
	}
	if strings.TrimSpace(input.Project) == "" || strings.TrimSpace(input.Environment) == "" || strings.TrimSpace(input.ServiceInstance) == "" {
		return RequestMySQLDatabaseCreateOutput{}, common.NewError(common.CodeRequestInvalid, "project/environment/service_instance are required", nil)
	}
	if strings.TrimSpace(input.DatabaseName) == "" {
		return RequestMySQLDatabaseCreateOutput{}, common.NewError(common.CodeRequestInvalid, "database_name is required", nil)
	}

	req := appaction.ActionRequestDTO{
		PrincipalID: authCtx.AuthenticatedPrincipalID,
		ActionHint:  "mysql.database.create",
		ResourceSelector: asset.Selector{
			Project:         input.Project,
			Environment:     input.Environment,
			ServiceInstance: input.ServiceInstance,
		},
		Parameters: map[string]any{
			"database_name": input.DatabaseName,
		},
		RequestContext: buildRequestContext(authCtx, input),
	}
	if input.CharacterSet != "" {
		req.Parameters["character_set"] = input.CharacterSet
	}
	if input.Collation != "" {
		req.Parameters["collation"] = input.Collation
	}

	var submitResult appaction.ActionSubmissionResult
	if err := s.doJSON(ctx, http.MethodPost, "/api/v1/action-requests", req, authCtx, &submitResult); err != nil {
		return RequestMySQLDatabaseCreateOutput{}, err
	}

	output := MapRequestMySQLDatabaseCreateOutput(submitResult)
	if !input.AutoExecute || submitResult.ApprovalRequired || submitResult.Status != order.StatusApproved {
		return output, nil
	}

	output.AutoExecuteTriggered = true
	executeResult, err := s.ExecuteAssistantOrder(ctx, authCtx, ExecuteAssistantOrderInput{
		OrderID: submitResult.OrderID,
		Reason:  autoExecuteReason(input.Reason),
	})
	if err != nil {
		output.AutoExecuteError = skillOperationErrorFromError(err)
		output.UserMessage = autoExecuteFailureUserMessage(output.AutoExecuteError)
		return output, nil
	}

	output.AutoExecuteResult = &executeResult
	output.OrderStatus = executeResult.OrderStatus
	output.TaskID = executeResult.TaskID
	output.UserMessage = autoExecuteSuccessUserMessage(executeResult)
	return output, nil
}

func (s *Service) ExecuteAssistantOrder(ctx context.Context, authCtx appauth.AuthContext, input ExecuteAssistantOrderInput) (ExecuteAssistantOrderOutput, error) {
	if strings.TrimSpace(authCtx.AuthenticatedPrincipalID) == "" {
		return ExecuteAssistantOrderOutput{}, common.NewError(common.CodePrincipalNotFound, "authenticated principal is required", nil)
	}
	if strings.TrimSpace(input.OrderID) == "" {
		return ExecuteAssistantOrderOutput{}, common.NewError(common.CodeRequestInvalid, "order_id is required", nil)
	}

	var executeResult appaction.ExecuteOrderResult
	if err := s.doJSON(ctx, http.MethodPost, "/api/v1/orders/"+url.PathEscape(input.OrderID)+"/execute", appaction.ExecuteOrderInput{
		OrderID: input.OrderID,
		Reason:  input.Reason,
	}, authCtx, &executeResult); err != nil {
		return ExecuteAssistantOrderOutput{}, err
	}
	return MapExecuteAssistantOrderOutput(executeResult), nil
}

func (s *Service) doJSON(ctx context.Context, method string, path string, requestBody any, authCtx appauth.AuthContext, responseBody any) error {
	payload, err := json.Marshal(requestBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, method, s.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if authCtx.AuthenticatedPrincipalID != "" {
		req.Header.Set("X-Principal-ID", authCtx.AuthenticatedPrincipalID)
	}
	if len(authCtx.Roles) > 0 {
		req.Header.Set("X-Roles", strings.Join(authCtx.Roles, ","))
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if responseBody == nil {
			return nil
		}
		return json.NewDecoder(resp.Body).Decode(responseBody)
	}

	var apiErr apiErrorEnvelope
	if decodeErr := json.NewDecoder(resp.Body).Decode(&apiErr); decodeErr != nil {
		return fmt.Errorf("control API %s %s returned status %d", method, path, resp.StatusCode)
	}
	return common.NewError(common.Code(apiErr.Error.Code), apiErr.Error.Message, detailsWithTraceID(apiErr.Error.Details, apiErr.TraceID))
}

func buildRequestContext(authCtx appauth.AuthContext, input RequestMySQLDatabaseCreateInput) map[string]any {
	source := strings.TrimSpace(authCtx.Source)
	if source == "" {
		source = "deep_agent"
	}

	requestContext := map[string]any{
		"source": source,
	}
	if input.ConversationID != "" {
		requestContext["conversation_id"] = input.ConversationID
	}
	if input.MessageID != "" {
		requestContext["message_id"] = input.MessageID
	}
	if input.Reason != "" {
		requestContext["reason"] = input.Reason
	}
	return requestContext
}

func autoExecuteReason(reason string) string {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return "auto-chain from request_mysql_database_create"
	}
	return "auto-chain from request_mysql_database_create: " + reason
}

func autoExecuteSuccessUserMessage(result ExecuteAssistantOrderOutput) string {
	if result.TaskID != "" {
		return "请求已创建并已自动触发执行，任务 " + result.TaskID + " 已进入受控执行链路。"
	}
	return "请求已创建并已自动触发执行。"
}

func autoExecuteFailureUserMessage(opErr *SkillOperationError) string {
	if opErr == nil {
		return "请求已创建，但自动 execute 未成功；请由具备执行权限的主体显式调用 execute_assistant_order。"
	}
	switch opErr.Code {
	case string(common.CodeExecutorNotAllowed):
		return "请求已创建，但当前主体没有 execute 权限；请由具备执行权限的主体显式调用 execute_assistant_order。"
	case string(common.CodeApprovalRequired):
		return "请求已创建，但当前工单仍需要审批；审批完成后再显式调用 execute_assistant_order。"
	default:
		return "请求已创建，但自动 execute 失败；请根据工单状态决定是否重新显式调用 execute_assistant_order。"
	}
}

func skillOperationErrorFromError(err error) *SkillOperationError {
	if err == nil {
		return nil
	}

	opErr := &SkillOperationError{
		Code:    string(common.ErrorCode(err)),
		Message: err.Error(),
	}

	var coded *common.CodedError
	if common.As(err, &coded) {
		opErr.Message = coded.Message
		opErr.Details = cloneMap(coded.Details)
		if traceID, ok := coded.Details["trace_id"].(string); ok {
			opErr.TraceID = traceID
		}
	}
	return opErr
}

func detailsWithTraceID(details map[string]any, traceID string) map[string]any {
	cloned := cloneMap(details)
	if traceID != "" {
		cloned["trace_id"] = traceID
	}
	return cloned
}

func cloneMap(input map[string]any) map[string]any {
	if len(input) == 0 {
		return nil
	}
	cloned := make(map[string]any, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}
