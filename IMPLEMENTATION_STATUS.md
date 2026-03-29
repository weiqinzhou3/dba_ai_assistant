# IMPLEMENTATION STATUS

## 当前阶段结论

- Phase 02 已完成 review closeout，并被标记为 `accepted`。
- 当前仓库状态已同步为：Phase 02 收口完成，准备进入 Phase 03。
- 当前系统已能通过 HTTP API 走通 request / approval / execute / audit / evidence 最小闭环。
- 当前系统仍不做真实 `mysql.database.create` 执行。
- `ready_for_next_phase = true`
- `next_phase = phase-03`
- `main_cleanup_verification = completed` — main 与 origin/main 完全对齐 (`547e23c`)，工作区干净，`go test ./...` 全部通过，适合作为 Phase 03 基线。

## 已完成

- 已新增 [design_docs/10-implementation-alignment-v0.1.md](/Users/zqw/Desktop/Project/dba_ai_assistant/design_docs/10-implementation-alignment-v0.1.md)，把主链路、边界、门禁和代码切分方式固定下来。
- 已新增 [design_docs/22-phase-01-gap-analysis-v0.1.md](/Users/zqw/Desktop/Project/dba_ai_assistant/design_docs/22-phase-01-gap-analysis-v0.1.md)，对现有 Go skeleton 做 Phase 01 级盘点与 gap analysis。
- 已新增 [design_docs/23-phase-01-skeleton-delta-v0.1.md](/Users/zqw/Desktop/Project/dba_ai_assistant/design_docs/23-phase-01-skeleton-delta-v0.1.md)，记录本轮骨架与门禁补齐范围。
- 已保留 [design_docs/21-current-codebase-baseline-v0.1.md](/Users/zqw/Desktop/Project/dba_ai_assistant/design_docs/21-current-codebase-baseline-v0.1.md) 作为 Phase 00.5 基线证据；Phase 01 的 `22` 文档是补充，不替代 Phase 00.5 证据。
- 已在最新 `main` 上重放 Phase 00.5 / Phase 01 closeout 文档，避免继续在旧 `docs/phase-01-closeout` PR 上处理冲突。
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
  - Phase 02 approval transition tests
  - Phase 02 execute revalidate / stale / idempotent execute tests
  - Phase 02 HTTP control-flow integration test

## Phase 02 已新增

- 已把 [internal/persistence/memory.go](/Users/zqw/Desktop/Project/dba_ai_assistant/internal/persistence/memory.go) 接成共享 in-memory persistence：
  - `ActionRequest`
  - `AssistantOrder`
  - `ExecutionPlan`
  - `ApprovalState`
  - `ExecutionTask`
  - `AuditEvent`
  - `EvidencePack`
- 已把 [internal/application/actionrequest/service.go](/Users/zqw/Desktop/Project/dba_ai_assistant/internal/application/actionrequest/service.go) 从内部 map 迁移到共享 store，并打通：
  - `ActionRequest` 提交
  - `AuthorizationDecision` 生成
  - `AssistantOrder` 创建
  - `ExecutionPlan` 冻结
  - `ApprovalService.Create(...)`
  - `ExecuteApprovedOrder(...)`
- 已新增 [internal/application/approval/service.go](/Users/zqw/Desktop/Project/dba_ai_assistant/internal/application/approval/service.go)，接通：
  - `WAITING_APPROVAL -> APPROVED`
  - `WAITING_APPROVAL -> REJECTED`
  - `WAITING_APPROVAL -> EXPIRED`
  - `SELF_APPROVAL_FORBIDDEN`
- 已升级 [internal/application/execution/stub_planner.go](/Users/zqw/Desktop/Project/dba_ai_assistant/internal/application/execution/stub_planner.go)：
  - 加入最小 `Plan Revalidate` 占位逻辑
  - `PLAN_STALE` 时阻断任务创建
  - `TaskRuntime` 仅创建 `RUNNING` task skeleton
- 已升级 [internal/application/audit/memory_service.go](/Users/zqw/Desktop/Project/dba_ai_assistant/internal/application/audit/memory_service.go) 与 [internal/application/audit/contracts.go](/Users/zqw/Desktop/Project/dba_ai_assistant/internal/application/audit/contracts.go)：
  - `AuditEvent` 采用 append-only 语义
  - `AuditLedgerView` 已可聚合 `latest_order_status` / `latest_task_id` / `latest_execution_summary`
- 已升级 [internal/application/evidence/memory_service.go](/Users/zqw/Desktop/Project/dba_ai_assistant/internal/application/evidence/memory_service.go) 与 [internal/application/evidence/contracts.go](/Users/zqw/Desktop/Project/dba_ai_assistant/internal/application/evidence/contracts.go)：
  - execute 成功起 task skeleton 后可查询最小 evidence
  - `PLAN_STALE` 路径会生成 `task_id = null` 的 failure evidence
- 已保留关键 guardrails：
  - `AuthorizationService` 仍是 request 路径唯一最终授权出口
  - execute 仍然必须显式触发
  - approval 与 execute 仍然分离
  - `AssetResolver` 仍然 exact match
  - `DBNativeAdapter` 仍不做真实 `CREATE DATABASE`
- 已新增 Phase 02 文档：
  - [design_docs/24-phase-02-control-flow-notes-v0.1.md](/Users/zqw/Desktop/Project/dba_ai_assistant/design_docs/24-phase-02-control-flow-notes-v0.1.md)
  - [design_docs/25-phase-02-manual-api-runbook-v0.1.md](/Users/zqw/Desktop/Project/dba_ai_assistant/design_docs/25-phase-02-manual-api-runbook-v0.1.md)

## 当前仍是 stub

- [internal/application/approval/noop_service.go](/Users/zqw/Desktop/Project/dba_ai_assistant/internal/application/approval/noop_service.go)
  - 旧 Phase 01 noop 仍保留在仓库中，但主程序已改用新的 repo-backed `ApprovalService`。
- [internal/application/actionrequest/service.go](/Users/zqw/Desktop/Project/dba_ai_assistant/internal/application/actionrequest/service.go)
  - request / execute 主链路已打通。
  - 但 `execute` 仍只起 task skeleton，不调用真实 adapter execute。
- [internal/application/execution/stub_planner.go](/Users/zqw/Desktop/Project/dba_ai_assistant/internal/application/execution/stub_planner.go)
  - planner/router/runtime 仍是静态 stub，`Revalidate(...)` 只做最小占位检查。
- [internal/application/evidence/memory_service.go](/Users/zqw/Desktop/Project/dba_ai_assistant/internal/application/evidence/memory_service.go)
  - 当前 evidence 语义仍是 Phase 02 最小控制流证据，不是最终执行证据模型。
- [internal/application/audit/memory_service.go](/Users/zqw/Desktop/Project/dba_ai_assistant/internal/application/audit/memory_service.go)
  - 当前仍是内存型 append-only 审计，不是持久化数据库实现。
- [internal/skill/contracts.go](/Users/zqw/Desktop/Project/dba_ai_assistant/internal/skill/contracts.go)
  - skill 输入输出 contract 已建立，但还没有单独的 Skill runtime / SDK 层。
- [internal/adapters/dbnative/adapter.go](/Users/zqw/Desktop/Project/dba_ai_assistant/internal/adapters/dbnative/adapter.go)
  - 仅实现 adapter contract、dry-run stub 和失败型 execute 返回，没有真实 MySQL 连接与 `CREATE DATABASE`。
- [internal/persistence/contracts.go](/Users/zqw/Desktop/Project/dba_ai_assistant/internal/persistence/contracts.go)
  - 仍是内存 contract + store，不是数据库级持久化仓储。

## 还没有实现

- 完整 `mysql.database.create` 真实执行闭环
- execute 前真实 `Plan Re-validate`
- 基于持久化 `ExecutePolicy` 的真实校验
- approval actor 从认证上下文解析并做审批角色校验
- `ExecutionTask` 真正推进到 terminal state
- 审批 TTL 的后台调度与真实周期扫描
- 成功/失败终态的真实 `EvidencePack` 固化
- 持久化数据库版 append-only audit repository
- 持久化数据库版 unified repository
- 多 adapter 路由策略

## 下一阶段准备做什么

1. 在 Phase 03 把 `TaskRuntime` 从 skeleton 升级为真实 southbound 执行。
2. 在 `DBNativeAdapter` 中只补 `mysql.database.create` 的最小纵切，不扩散到其他动作。
3. 把 execute policy 与 approval actor 校验从静态/请求体模式升级到正式认证上下文与策略求值。
4. 把当前内存型 audit / evidence / repository 切到持久化实现。

## 当前架构风险与待确认点

- 当前 execute 成功路径的 `EvidencePack.execution_success=true` 表示“控制链路成功启动 task skeleton”，不是“数据库已创建成功”；Phase 03 需要把这个语义切回真实终态。
- approval API 当前仍通过 body 传 `approver_id`，正式鉴权语义还未收口到认证上下文。
- 当前 persistence 是共享内存 store，适合 Phase 02 演示与测试，但不具备重启恢复能力。
- 目前 northbound auth context 通过 HTTP header 占位承载，正式接入时需要替换成真实认证中间件，但不能改变 `PrincipalResolver` 是唯一身份装配入口这一原则。
- review 文档里把 `APPROVAL_EXPIRED` 写成 order status，但正式 spec / interface design / schema 的权威语义是 order `EXPIRED` + audit event `APPROVAL_EXPIRED`。当前代码按正式文档实现，并已用 `// REVIEW:` 注释标明。
- 目前 `DBNativeAdapter` 的 stub 明确拒绝真实执行；下一轮实现时必须继续保持：
  - 不绕过 `AuthorizationService`
  - 不把 request 权限与 execute 权限混成一个策略
  - 不绕过审批 / execute 分离
  - 不在 `AssetResolver` 中引入 fuzzy match

## 验证记录

- reviewed branch `feat/p1-baseline-gap-and-guardrails` 已在 Claude review 中确认：`go test ./...` 全部通过
- `docs/phase-01-closeout-v2` 已于 2026-03-29 fresh 执行 `go test ./...` 并通过
- `feat/p2-min-control-flow-v2` 已获 Claude review `PASS`，允许进入 Phase 03
- `go test ./...`
- 本地 HTTP smoke 已验证：
  - prod: `WAITING_APPROVAL -> APPROVED -> EXECUTING`
  - stale: `APPROVED -> PLAN_STALE`
  - `GET /api/v1/audit-ledger/{request_id}` 可查询最小闭环
  - `GET /api/v1/evidence-packs/{order_id}` 可查询最小闭环
