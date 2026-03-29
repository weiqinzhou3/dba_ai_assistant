package evidence

import "context"

type packRepository interface {
	SaveEvidencePack(ctx context.Context, pack Pack) error
	GetEvidencePackByOrderID(ctx context.Context, orderID string) (Pack, error)
}

type MemoryService struct {
	repo packRepository
}

func NewMemoryService(repo packRepository) *MemoryService {
	return &MemoryService{repo: repo}
}

func (s *MemoryService) Build(ctx context.Context, input BuildInput) (Pack, error) {
	pack := Pack{
		EvidenceID:          "evd_" + input.OrderID,
		OrderID:             input.OrderID,
		TaskID:              input.TaskID,
		ArtifactRefs:        append([]string(nil), input.ArtifactRefs...),
		RequestSummary:      input.RequestSummary,
		BeforeStateSnapshot: cloneMap(input.BeforeStateSnapshot),
		AfterStateSnapshot:  cloneMap(input.AfterStateSnapshot),
		ApprovalRefs:        append([]string(nil), input.ApprovalRefs...),
		ExecutionSuccess:    input.ExecutionSuccess,
		FailureDetail:       cloneMap(input.FailureDetail),
		ResultSummary:       input.ResultSummary,
		RollbackSuggestion:  input.RollbackSuggestion,
	}
	if err := s.repo.SaveEvidencePack(ctx, pack); err != nil {
		return Pack{}, err
	}
	return pack, nil
}

func (s *MemoryService) GetByOrderID(ctx context.Context, orderID string) (PackView, error) {
	return s.repo.GetEvidencePackByOrderID(ctx, orderID)
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
