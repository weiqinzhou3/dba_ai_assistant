# Phase 01 - Codex Plan

## 文档作用

由 Codex 在进入 Phase 01 编码前填写，明确本 phase 的范围、计划顺序、验证方式和禁止事项自检。

## 当前 phase 目标

固化 skeleton、interfaces、guardrails，不进入真实 `mysql.database.create` 执行。

## 已有产物盘点

在开始本 phase 前，必须先盘点当前仓库已有代码与测试，并回答：

- 已存在的 domain objects：
  - `ActionRequest`
  - `Principal`
  - `ResolvedAssetSet`
  - `PolicyDecision`
  - `RiskDecision`
  - `AuthorizationDecision`
  - `AssistantOrder`
  - `ExecutionPlan`
  - `ApprovalRecord` / `ApprovalState`
  - `ExecutionTask`
  - `AuditEvent`
  - `EvidencePack`
- 已存在的 application interfaces / services：
  - `ActionRequestService`
  - `AuthorizationService`
  - `ExecuteAuthorizationService`
  - `ApprovalService`
  - `ExecutionPlanner`
  - `ExecutionRouter`
  - `TaskRuntime`
  - `AuditService`
  - `EvidenceService`
- 已存在的 northbound API skeleton：
  - `POST /api/v1/action-requests`
  - `GET /api/v1/orders/{order_id}`
  - `GET /api/v1/tasks/{task_id}`
  - `POST /api/v1/orders/{order_id}/approvals`
  - `POST /api/v1/orders/{order_id}/execute`
  - `GET /api/v1/audit-ledger/{request_id}`
  - `GET /api/v1/evidence-packs/{order_id}`
- 已存在的 adapter SPI / `DBNativeAdapter` skeleton：
  - `internal/application/execution/contracts.go` 已定义 `Adapter` / `AdapterDryRunResult` / `AdapterExecutionResult`
  - `internal/adapters/dbnative/adapter.go` 仍是 Phase 01 stub
- 已存在的测试：
  - authorization aggregation
  - exact asset guardrail
  - execute auth guardrail
  - action request / execute separation
  - API unified entry / error mapping
  - idempotency key format
  - skill contract mapping
- 与 Phase 01 验收标准已对齐的项：
  - `AuthorizationService` 是 request 路径唯一最终授权出口
  - `AssetResolver` 仅暴露 `ResolveExact(...)`
  - approval / execute northbound 路由已分离
  - `control_executor` 已进入 role model
  - `ApprovalPolicy` TTL 与 `ExecutePolicy` 已建模
  - `DryRun(...)` 与 `AdapterDryRunResult` 已进入 Adapter SPI
  - `DBNativeAdapter` 仍保持 skeleton
- 仍需补齐的 gap：
  - all northbound / skill 输出上的 `trace_id` contract 未全覆盖
  - approval route separation、`DBNativeAdapter` skeleton、`DryRun` contract 仍缺锁定测试
  - 文档层还未形成正式 Phase 01 gap analysis / delta / handoff 记录

## 必须回应事项

1. DryRun 在 Adapter SPI skeleton 中的落点是什么：
   - 落在 `internal/application/execution/contracts.go` 的 `Adapter` interface 上，作为 southbound SPI 的一级方法。
2. `AdapterDryRunResult` 是否已存在，若不存在如何补：
   - 已存在于 `internal/application/execution/contracts.go`，本轮不新增第二套 dry-run contract，只补测试锁定。
3. Skill contract structs 是否完整覆盖：
   - `request_mysql_database_create`
   - `execute_assistant_order`
   - 已覆盖；本轮补齐 `trace_id` 映射，使其与 northbound 契约一致。

## 本轮计划

- 当前分支：
  - `feat/p1-baseline-gap-and-guardrails`
- 本轮目标：
  - 基于现有 Go skeleton 做 Phase 01 级盘点、gap analysis 和最小 guardrail 收口。
- 本轮范围：
  - 补 northbound / skill `trace_id` contract
  - 补 approval separation / adapter skeleton / dry-run tests
  - 更新 dashboard / status / handoff / implementation status
- 明确不做：
  - 不实现真实 `mysql.database.create`
  - 不推进 approval 共享状态机
  - 不推进 execute `re-validate -> router -> runtime`
  - 不接 Deep Agent / MCP / 多 adapter
- 预计修改路径：
  - `internal/domain/order/types.go`
  - `internal/domain/task/types.go`
  - `internal/application/approval/*`
  - `internal/application/audit/*`
  - `internal/application/evidence/*`
  - `internal/skill/*`
  - `internal/api/server_test.go`
  - `internal/adapters/dbnative/adapter_test.go`
  - `design_docs/22-phase-01-gap-analysis-v0.1.md`
  - `design_docs/23-phase-01-skeleton-delta-v0.1.md`
- 计划验证命令：
  - `gofmt -w $(find internal cmd -type f -name '*.go')`
  - `go test ./...`
- 请求 Claude review 的重点：
  - 是否仍严格停留在 Phase 01，不越界进入 Phase 02
  - `trace_id` contract 是否已覆盖 northbound / skill 出口
  - `DBNativeAdapter` 是否仍保持 skeleton-only
