package dbnative

import (
	"context"
	"testing"

	appexec "dba_ai_assistant/internal/application/execution"
)

func TestDBNativeAdapterDryRunReturnsSkeletonPreview(t *testing.T) {
	adapter := New()

	result, err := adapter.DryRun(context.Background(), appexec.AdapterExecutionRequest{
		TraceID:    "trace_01",
		OrderID:    "ord_01",
		ActionName: "mysql.database.create",
	})
	if err != nil {
		t.Fatalf("DryRun returned error: %v", err)
	}
	if !result.Supported || !result.Ready {
		t.Fatalf("expected dry run skeleton to report supported and ready")
	}
	if result.RenderedPreview["mode"] != "dry_run_stub" {
		t.Fatalf("expected dry run skeleton preview, got %#v", result.RenderedPreview["mode"])
	}
}

func TestDBNativeAdapterExecuteRemainsPhaseOneSkeleton(t *testing.T) {
	adapter := New()

	result, err := adapter.Execute(context.Background(), appexec.AdapterExecutionRequest{
		TraceID:    "trace_01",
		OrderID:    "ord_01",
		ActionName: "mysql.database.create",
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if result.Success {
		t.Fatalf("phase 1 adapter must not report successful execution")
	}
	if result.Status != "FAILED" {
		t.Fatalf("expected FAILED status, got %q", result.Status)
	}
	if result.Error["code"] != "NOT_IMPLEMENTED_IN_PHASE_1" {
		t.Fatalf("expected phase 1 not implemented error, got %#v", result.Error["code"])
	}
}
