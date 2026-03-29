# IMPLEMENTATION STATUS

## 当前阶段结论

- Phase 01 已完成 review closeout，并被标记为 `accepted`。
- 当前阶段只完成骨架与门禁收口，不包含真实 `mysql.database.create` 执行。
- `ready_for_next_phase = true`
- `next_phase = phase-02`

## 已完成

- 已新增 [design_docs/10-implementation-alignment-v0.1.md](/Users/zqw/Desktop/Project/dba_ai_assistant/design_docs/10-implementation-alignment-v0.1.md)，把主链路、边界、门禁和代码切分方式固定下来。
- 已新增 [design_docs/22-phase-01-gap-analysis-v0.1.md](/Users/zqw/Desktop/Project/dba_ai_assistant/design_docs/22-phase-01-gap-analysis-v0.1.md)，对现有 Go skeleton 做 Phase 01 级盘点与 gap analysis。
- 已新增 [design_docs/23-phase-01-skeleton-delta-v0.1.md](/Users/zqw/Desktop/Project/dba_ai_assistant/design_docs/23-phase-01-skeleton-delta-v0.1.md)，记录本轮骨架与门禁补齐范围。
- 已保留 [design_docs/21-current-codebase-baseline-v0.1.md](/Users/zqw/Desktop/Project/dba_ai_assistant/design_docs/21-current-codebase-baseline-v0.1.md) 作为 Phase 00.5 基线证据；Phase 01 的 `22` 文档是补充，不替代 Phase 00.5 证据。
- 已建立 Go 模块化单体 skeleton：
  - `cmd/server/`
  - `internal/api/`
  - `internal/application/actionrequest/`
  - `internal/application/approval/`
  - `internal/application/authorization/`
  - `internal/application/execution/`
  - `internal/application/audit/`
  - `internal/application/evidence/`
  - `internal/domain/action/`
  - `internal/domain/asset/`
  - `internal/domain/principal/`
  - `internal/domain/order/`
  - `internal/domain/plan/`
  - `internal/domain/task/`
  - `internal/domain/policy/`
  - `internal/domain/risk/`
  - `internal/domain/authorization/`
  - `internal/adapters/dbnative/`
  - `internal/skill/`
  - `internal/persistence/`
- 已定义核心 interface / contract：
  - `ActionRequestService`
  - `PrincipalResolver`
  - `AssetResolver`
  - `PolicyEngine`
  - `RiskEngine`
  - `AuthorizationService`
  - `ExecuteAuthorizationService`
  - `ApprovalService`
  - `ExecutionPlanner`
  - `ExecutionRouter`
  - `TaskRuntime`
  - `AuditService`
  - `EvidenceService`
- 已定义基础 domain object / DTO / 错误模型，包括：
  - `ExecutePolicy`
  - `ApprovalPolicy.TTL`
  - role constants（含 `control_executor`）
  - skill contract structs（含 `trace_id` 映射）
- 已建立 northbound API skeleton：
  - `POST /api/v1/action-requests`
  - `GET /api/v1/orders/{order_id}`
  - `GET /api/v1/tasks/{task_id}`
  - `POST /api/v1/orders/{order_id}/approvals`
  - `POST /api/v1/orders/{order_id}/execute`
  - `GET /api/v1/audit-ledger/{request_id}`
  - `GET /api/v1/evidence-packs/{order_id}`
- 已建立 guardrails：
  - 统一动作入口走 `ActionRequestService`
  - `AuthorizationService` 是唯一授权出口
  - execute 走独立 `ExecuteAuthorizationService`
  - `AssetResolver` 只有 `ResolveExact`
  - 审批与执行物理分离
  - `AuditService` 已进入 request/query skeleton
  - `EvidenceService` 已进入 query/view skeleton 与 `trace_id` contract
  - southbound 只建立了 `DBNativeAdapter` skeleton
- 已补充 Phase 1 缺项：
  - all northbound / skill 输出 `trace_id` contract
  - `AdapterExecutionRequest.IdempotencyKey`
  - skill 映射 helper
  - approval route separation guardrail test
  - `DBNativeAdapter` skeleton / `DryRun` guardrail tests
  - `REQUEST_ACCEPTED` 首事件回归测试
- 已补最小测试并通过：
  - exact asset match guardrail
  - authorization aggregation guardrail
  - action request/execute separation guardrail
  - approval/execute physical separation guardrail
  - execute policy guardrail
  - `control_executor` execute allowlist guardrail
  - northbound / skill `trace_id` contract guardrail
  - audit ledger view trace propagation
  - evidence pack trace propagation
  - `DBNativeAdapter.DryRun(...)` stub preview
  - `DBNativeAdapter.Execute(...)` phase-one skeleton lock
  - adapter idempotency key format guardrail
  - skill contract mapping guardrail
  - API unified entry / error mapping guardrail

## 当前仍是 stub

- [internal/application/approval/noop_service.go](/Users/zqw/Desktop/Project/dba_ai_assistant/internal/application/approval/noop_service.go)
  - 审批 service 仅是占位，没有与共享 order store 打通。
- [internal/application/actionrequest/service.go](/Users/zqw/Desktop/Project/dba_ai_assistant/internal/application/actionrequest/service.go)
  - `Submit(...)` 已形成 request -> authorization -> order -> plan 的骨架。
  - `ExecuteApprovedOrder(...)` 现在已接入独立 execute auth 门禁。
  - 但仍不做真实 `re-validate -> router -> runtime` 启动。
- [internal/application/execution/stub_planner.go](/Users/zqw/Desktop/Project/dba_ai_assistant/internal/application/execution/stub_planner.go)
  - planner/router/runtime 仍是静态 stub。
- [internal/application/evidence/memory_service.go](/Users/zqw/Desktop/Project/dba_ai_assistant/internal/application/evidence/memory_service.go)
  - 当前只提供 query/build contract 的内存 skeleton；success/failure/`PLAN_STALE` 的真实写入时机仍未接线。
- [internal/application/audit/memory_service.go](/Users/zqw/Desktop/Project/dba_ai_assistant/internal/application/audit/memory_service.go)
  - 当前只提供最小内存事件流与 query view，不是 append-only 持久化实现。
- [internal/skill/contracts.go](/Users/zqw/Desktop/Project/dba_ai_assistant/internal/skill/contracts.go)
  - skill 输入输出 contract 已建立，但还没有单独的 Skill runtime / SDK 层。
- [internal/adapters/dbnative/adapter.go](/Users/zqw/Desktop/Project/dba_ai_assistant/internal/adapters/dbnative/adapter.go)
  - 仅实现 adapter contract、dry-run stub 和失败型 execute 返回，没有真实 MySQL 连接与 `CREATE DATABASE`。
- [internal/persistence/contracts.go](/Users/zqw/Desktop/Project/dba_ai_assistant/internal/persistence/contracts.go)
  - repository interface 已定义，但未在 application service 中全面接线。

## 还没有实现

- 完整 `mysql.database.create` 真实执行闭环
- execute 前真实 `Plan Re-validate`
- 基于持久化 `ExecutePolicy` 的真实校验
- 审批状态与 order 状态的共享持久化
- `ExecutionTask` 真正创建、推进与查询
- 审批 TTL 的真实过期扫描与状态收敛
- 成功/失败/`PLAN_STALE` 三条路径的真实 `EvidencePack` 固化
- append-only audit repository
- 基于 persistence repository 的统一存储层
- 多 adapter 路由策略

## 下一阶段准备做什么

1. 先把 `ActionRequestService` 从内部 map 存储切到 `internal/persistence` repository interface。
2. 打通 `ApprovalService` 与 `AssistantOrder` / `ExecutionPlan` 共享存储，落真实：
   - `WAITING_APPROVAL -> APPROVED`
   - `SELF_APPROVAL_FORBIDDEN`
   - `REJECTED / EXPIRED`
   - `ApprovalPolicy.TTL` 驱动的过期扫描
3. 打通 `ExecuteApprovedOrder(...)` 的最小闭环：
   - execute auth（基于独立 `ExecutePolicy`）
   - plan revalidate
   - `ExecutionRouter`
   - `TaskRuntime`
4. 在 `DBNativeAdapter` 中只补 `mysql.database.create` 的最小纵切，不扩散到其他动作。
5. 把审计与证据从当前内存 stub 升级为 append-only / 可查询的真实实现。
6. 补 `internal/application/approval/` 的直接测试文件。
7. 明确 `MemoryAuditService.GetViewByRequestID(...)` 在后续 execute / retry 场景下的 `trace_id` 策略。

## 当前架构风险与待确认点

- 目前 `ApprovalService` 与 `ActionRequestService` 还没有共享状态仓库，approval API 只是接口占位，不是可用审批闭环。
- `ExecuteApprovedOrder(...)` 还没有真正进入 `ExecutionTask`，因此 execute API 现在主要体现“独立 execute 门禁存在”，不是“执行已可用”。
- 当前 persistence 仍是 contract-first，尚未固化最终 repository shape；下一轮接线时要避免再把 application service 写回 internal map。
- 目前 northbound auth context 通过 HTTP header 占位承载，正式接入时需要替换成真实认证中间件，但不能改变 `PrincipalResolver` 是唯一身份装配入口这一原则。
- 当前 `AuditService` / `EvidenceService` 的 Phase 01 含义是“contract + query skeleton + trace contract 已固定”，不是“执行结束落账链路已完成”。
- review 文档里把 `APPROVAL_EXPIRED` 写成 order status，但正式 spec / interface design / schema 的权威语义是 order `EXPIRED` + audit event `APPROVAL_EXPIRED`。当前代码按正式文档实现，并已用 `// REVIEW:` 注释标明。
- 目前 `DBNativeAdapter` 的 stub 明确拒绝真实执行；下一轮实现时必须继续保持：
  - 不绕过 `AuthorizationService`
  - 不把 request 权限与 execute 权限混成一个策略
  - 不绕过审批 / execute 分离
  - 不在 `AssetResolver` 中引入 fuzzy match

## 验证记录

- reviewed branch `feat/p1-baseline-gap-and-guardrails` 已在 Claude review 中确认：`go test ./...` 全部通过
- 本轮 closeout 验证只覆盖文档与状态收口，不新增业务代码
