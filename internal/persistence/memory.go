package persistence

import (
	"context"
	"sync"

	"dba_ai_assistant/internal/application/approval"
	"dba_ai_assistant/internal/application/audit"
	"dba_ai_assistant/internal/application/evidence"
	"dba_ai_assistant/internal/domain/action"
	"dba_ai_assistant/internal/domain/common"
	"dba_ai_assistant/internal/domain/order"
	"dba_ai_assistant/internal/domain/plan"
	"dba_ai_assistant/internal/domain/policy"
	"dba_ai_assistant/internal/domain/task"
)

type MemoryStore struct {
	mu sync.Mutex

	requests        map[string]action.Request
	orders          map[string]order.AssistantOrder
	plansByOrderID  map[string]plan.ExecutionPlan
	tasksByID       map[string]task.ExecutionTask
	taskIDByOrderID map[string]string

	approvalStatesByOrderID map[string]approval.State
	approvalPolicies        []policy.ApprovalPolicy
	executePolicies         []policy.ExecutePolicy

	auditEvents          []audit.Event
	evidencePacksByOrder map[string]evidence.Pack
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		requests:                map[string]action.Request{},
		orders:                  map[string]order.AssistantOrder{},
		plansByOrderID:          map[string]plan.ExecutionPlan{},
		tasksByID:               map[string]task.ExecutionTask{},
		taskIDByOrderID:         map[string]string{},
		approvalStatesByOrderID: map[string]approval.State{},
		evidencePacksByOrder:    map[string]evidence.Pack{},
	}
}

func (s *MemoryStore) SaveRequest(_ context.Context, request action.Request) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.requests[request.RequestID] = cloneRequest(request)
	return nil
}

func (s *MemoryStore) GetRequest(_ context.Context, requestID string) (action.Request, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	request, ok := s.requests[requestID]
	if !ok {
		return action.Request{}, common.NewError(common.CodeRequestInvalid, "request not found", map[string]any{"request_id": requestID})
	}
	return cloneRequest(request), nil
}

func (s *MemoryStore) SaveOrder(_ context.Context, ord order.AssistantOrder) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.orders[ord.OrderID] = cloneOrder(ord)
	return nil
}

func (s *MemoryStore) GetOrder(_ context.Context, orderID string) (order.AssistantOrder, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ord, ok := s.orders[orderID]
	if !ok {
		return order.AssistantOrder{}, common.NewError(common.CodeRequestInvalid, "order not found", map[string]any{"order_id": orderID})
	}
	return cloneOrder(ord), nil
}

func (s *MemoryStore) SavePlan(_ context.Context, executionPlan plan.ExecutionPlan) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.plansByOrderID[executionPlan.OrderID] = clonePlan(executionPlan)
	return nil
}

func (s *MemoryStore) GetPlanByOrderID(_ context.Context, orderID string) (plan.ExecutionPlan, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	executionPlan, ok := s.plansByOrderID[orderID]
	if !ok {
		return plan.ExecutionPlan{}, common.NewError(common.CodeRequestInvalid, "plan not found", map[string]any{"order_id": orderID})
	}
	return clonePlan(executionPlan), nil
}

func (s *MemoryStore) SaveTask(_ context.Context, executionTask task.ExecutionTask) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.tasksByID[executionTask.TaskID] = executionTask
	if executionTask.OrderID != "" {
		s.taskIDByOrderID[executionTask.OrderID] = executionTask.TaskID
	}
	return nil
}

func (s *MemoryStore) GetTask(_ context.Context, taskID string) (task.ExecutionTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	executionTask, ok := s.tasksByID[taskID]
	if !ok {
		return task.ExecutionTask{}, common.NewError(common.CodeRequestInvalid, "task not found", map[string]any{"task_id": taskID})
	}
	return executionTask, nil
}

func (s *MemoryStore) GetTaskByOrderID(_ context.Context, orderID string) (task.ExecutionTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	taskID, ok := s.taskIDByOrderID[orderID]
	if !ok {
		return task.ExecutionTask{}, common.NewError(common.CodeRequestInvalid, "task not found", map[string]any{"order_id": orderID})
	}
	return s.tasksByID[taskID], nil
}

func (s *MemoryStore) SaveApprovalState(_ context.Context, state approval.State) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.approvalStatesByOrderID[state.OrderID] = cloneApprovalState(state)
	return nil
}

func (s *MemoryStore) GetApprovalStateByOrderID(_ context.Context, orderID string) (approval.State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, ok := s.approvalStatesByOrderID[orderID]
	if !ok {
		return approval.State{}, common.NewError(common.CodeRequestInvalid, "approval state not found", map[string]any{"order_id": orderID})
	}
	return cloneApprovalState(state), nil
}

func (s *MemoryStore) ListWaitingApprovalStates(_ context.Context, limit int) ([]approval.State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if limit <= 0 {
		limit = len(s.approvalStatesByOrderID)
	}

	result := make([]approval.State, 0, limit)
	for _, state := range s.approvalStatesByOrderID {
		if state.ApprovalStatus != order.ApprovalStatusWaitingApproval {
			continue
		}
		result = append(result, cloneApprovalState(state))
		if len(result) == limit {
			break
		}
	}
	return result, nil
}

func (s *MemoryStore) SaveApprovalPolicy(_ context.Context, approvalPolicy policy.ApprovalPolicy) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.approvalPolicies = append(s.approvalPolicies, approvalPolicy)
	return nil
}

func (s *MemoryStore) ListApprovalPoliciesByAction(_ context.Context, actionName string) ([]policy.ApprovalPolicy, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var result []policy.ApprovalPolicy
	for _, candidate := range s.approvalPolicies {
		if candidate.ActionName == actionName {
			result = append(result, candidate)
		}
	}
	return append([]policy.ApprovalPolicy(nil), result...), nil
}

func (s *MemoryStore) SaveExecutePolicy(_ context.Context, executePolicy policy.ExecutePolicy) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.executePolicies = append(s.executePolicies, executePolicy)
	return nil
}

func (s *MemoryStore) ListExecutePoliciesByAction(_ context.Context, actionName string) ([]policy.ExecutePolicy, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var result []policy.ExecutePolicy
	for _, candidate := range s.executePolicies {
		if candidate.ActionName == actionName {
			result = append(result, candidate)
		}
	}
	return append([]policy.ExecutePolicy(nil), result...), nil
}

func (s *MemoryStore) AppendAuditEvent(_ context.Context, event audit.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.auditEvents = append(s.auditEvents, cloneAuditEvent(event))
	return nil
}

func (s *MemoryStore) ListAuditEventsByRequestID(_ context.Context, requestID string) ([]audit.Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	filtered := make([]audit.Event, 0, len(s.auditEvents))
	for _, event := range s.auditEvents {
		if event.RequestID == requestID {
			filtered = append(filtered, cloneAuditEvent(event))
		}
	}
	return filtered, nil
}

func (s *MemoryStore) SaveEvidencePack(_ context.Context, pack evidence.Pack) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.evidencePacksByOrder[pack.OrderID] = cloneEvidencePack(pack)
	return nil
}

func (s *MemoryStore) GetEvidencePackByOrderID(_ context.Context, orderID string) (evidence.Pack, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pack, ok := s.evidencePacksByOrder[orderID]
	if !ok {
		return evidence.Pack{}, common.NewError(common.CodeRequestInvalid, "evidence pack not found", map[string]any{"order_id": orderID})
	}
	return cloneEvidencePack(pack), nil
}

func cloneRequest(request action.Request) action.Request {
	request.Parameters = cloneMap(request.Parameters)
	request.RequestContext = cloneMap(request.RequestContext)
	return request
}

func cloneOrder(ord order.AssistantOrder) order.AssistantOrder {
	ord.ResolvedAssetIDs = append([]string(nil), ord.ResolvedAssetIDs...)
	return ord
}

func clonePlan(executionPlan plan.ExecutionPlan) plan.ExecutionPlan {
	executionPlan.AdapterChain = append([]string(nil), executionPlan.AdapterChain...)
	executionPlan.Steps = append([]plan.Step(nil), executionPlan.Steps...)
	return executionPlan
}

func cloneApprovalState(state approval.State) approval.State {
	state.Records = append([]approval.Record(nil), state.Records...)
	return state
}

func cloneAuditEvent(event audit.Event) audit.Event {
	event.Metadata = cloneMap(event.Metadata)
	return event
}

func cloneEvidencePack(pack evidence.Pack) evidence.Pack {
	pack.ArtifactRefs = append([]string(nil), pack.ArtifactRefs...)
	pack.ApprovalRefs = append([]string(nil), pack.ApprovalRefs...)
	pack.BeforeStateSnapshot = cloneMap(pack.BeforeStateSnapshot)
	pack.AfterStateSnapshot = cloneMap(pack.AfterStateSnapshot)
	pack.FailureDetail = cloneMap(pack.FailureDetail)
	return pack
}

func cloneMap(input map[string]any) map[string]any {
	if len(input) == 0 {
		return nil
	}
	output := make(map[string]any, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}
