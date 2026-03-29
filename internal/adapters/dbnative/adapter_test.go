package dbnative

import (
	"context"
	"errors"
	"testing"
	"time"

	appexec "dba_ai_assistant/internal/application/execution"
)

func TestDBNativeAdapterDryRunValidatesConnectionAndDatabaseExistence(t *testing.T) {
	adapter := New(Dependencies{
		ConnectionResolver: staticConnectionResolver{
			config: ConnectionConfig{Engine: "mysql", DSN: "mysql-test"},
		},
		MySQL: &fakeMySQLAdmin{
			existing: map[string]bool{},
		},
		Now: fixedNow,
	})

	result, err := adapter.DryRun(context.Background(), testExecutionRequest("order_center"))
	if err != nil {
		t.Fatalf("DryRun returned error: %v", err)
	}
	if !result.Supported || !result.Ready {
		t.Fatalf("expected dry run to report supported=true ready=true, got %+v", result)
	}
	if got, _ := result.RenderedPreview["database_exists"].(bool); got {
		t.Fatalf("expected dry run to report database_exists=false, got %#v", result.RenderedPreview["database_exists"])
	}
}

func TestDBNativeAdapterDryRunReturnsDatabaseAlreadyExistsIssue(t *testing.T) {
	adapter := New(Dependencies{
		ConnectionResolver: staticConnectionResolver{
			config: ConnectionConfig{Engine: "mysql", DSN: "mysql-test"},
		},
		MySQL: &fakeMySQLAdmin{
			existing: map[string]bool{"order_center": true},
		},
		Now: fixedNow,
	})

	result, err := adapter.DryRun(context.Background(), testExecutionRequest("order_center"))
	if err != nil {
		t.Fatalf("DryRun returned error: %v", err)
	}
	if result.Ready {
		t.Fatalf("expected dry run to block when database already exists")
	}
	if len(result.Issues) != 1 || result.Issues[0] != IssueDatabaseAlreadyExists {
		t.Fatalf("expected %s issue, got %+v", IssueDatabaseAlreadyExists, result.Issues)
	}
}

func TestDBNativeAdapterExecuteCreatesDatabaseAndVerifiesIt(t *testing.T) {
	admin := &fakeMySQLAdmin{existing: map[string]bool{}}
	adapter := New(Dependencies{
		ConnectionResolver: staticConnectionResolver{
			config: ConnectionConfig{Engine: "mysql", DSN: "mysql-test"},
		},
		MySQL: admin,
		Now:   fixedNow,
	})

	result, err := adapter.Execute(context.Background(), testExecutionRequest("order_center"))
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if !result.Success || result.Status != "SUCCEEDED" {
		t.Fatalf("expected success result, got %+v", result)
	}
	if got := admin.createCount("order_center"); got != 1 {
		t.Fatalf("expected exactly one CREATE DATABASE call, got %d", got)
	}
	if after, _ := result.Outputs["database_exists_after"].(bool); !after {
		t.Fatalf("expected execution output database_exists_after=true, got %#v", result.Outputs["database_exists_after"])
	}
	if len(result.Artifacts) == 0 {
		t.Fatalf("expected execution artifacts to be returned")
	}
}

func TestDBNativeAdapterExecuteReturnsFailureWhenCreateFails(t *testing.T) {
	adapter := New(Dependencies{
		ConnectionResolver: staticConnectionResolver{
			config: ConnectionConfig{Engine: "mysql", DSN: "mysql-test"},
		},
		MySQL: &fakeMySQLAdmin{
			existing:   map[string]bool{},
			createErrs: map[string]error{"order_center": errors.New("boom")},
		},
		Now: fixedNow,
	})

	result, err := adapter.Execute(context.Background(), testExecutionRequest("order_center"))
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if result.Success {
		t.Fatalf("expected execution to fail when CREATE DATABASE returns an error")
	}
	if result.Error["code"] != "MYSQL_EXECUTION_ERROR" {
		t.Fatalf("expected MYSQL_EXECUTION_ERROR, got %#v", result.Error["code"])
	}
}

func TestDBNativeAdapterExecuteFailsWhenPostCreateVerificationDoesNotObserveDatabase(t *testing.T) {
	adapter := New(Dependencies{
		ConnectionResolver: staticConnectionResolver{
			config: ConnectionConfig{Engine: "mysql", DSN: "mysql-test"},
		},
		MySQL: &fakeMySQLAdmin{
			existing:       map[string]bool{},
			createNoEffect: true,
		},
		Now: fixedNow,
	})

	result, err := adapter.Execute(context.Background(), testExecutionRequest("order_center"))
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if result.Success {
		t.Fatalf("expected verification failure result")
	}
	if result.Error["code"] != "DATABASE_VERIFICATION_FAILED" {
		t.Fatalf("expected DATABASE_VERIFICATION_FAILED, got %#v", result.Error["code"])
	}
}

func testExecutionRequest(databaseName string) appexec.AdapterExecutionRequest {
	return appexec.AdapterExecutionRequest{
		TraceID:        "trace_01",
		OrderID:        "ord_01",
		TaskID:         "task_01",
		ActionName:     "mysql.database.create",
		IdempotencyKey: "mysql.database.create:dbt_1001:" + databaseName,
		Target: map[string]any{
			"asset_id":       "dbt_1001",
			"asset_type":     "DatabaseTarget",
			"connection_ref": "secret://db-targets/mysql-order-main-test",
			"engine":         "mysql",
		},
		Parameters: map[string]any{
			"database_name": databaseName,
		},
	}
}

func fixedNow() time.Time {
	return time.Date(2026, 3, 29, 12, 0, 0, 0, time.UTC)
}

type staticConnectionResolver struct {
	config ConnectionConfig
	err    error
}

func (r staticConnectionResolver) Resolve(_ context.Context, _ string) (ConnectionConfig, error) {
	if r.err != nil {
		return ConnectionConfig{}, r.err
	}
	return r.config, nil
}

type fakeMySQLAdmin struct {
	existing       map[string]bool
	createErrs     map[string]error
	createNoEffect bool
	createCalls    []string
}

func (a *fakeMySQLAdmin) Ping(_ context.Context, _ ConnectionConfig) error {
	return nil
}

func (a *fakeMySQLAdmin) DatabaseExists(_ context.Context, _ ConnectionConfig, databaseName string) (bool, error) {
	return a.existing[databaseName], nil
}

func (a *fakeMySQLAdmin) CreateDatabase(_ context.Context, _ ConnectionConfig, databaseName string) error {
	a.createCalls = append(a.createCalls, databaseName)
	if err := a.createErrs[databaseName]; err != nil {
		return err
	}
	if !a.createNoEffect {
		a.existing[databaseName] = true
	}
	return nil
}

func (a *fakeMySQLAdmin) createCount(databaseName string) int {
	count := 0
	for _, call := range a.createCalls {
		if call == databaseName {
			count++
		}
	}
	return count
}
