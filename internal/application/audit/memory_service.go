package audit

import (
	"context"
	"sync"
)

type MemoryService struct {
	mu     sync.Mutex
	events []Event
}

func NewMemoryService() *MemoryService {
	return &MemoryService{}
}

func (s *MemoryService) AppendEvent(_ context.Context, event Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.events = append(s.events, event)
	return nil
}

func (s *MemoryService) ListEventsByRequestID(_ context.Context, requestID string) ([]Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	filtered := make([]Event, 0, len(s.events))
	for _, event := range s.events {
		if event.RequestID == requestID {
			filtered = append(filtered, event)
		}
	}
	return filtered, nil
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
			break
		}
	}
	return LedgerView{
		RequestID: requestID,
		TraceID:   traceID,
		Events:    events,
	}, nil
}
