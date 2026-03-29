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
	return PlanValidationResult{
		Valid:  true,
		Status: plan.StatusRevalidated,
		Reason: existing.StaleReason,
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
	tasks map[string]task.ExecutionTask
}

func NewNoopTaskRuntime() *NoopTaskRuntime {
	return &NoopTaskRuntime{tasks: map[string]task.ExecutionTask{}}
}

func (r *NoopTaskRuntime) Start(_ context.Context, ord order.AssistantOrder, _ plan.ExecutionPlan) (task.ExecutionTask, error) {
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
	r.tasks[created.TaskID] = created
	return created, nil
}

func (r *NoopTaskRuntime) Get(_ context.Context, taskID string) (task.ExecutionTask, error) {
	if taskValue, ok := r.tasks[taskID]; ok {
		return taskValue, nil
	}
	return task.ExecutionTask{}, nil
}
