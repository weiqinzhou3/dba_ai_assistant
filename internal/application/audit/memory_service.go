package audit

import (
	"context"
	"fmt"
	"time"
)

type eventRepository interface {
	AppendAuditEvent(ctx context.Context, event Event) error
	ListAuditEventsByRequestID(ctx context.Context, requestID string) ([]Event, error)
}

type MemoryService struct {
	repo     eventRepository
	sequence int
}

type localEventRepository struct {
	events []Event
}

func (r *localEventRepository) AppendAuditEvent(_ context.Context, event Event) error {
	r.events = append(r.events, event)
	return nil
}

func (r *localEventRepository) ListAuditEventsByRequestID(_ context.Context, requestID string) ([]Event, error) {
	filtered := make([]Event, 0, len(r.events))
	for _, event := range r.events {
		if event.RequestID == requestID {
			filtered = append(filtered, event)
		}
	}
	return filtered, nil
}

func NewMemoryService(repos ...eventRepository) *MemoryService {
	repo := eventRepository(&localEventRepository{})
	if len(repos) > 0 && repos[0] != nil {
		repo = repos[0]
	}
	return &MemoryService{repo: repo}
}

func (s *MemoryService) AppendEvent(ctx context.Context, event Event) error {
	s.sequence++
	if event.EventID == "" {
		event.EventID = fmt.Sprintf("audit_%04d", s.sequence)
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}
	return s.repo.AppendAuditEvent(ctx, event)
}

func (s *MemoryService) ListEventsByRequestID(ctx context.Context, requestID string) ([]Event, error) {
	return s.repo.ListAuditEventsByRequestID(ctx, requestID)
}

func (s *MemoryService) GetViewByRequestID(ctx context.Context, requestID string) (LedgerView, error) {
	events, err := s.ListEventsByRequestID(ctx, requestID)
	if err != nil {
		return LedgerView{}, err
	}

	traceID := ""
	for _, event := range events {
		if event.TraceID != "" {
			traceID = event.TraceID
		}
	}

	view := LedgerView{
		RequestID:  requestID,
		TraceID:    traceID,
		Events:     events,
		EventCount: len(events),
	}

	for _, event := range events {
		if event.OrderID != "" {
			view.LatestOrderID = event.OrderID
		}
		if event.TaskID != "" {
			view.LatestTaskID = event.TaskID
		}
		if status, _ := event.Metadata["order_status"].(string); status != "" {
			view.LatestOrderStatus = status
		}
		if status, _ := event.Metadata["approval_status"].(string); status != "" {
			view.LatestApprovalStatus = status
		}
		if summary, _ := event.Metadata["execution_summary"].(string); summary != "" {
			view.LatestExecutionSummary = summary
		}
		if code, _ := event.Metadata["error_code"].(string); code != "" {
			view.LatestErrorCode = code
		}
		if message, _ := event.Metadata["error_message"].(string); message != "" {
			view.LatestErrorMessage = message
		}
		if event.Success != nil {
			success := *event.Success
			view.LatestSuccess = &success
		}
		if event.CreatedAt.After(view.LastEventAt) {
			view.LastEventAt = event.CreatedAt
		}
	}

	return view, nil
}
