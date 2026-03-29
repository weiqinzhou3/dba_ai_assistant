package execution

import (
	"context"
	"time"

	"dba_ai_assistant/internal/domain/common"
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

type StaticExecutionRouter struct {
	adapters map[AdapterType]Adapter
}

func NewStaticExecutionRouter(adapters ...Adapter) *StaticExecutionRouter {
	indexed := make(map[AdapterType]Adapter, len(adapters))
	for _, adapter := range adapters {
		if adapter == nil {
			continue
		}
		indexed[adapter.Type()] = adapter
	}
	return &StaticExecutionRouter{adapters: indexed}
}

func (r *StaticExecutionRouter) Route(_ context.Context, executionPlan plan.ExecutionPlan) (AdapterBinding, error) {
	adapterType := AdapterTypeDBNative
	if executionPlan.SelectedRoute != "" {
		adapterType = AdapterType(executionPlan.SelectedRoute)
	}
	adapter, ok := r.adapters[adapterType]
	if !ok {
		return AdapterBinding{}, common.NewError(common.CodeAdapterNotAvailable, "execution router could not find a bound adapter", map[string]any{
			"adapter_type": adapterType,
		})
	}
	return AdapterBinding{
		AdapterType: string(adapterType),
		RouteName:   "db_native/mysql",
		Adapter:     adapter,
	}, nil
}

type SynchronousTaskRuntime struct {
	tasks taskRepository
	now   func() time.Time
}

type taskRepository interface {
	SaveTask(ctx context.Context, executionTask task.ExecutionTask) error
	GetTask(ctx context.Context, taskID string) (task.ExecutionTask, error)
}

func NewSynchronousTaskRuntime(tasks taskRepository) *SynchronousTaskRuntime {
	return &SynchronousTaskRuntime{
		tasks: tasks,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func NewNoopTaskRuntime(tasks taskRepository) *SynchronousTaskRuntime {
	return NewSynchronousTaskRuntime(tasks)
}

func (r *SynchronousTaskRuntime) Start(ctx context.Context, input TaskStartInput) (TaskStartResult, error) {
	if input.Binding.Adapter == nil {
		return TaskStartResult{}, common.NewError(common.CodeAdapterNotAvailable, "task runtime received nil adapter binding", map[string]any{
			"order_id": input.Order.OrderID,
		})
	}

	now := r.now()
	taskID := input.AdapterRequest.TaskID
	if taskID == "" {
		taskID = "task_" + input.Order.OrderID
	}

	executionTask := task.ExecutionTask{
		TaskID:      taskID,
		OrderID:     input.Order.OrderID,
		TraceID:     input.Order.TraceID,
		ActionName:  input.Order.ActionName,
		Status:      task.StatusRunning,
		StartedAt:   now,
		HeartbeatAt: now,
	}
	if err := r.tasks.SaveTask(ctx, executionTask); err != nil {
		return TaskStartResult{}, err
	}

	req := input.AdapterRequest
	req.TaskID = taskID
	result, err := input.Binding.Adapter.Execute(ctx, req)
	if err != nil {
		executionTask.Status = task.StatusFailed
		executionTask.EndedAt = r.now()
		executionTask.HeartbeatAt = executionTask.EndedAt
		if saveErr := r.tasks.SaveTask(ctx, executionTask); saveErr != nil {
			return TaskStartResult{}, saveErr
		}
		return TaskStartResult{
			Task:           executionTask,
			AdapterRequest: req,
		}, err
	}

	if result.StartedAt.IsZero() {
		result.StartedAt = executionTask.StartedAt
	}
	if result.EndedAt.IsZero() {
		result.EndedAt = r.now()
	}

	switch result.Status {
	case "SUCCEEDED":
		executionTask.Status = task.StatusSucceeded
	case "TIMEOUT":
		executionTask.Status = task.StatusTimeout
	default:
		executionTask.Status = task.StatusFailed
	}
	executionTask.EndedAt = result.EndedAt
	executionTask.HeartbeatAt = result.EndedAt
	if err := r.tasks.SaveTask(ctx, executionTask); err != nil {
		return TaskStartResult{}, err
	}

	return TaskStartResult{
		Task:           executionTask,
		AdapterRequest: req,
		AdapterResult:  result,
	}, nil
}

func (r *SynchronousTaskRuntime) Get(ctx context.Context, taskID string) (task.ExecutionTask, error) {
	return r.tasks.GetTask(ctx, taskID)
}
