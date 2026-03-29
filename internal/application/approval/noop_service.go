package approval

import (
	"context"
	"time"

	"dba_ai_assistant/internal/domain/common"
	"dba_ai_assistant/internal/domain/order"
)

type NoopService struct{}

func NewNoopService() *NoopService {
	return &NoopService{}
}

func (s *NoopService) Create(_ context.Context, ord order.AssistantOrder, _ CreateInput) (State, error) {
	return State{
		OrderID:        ord.OrderID,
		TraceID:        ord.TraceID,
		ApprovalStatus: ord.ApprovalStatus,
		Status:         ord.Status,
	}, nil
}

func (s *NoopService) Decide(_ context.Context, orderID string, _ DecisionInput) (State, error) {
	return State{}, common.NewError(common.CodeOrderNotExecutable, "approval stub is not wired to a shared order store yet", map[string]any{"order_id": orderID})
}

func (s *NoopService) Get(_ context.Context, orderID string) (State, error) {
	return State{}, common.NewError(common.CodeRequestInvalid, "approval state not found", map[string]any{"order_id": orderID})
}

func (s *NoopService) ExpireStaleApprovals(_ context.Context, _ time.Time, _ int) ([]ExpiryResult, error) {
	return nil, nil
}
