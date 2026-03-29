package evidence

import (
	"context"
	"sync"
)

type MemoryService struct {
	mu    sync.Mutex
	packs map[string]Pack
}

func NewMemoryService() *MemoryService {
	return &MemoryService{
		packs: map[string]Pack{},
	}
}

func (s *MemoryService) Build(_ context.Context, input BuildInput) (Pack, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	pack := Pack{
		EvidenceID:          "evd_" + input.OrderID,
		OrderID:             input.OrderID,
		TaskID:              input.TaskID,
		RequestSummary:      input.RequestSummary,
		BeforeStateSnapshot: input.BeforeStateSnapshot,
		AfterStateSnapshot:  input.AfterStateSnapshot,
		ApprovalRefs:        append([]string(nil), input.ApprovalRefs...),
		ExecutionSuccess:    input.ExecutionSuccess,
		FailureDetail:       input.FailureDetail,
		ResultSummary:       input.ResultSummary,
		RollbackSuggestion:  input.RollbackSuggestion,
	}
	s.packs[input.OrderID] = pack
	return pack, nil
}

func (s *MemoryService) GetByOrderID(_ context.Context, orderID string) (PackView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.packs[orderID], nil
}
