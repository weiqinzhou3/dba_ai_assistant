package audit

import (
	"context"
	"testing"
)

func TestMemoryServiceGetViewByRequestIDIncludesTraceID(t *testing.T) {
	service := NewMemoryService()

	err := service.AppendEvent(context.Background(), Event{
		EventType: EventRequestAccepted,
		RequestID: "req_01",
		TraceID:   "trace_01",
	})
	if err != nil {
		t.Fatalf("AppendEvent returned error: %v", err)
	}

	view, err := service.GetViewByRequestID(context.Background(), "req_01")
	if err != nil {
		t.Fatalf("GetViewByRequestID returned error: %v", err)
	}
	if view.TraceID != "trace_01" {
		t.Fatalf("expected trace id to be preserved, got %q", view.TraceID)
	}
}
