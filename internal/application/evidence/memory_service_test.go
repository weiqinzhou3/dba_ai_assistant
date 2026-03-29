package evidence

import (
	"context"
	"testing"
)

func TestMemoryServiceBuildPersistsTraceID(t *testing.T) {
	service := NewMemoryService()

	pack, err := service.Build(context.Background(), BuildInput{
		OrderID:          "ord_01",
		TaskID:           "task_01",
		TraceID:          "trace_01",
		ExecutionSuccess: true,
		ResultSummary:    "skeleton",
	})
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	if pack.TraceID != "trace_01" {
		t.Fatalf("expected built evidence pack to carry trace id, got %q", pack.TraceID)
	}

	view, err := service.GetByOrderID(context.Background(), "ord_01")
	if err != nil {
		t.Fatalf("GetByOrderID returned error: %v", err)
	}
	if view.TraceID != "trace_01" {
		t.Fatalf("expected stored evidence pack to carry trace id, got %q", view.TraceID)
	}
}
