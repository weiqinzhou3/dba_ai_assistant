# Phase 01 Gap Analysis v0.1

## 0. 文档定位

本文档用于回答一个 Phase 01 级别的收口问题：

> 基于当前已有 Go skeleton，哪些关键 guardrail 已经成立，哪些仍需在本轮补齐，哪些必须明确留到 Phase 02？

本文档不宣布 Phase 02/03 已完成，只记录 2026-03-29 对当前代码与正式文档 `03~08` 的对照结果。

---

## 1. 盘点结论

结论如下：

1. 当前仓库已经具备 Phase 01 所需的大部分 skeleton，不能按“空仓”处理。
2. `AuthorizationService`、`ResolveExact(...)`、独立 execute auth、单一 `DBNativeAdapter`、`DryRun(...)` contract 均已存在。
3. 本轮真正需要补的是“Phase 01 级契约收口”，不是 Phase 02 的最小控制闭环。
4. 最明显的代码 gap 是 northbound / skill 返回对象上的 `trace_id` 契约还没有全覆盖，以及部分关键 guardrail 还缺测试锁定。

---

## 2. 证据范围

本次对照使用：

1. `design_docs/03-assistant-spec-v0.7.md`
2. `design_docs/04-interface-design-v0.8.md`
3. `design_docs/05-control-layer-schema-v0.1.md`
4. `design_docs/06-rbac-model-v0.1.md`
5. `design_docs/07-adapter-interface-v0.1.md`
6. `design_docs/08-mysql-database-create-sequence.md`
7. `design_docs/phases/phase-01-skeleton-guardrails.md`
8. 当前 `cmd/` + `internal/` Go 代码树

当前代码树统计：

1. Go 文件共 43 个。
2. Go 测试文件共 11 个。

---

## 3. 关键检查项结论

### 3.1 AuthorizationService 是否仍是唯一最终授权出口

结论：`aligned`

证据：

1. `internal/application/actionrequest/service.go` 只依赖 `AuthorizationService`，没有自行拼 `PolicyDecision + RiskDecision`。
2. `internal/application/authorization/authorization_service.go` 统一合并 Policy 与 Risk。
3. 现有 `internal/application/authorization/authorization_service_test.go` 已锁定 deny/approval-required 的聚合语义。

本轮动作：

1. 不改模型，只在文档中确认该 guardrail 已成立。

### 3.2 AssetResolver 是否仍然只支持 exact match

结论：`aligned`

证据：

1. `internal/application/authorization/contracts.go` 仅暴露 `ResolveExact(...)`。
2. `internal/application/authorization/exact_asset_resolver.go` 只做精确匹配。
3. `internal/application/authorization/exact_asset_resolver_test.go` 已覆盖 not found / ambiguous / case mismatch。

本轮动作：

1. 不新增 fuzzy / alias / best-effort 接口。

### 3.3 Approval 与 Execute 是否物理分离

结论：`aligned_with_guardrail_test_added`

证据：

1. `internal/api/server.go` 已分离：
   - `POST /api/v1/orders/{order_id}/approvals`
   - `POST /api/v1/orders/{order_id}/execute`
2. `ExecuteApprovedOrder(...)` 仍只在 execute 路径下调用。

本轮补齐：

1. 新增 API 测试，锁定 approval 路由不会误触 execute 路径。

边界说明：

1. `ApprovalService` 仍是 noop skeleton。
2. 共享 order/approval 状态接线属于 Phase 02，不在本轮推进。

### 3.4 AuditService / EvidenceService 是否已进入主链路骨架

结论：`partial_phase_01_aligned`

已成立部分：

1. `AuditService` / `EvidenceService` interface 已存在。
2. query API skeleton 已存在：
   - `GET /api/v1/audit-ledger/{request_id}`
   - `GET /api/v1/evidence-packs/{order_id}`
3. `Submit(...)` 已写 `REQUEST_ACCEPTED` / `AUTHORIZATION_DECIDED` / `ORDER_CREATED` / `PLAN_FROZEN` 审计事件。

未在本轮推进的部分：

1. execute 前 `PLAN_STALE` 审计/证据写入。
2. success / failure evidence build。
3. append-only repository 与共享状态接线。

本轮补齐：

1. 给 audit / evidence query 视图补上 `trace_id` contract。
2. 给内存 audit / evidence skeleton 补最小测试，锁定 Phase 01 查询契约。

边界说明：

1. evidence 的真实 build 链路仍属于 Phase 02。
2. 本轮不把“query skeleton 已存在”误报成“执行结束证据链已完成”。

### 3.5 DBNativeAdapter 是否仍只是 skeleton

结论：`aligned_with_guardrail_test_added`

证据：

1. `internal/adapters/dbnative/adapter.go` 的 `Execute(...)` 明确返回 `NOT_IMPLEMENTED_IN_PHASE_1`。
2. 当前未连接真实 MySQL。
3. 当前未进入真实 `CREATE DATABASE`。

本轮补齐：

1. 新增 adapter 测试，锁定：
   - `DryRun(...)` 仍是 skeleton preview
   - `Execute(...)` 仍明确失败，不允许误报为真实执行

### 3.6 DryRun contract 是否已进入正确位置

结论：`aligned_with_guardrail_test_added`

证据：

1. `internal/application/execution/contracts.go` 的 `Adapter` interface 已包含 `DryRun(...)`。
2. `AdapterDryRunResult` 已存在于同一 contract 文件中。
3. `DBNativeAdapter` 已实现 `DryRun(...)`。

本轮补齐：

1. 新增 adapter test，锁定 `DryRun` 的 Phase 01 stub 语义。

### 3.7 northbound / skill trace_id contract 是否完整

结论：`gap_fixed_in_this_round`

原始问题：

1. `ActionSubmissionResult` / `ExecuteOrderResult` 已有 `trace_id`。
2. 但 `AssistantOrderView`、`ExecutionTaskView`、`ApprovalState`、`AuditLedgerView`、`EvidencePackView`、skill output mapping 并未完整覆盖。

本轮补齐：

1. `AssistantOrder` 增加 `trace_id`。
2. `ExecutionTask` 增加 `trace_id`。
3. `approval.State` 增加 `trace_id`。
4. `audit.LedgerView` 增加 `trace_id`。
5. `evidence.Pack` / `BuildInput` 增加 `trace_id`。
6. skill output mapping 增加 `trace_id`。
7. 新增 server / audit / evidence / actionrequest / skill 测试锁定。

---

## 4. 本轮应做与不应做

本轮应做：

1. 盘点已有 skeleton。
2. 补 Phase 01 级 contract 缺口。
3. 补 guardrail tests。
4. 更新 dashboard / status / handoff / implementation status。

本轮不应做：

1. 不实现真实 `mysql.database.create`。
2. 不把 `ApprovalService` 接成共享状态机。
3. 不把 `ExecuteApprovedOrder(...)` 推进到 `Plan Revalidate -> TaskRuntime -> Evidence Build`。
4. 不接 Deep Agent / MCP / 多 adapter。

---

## 5. 最终判断

最终判断：

1. 当前仓库适合继续作为 Phase 01 基线。
2. 本轮应把重点放在“contract + trace + guardrail test + 文档收口”。
3. approval 共享状态、execute revalidate、`PLAN_STALE` 证据固化仍留在 Phase 02。

一句话结论：

> Phase 01 不是重写骨架，而是对已有 skeleton 做 gap analysis、contract 收口和 guardrail 补强。
