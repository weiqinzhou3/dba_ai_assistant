# Phase 02 Control Flow Notes v0.1

## 文档作用

记录 Phase 02 最小控制链路的关键实现取舍，避免后续 Phase 03 在真实执行接入时误判当前能力边界。

## 本轮落地的最小闭环

当前系统已经可以通过 HTTP API 手工走通：

1. `ActionRequest` 提交
2. `AuthorizationDecision` 生成
3. `AssistantOrder` 创建
4. `ExecutionPlan` 冻结
5. `Approval` 决策
6. 显式 `execute`
7. `Plan Revalidate`
8. `ExecutionTask` 骨架创建
9. `AuditLedger` / `EvidencePack` 查询

但本轮仍然明确停留在控制流接线，不进入真实 MySQL 写操作。

## 关键实现说明

### 1. 共享 in-memory persistence

- `internal/persistence/MemoryStore` 现在承接 request / order / plan / approval state / task / audit / evidence 的共享内存存储。
- 由于 Go 不支持单个 type 以不同签名同时实现多组 `Save(...)` / `Get(...)` 方法，本轮把 repository contract 改为具名方法：
  - `SaveRequest(...)`
  - `SaveOrder(...)`
  - `SavePlan(...)`
  - `SaveTask(...)`
  - `SaveApprovalState(...)`
  - `AppendAuditEvent(...)`
  - `SaveEvidencePack(...)`
- 这个改动只是为了让单个 `MemoryStore` 可以成为 Phase 02 的共享状态源，不改变后续是否替换为真实持久化层。

### 2. Authorization 与 execute 边界

- request 路径的最终授权结论仍然只来自 `AuthorizationService.Evaluate(...)`。
- execute 路径仍然走独立 `ExecuteAuthorizationService.Authorize(...)`。
- approval 通过不会自动执行；执行仍然只能通过 `POST /api/v1/orders/{order_id}/execute` 显式触发。
- `AssetResolver` 仍只保留 `ResolveExact(...)`；没有新增 fuzzy / alias 匹配。

### 3. Approval 最小状态机

- 已接线 `WAITING_APPROVAL -> APPROVED`
- 已接线 `WAITING_APPROVAL -> REJECTED`
- 已接线 `WAITING_APPROVAL -> EXPIRED`
- 已接线 `SELF_APPROVAL_FORBIDDEN`
- 审批 TTL 明确来自 `ApprovalPolicy.TTL`，本轮通过内存 seeded policy 提供，不走隐式常量。

### 4. Plan Revalidate 最小占位逻辑

- `StaticExecutionPlanner.Revalidate(...)` 现在至少检查：
  - plan 未提前标记为 `STALE`
  - plan snapshot 仍为 frozen
  - order 与 plan 的 `plan_version` 一致
- 为了让手工 API 演示能覆盖 `PLAN_STALE`，本轮加了一个显式占位开关：
  - 在 `ActionRequest.request_context` 里传 `simulate_plan_stale=true`
  - execute 时会把 stale reason 注入到 revalidate 输入
  - 结果仍走正式的 `PLAN_STALE` 阻断路径
- 这是 Phase 02 的演示/验证占位逻辑，不代表正式 revalidate 最终形态。

### 5. Audit / Evidence 最小语义

- `AuditEvent` 已变为 append-only 事件流，并补了：
  - `event_id`
  - `created_at`
  - `approval_actor_id`
  - `execute_actor_id`
  - `success`
- `AuditLedgerView` 现在聚合出：
  - `latest_order_id`
  - `latest_task_id`
  - `latest_order_status`
  - `latest_approval_status`
  - `latest_execution_summary`
  - `event_count`
  - `last_event_at`
- `EvidencePack` 已支持两条最小路径：
  - execute 成功起任务骨架后的控制流证据
  - `PLAN_STALE` 失败证据
- `PLAN_STALE` 证据明确满足：
  - `task_id = null`
  - `execution_success = false`
  - `failure_detail.reason != ""`

### 6. Task / adapter 边界

- `ExecutionRouter` 仍固定路由到 `db_native`。
- `TaskRuntime` 只创建 `RUNNING` 状态的任务骨架，不触发真实 adapter execute。
- `DBNativeAdapter` 继续只保留 `Supports` / `DryRun` / skeleton `Execute`，不做真实 `CREATE DATABASE`。

## 当前明确未做

- 未实现真实 `mysql.database.create`
- 未接 Deep Agent
- 未接 MCP
- 未引入多个 adapter
- 未把 execute 回塞到 `action-requests`
- 未把 execute policy 切到持久化策略求值
- 未实现真实 terminal task runtime 与成功/失败最终态

## 对 Phase 03 的直接影响

1. 可以在不重写 northbound API 的前提下，把 `TaskRuntime` 从骨架切到真实执行。
2. 可以在不打破 approval / execute 分离的前提下，把 `DBNativeAdapter` 从 stub 升级为真实 MySQL 纵切。
3. `PLAN_STALE`、`APPROVED`、`EXECUTING` 的状态推进与审计语义已经固定，后续应在此基础上扩展，而不是回退到“一个 request 自动全跑完”的隐式模型。
