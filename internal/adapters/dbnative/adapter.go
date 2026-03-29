package dbnative

import (
	"context"
	"time"

	appexec "dba_ai_assistant/internal/application/execution"
)

type DBNativeAdapter struct{}

func New() *DBNativeAdapter {
	return &DBNativeAdapter{}
}

func (a *DBNativeAdapter) Type() appexec.AdapterType {
	return appexec.AdapterTypeDBNative
}

func (a *DBNativeAdapter) Supports(_ context.Context, req appexec.AdapterCapabilityRequest) (bool, error) {
	return req.ActionName == "mysql.database.create" && req.TargetAssetType == "DatabaseTarget", nil
}

func (a *DBNativeAdapter) DryRun(_ context.Context, req appexec.AdapterExecutionRequest) (appexec.AdapterDryRunResult, error) {
	return appexec.AdapterDryRunResult{
		Supported: true,
		Ready:     true,
		Issues:    nil,
		RenderedPreview: map[string]any{
			"action_name": req.ActionName,
			"mode":        "dry_run_stub",
		},
	}, nil
}

func (a *DBNativeAdapter) Execute(_ context.Context, req appexec.AdapterExecutionRequest) (appexec.AdapterExecutionResult, error) {
	now := time.Now().UTC()
	return appexec.AdapterExecutionResult{
		Success:   false,
		Status:    "FAILED",
		Summary:   "dbnative adapter skeleton only; real mysql.database.create is intentionally not implemented in this phase",
		StartedAt: now,
		EndedAt:   now,
		Error: map[string]any{
			"code":    "NOT_IMPLEMENTED_IN_PHASE_1",
			"message": "dbnative execution is intentionally stubbed until the next slice",
		},
	}, nil
}
