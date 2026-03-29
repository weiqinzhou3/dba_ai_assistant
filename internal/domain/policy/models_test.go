package policy

import (
	"testing"
	"time"

	"dba_ai_assistant/internal/domain/principal"
)

func TestPolicyModelsExposeApprovalTTLAndExecutePolicy(t *testing.T) {
	approvalPolicy := ApprovalPolicy{
		PolicyID:   "approval_01",
		ActionName: "mysql.database.create",
		TTL:        72 * time.Hour,
	}
	if approvalPolicy.TTL != 72*time.Hour {
		t.Fatalf("expected TTL to be stored on ApprovalPolicy")
	}

	executePolicy := ExecutePolicy{
		PolicyID:    "execute_01",
		SubjectType: SubjectTypeRole,
		SubjectRef:  principal.RoleControlExecutor,
		ActionName:  "mysql.database.create",
		Effect:      EffectAllow,
	}
	if executePolicy.SubjectRef != principal.RoleControlExecutor {
		t.Fatalf("expected control_executor role to be representable in ExecutePolicy")
	}
}
