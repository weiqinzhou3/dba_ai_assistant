# IMPLEMENTATION STATUS

## 已完成

- 已新增 [design_docs/10-implementation-alignment-v0.1.md](/Users/zqw/Desktop/Project/dba_ai_assistant/design_docs/10-implementation-alignment-v0.1.md)，把本轮主链路、边界、门禁和代码切分方式固定下来。
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
  - skill contract structs
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
  - `AuditService` / `EvidenceService` 已进入主链路骨架
  - southbound 只建立了 `DBNativeAdapter` skeleton
- 已补充 Phase 1 缺项：
  - `AdapterExecutionRequest.IdempotencyKey`
  - skill 映射 helper
  - `REQUEST_ACCEPTED` 首事件回归测试
- 已补最小测试并通过：
  - exact asset match guardrail
  - authorization aggregation guardrail
  - action request/execute separation guardrail
  - execute policy guardrail
  - `control_executor` execute allowlist guardrail
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
- [internal/skill/contracts.go](/Users/zqw/Desktop/Project/dba_ai_assistant/internal/skill/contracts.go)
  - skill 输入输出 contract 已建立，但还没有单独的 Skill runtime / SDK 层。
- [internal/adapters/dbnative/adapter.go](/Users/zqw/Desktop/Project/dba_ai_assistant/internal/adapters/dbnative/adapter.go)
  - 仅实现 adapter contract 和失败型 stub 返回，没有真实 MySQL 连接与 `CREATE DATABASE`。
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

## 下一轮最合理的工作

1. 先把 `ActionRequestService` 从内部 map 存储切到 `internal/persistence` repository interface。
2. 打通 `ApprovalService` 与 `AssistantOrder` / `ExecutionPlan` 共享存储，落真实：
   - `WAITING_APPROVAL -> APPROVED`
   - `SELF_APPROVAL_FORBIDDEN`
   - `REJECTED / EXPIRED`
   - `ApprovalPolicy.TTL` 驱动的过期扫描
3. 打通 `ExecuteApprovedOrder(...)` 的最小闭环：
   - execute auth（基于独立 `ExecutePolicy`）
   - plan re-validate
   - `ExecutionRouter`
   - `TaskRuntime`
4. 在 `DBNativeAdapter` 中只补 `mysql.database.create` 的最小纵切，不扩散到其他动作。
5. 把审计与证据从当前内存 stub 升级为 append-only / 可查询的真实实现。

## 当前架构风险与待确认点

- 目前 `ApprovalService` 与 `ActionRequestService` 还没有共享状态仓库，approval API 只是接口占位，不是可用审批闭环。
- `ExecuteApprovedOrder(...)` 还没有真正进入 `ExecutionTask`，因此 execute API 现在主要体现“独立 execute 门禁存在”，不是“执行已可用”。
- 当前 persistence 仍是 contract-first，尚未固化最终 repository shape；下一轮接线时要避免再把 application service 写回 internal map。
- 目前 northbound auth context 通过 HTTP header 占位承载，正式接入时需要替换成真实认证中间件，但不能改变 `PrincipalResolver` 是唯一身份装配入口这一原则。
- review 文档里把 `APPROVAL_EXPIRED` 写成 order status，但正式 spec / interface design / schema 的权威语义是 order `EXPIRED` + audit event `APPROVAL_EXPIRED`。当前代码按正式文档实现，并已用 `// REVIEW:` 注释标明。
- 目前 `DBNativeAdapter` 的 stub 明确拒绝真实执行；下一轮实现时必须继续保持：
  - 不绕过 `AuthorizationService`
  - 不把 request 权限与 execute 权限混成一个策略
  - 不绕过审批 / execute 分离
  - 不在 `AssetResolver` 中引入 fuzzy match

## 验证记录

- `gofmt -w $(rg --files -g '*.go')`
- `go test ./...`
