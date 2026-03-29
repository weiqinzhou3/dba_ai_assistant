package execution

import (
	"context"
	"time"

	"dba_ai_assistant/internal/domain/order"
	"dba_ai_assistant/internal/domain/plan"
	"dba_ai_assistant/internal/domain/task"
)

type StaticExecutionPlanner struct{}

func NewStaticExecutionPlanner() *StaticExecutionPlanner {
	return &StaticExecutionPlanner{}
}

func (p *StaticExecutionPlanner) Build(_ context.Context, ord order.AssistantOrder) (plan.ExecutionPlan, error) {
	return plan.ExecutionPlan{
		PlanID:              "plan_" + ord.OrderID,
		OrderID:             ord.OrderID,
		PlanVersion:         1,
		PlanStatus:          plan.StatusFrozen,
		SelectedRoute:       string(AdapterTypeDBNative),
		AdapterChain:        []string{string(AdapterTypeDBNative)},
		IdempotencyStrategy: "action+asset+database_name",
		SnapshotFrozen:      true,
		Steps: []plan.Step{
			{StepID: "step_validate_target", Priority: 1, AdapterType: string(AdapterTypeDBNative), Operation: "validate_target", TimeoutSeconds: 10},
			{StepID: "step_check_database_not_exists", Priority: 2, AdapterType: string(AdapterTypeDBNative), Operation: "check_database_not_exists", TimeoutSeconds: 10},
			{StepID: "step_create_database", Priority: 3, AdapterType: string(AdapterTypeDBNative), Operation: "create_database", TimeoutSeconds: 30},
			{StepID: "step_verify_database_created", Priority: 4, AdapterType: string(AdapterTypeDBNative), Operation: "verify_database_created", TimeoutSeconds: 10},
		},
	}, nil
}

func (p *StaticExecutionPlanner) Revalidate(_ context.Context, _ order.AssistantOrder, existing plan.ExecutionPlan) (PlanValidationResult, error) {
	if existing.PlanStatus == plan.StatusStale || existing.StaleReason != "" {
		reason := existing.StaleReason
		if reason == "" {
			reason = "plan already marked stale"
		}
		return PlanValidationResult{
			Valid:  false,
			Status: plan.StatusStale,
			Reason: reason,
		}, nil
	}
	if !existing.SnapshotFrozen {
		return PlanValidationResult{
			Valid:  false,
			Status: plan.StatusStale,
			Reason: "plan snapshot is not frozen",
		}, nil
	}
	return PlanValidationResult{
		Valid:  true,
		Status: plan.StatusRevalidated,
	}, nil
}

type StaticExecutionRouter struct{}

func NewStaticExecutionRouter() *StaticExecutionRouter {
	return &StaticExecutionRouter{}
}

func (r *StaticExecutionRouter) Route(_ context.Context, _ plan.ExecutionPlan) (AdapterBinding, error) {
	return AdapterBinding{
		AdapterType: string(AdapterTypeDBNative),
		RouteName:   "db_native/mysql",
	}, nil
}

type NoopTaskRuntime struct {
	tasks taskRepository
}

type taskRepository interface {
	SaveTask(ctx context.Context, executionTask task.ExecutionTask) error
	GetTask(ctx context.Context, taskID string) (task.ExecutionTask, error)
}

func NewNoopTaskRuntime(tasks taskRepository) *NoopTaskRuntime {
	return &NoopTaskRuntime{tasks: tasks}
}

func (r *NoopTaskRuntime) Start(ctx context.Context, ord order.AssistantOrder, _ plan.ExecutionPlan) (task.ExecutionTask, error) {
	now := time.Now().UTC()
	created := task.ExecutionTask{
		TaskID:      "task_" + ord.OrderID,
		OrderID:     ord.OrderID,
		TraceID:     ord.TraceID,
		ActionName:  ord.ActionName,
		Status:      task.StatusRunning,
		StartedAt:   now,
		HeartbeatAt: now,
	}
	return created, r.tasks.SaveTask(ctx, created)
}

func (r *NoopTaskRuntime) Get(ctx context.Context, taskID string) (task.ExecutionTask, error) {
	return r.tasks.GetTask(ctx, taskID)
}
