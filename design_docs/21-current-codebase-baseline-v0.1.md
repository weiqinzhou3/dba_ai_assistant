# Current Codebase Baseline v0.1

## 0. 文档定位

本文档用于固化 Phase 00.5 的当前代码基线判断，回答一个明确问题：

> 后续 Phase 01 应该从零重写，还是基于当前仓库已有 Go skeleton 继续推进？

本文档不新增任何主链路语义，只记录当前仓库中的可见事实，作为 Phase 00.5 放行证据的一部分。

---

## 1. 结论

结论明确如下：

1. 当前仓库已经存在可编译的 Go 模块化单体 skeleton，不是空仓。
2. 当前仓库已经覆盖大量 Phase 01 级别的对象、接口、API skeleton、adapter stub 和 guardrail tests。
3. 因此，Phase 01 不应从零重写，而应以“已有产物盘点 + gap analysis + 验收补齐”为起点。
4. 当前仓库仍未进入真实可执行控制链路，不能跳过 Phase 01 / Phase 02 直接进入 `mysql.database.create` 真实执行。

一句话基线判断：

> 当前代码库适合作为 Phase 01 的继续推进基线，不适合作为“从零重新开工”的依据。

---

## 2. 证据来源

本结论基于 2026-03-29 对当前仓库的直接盘点：

1. `find cmd internal -type f | sort`
2. `find internal cmd -type f -name '*.go' | wc -l`
3. `find internal cmd -type f -name '*_test.go' | wc -l`
4. `rg -n "DryRun\\(|ResolveExact|control_executor|PLAN_STALE|request_mysql_database_create|execute_assistant_order|ApprovalPolicy|EvidencePack|AuditEvent" cmd internal design_docs -g'*.go'`
5. `IMPLEMENTATION_STATUS.md`

盘点结果摘要：

1. 当前 `cmd/` + `internal/` 下共有 40 个 Go 文件。
2. 当前共有 8 个 Go 测试文件。
3. 代码树已经覆盖 `api`、`application`、`domain`、`persistence`、`skill`、`adapters/dbnative`。

---

## 3. 已存在的实现骨架

### 3.1 目录与模块

当前已存在以下核心目录：

- `cmd/server/`
- `internal/api/`
- `internal/application/actionrequest/`
- `internal/application/approval/`
- `internal/application/audit/`
- `internal/application/authorization/`
- `internal/application/evidence/`
- `internal/application/execution/`
- `internal/domain/action/`
- `internal/domain/asset/`
- `internal/domain/authorization/`
- `internal/domain/common/`
- `internal/domain/order/`
- `internal/domain/plan/`
- `internal/domain/policy/`
- `internal/domain/principal/`
- `internal/domain/risk/`
- `internal/domain/task/`
- `internal/persistence/`
- `internal/skill/`
- `internal/adapters/dbnative/`

### 3.2 已存在的关键 contract / guardrail

当前仓库已经能直接看到以下 Phase 01 关键结构：

1. `AuthorizationService` 已存在，且作为 request 路径的权威授权出口。
2. `ExecuteAuthorizationService` 已存在，execute 路径未与 request 权限混写。
3. `AssetResolver` contract 只暴露 `ResolveExact(...)`。
4. Adapter SPI 已包含 `DryRun(...)`。
5. `DBNativeAdapter` 已有 skeleton，且当前仍是 stub。
6. role 常量中已包含 `control_executor`。
7. order status 中已包含 `PLAN_STALE`。
8. `ApprovalPolicy` 已带 TTL 语义。
9. skill contract 已包含：
   - `request_mysql_database_create`
   - `execute_assistant_order`
10. `AuditEvent`、`EvidencePack` contract 已存在。
11. adapter 请求中已带 `IdempotencyKey`。

### 3.3 已存在的 northbound / southbound 骨架

当前仓库已存在的 northbound API skeleton 包括：

1. `POST /api/v1/action-requests`
2. `GET /api/v1/orders/{order_id}`
3. `GET /api/v1/tasks/{task_id}`
4. `POST /api/v1/orders/{order_id}/approvals`
5. `POST /api/v1/orders/{order_id}/execute`
6. `GET /api/v1/audit-ledger/{request_id}`
7. `GET /api/v1/evidence-packs/{order_id}`

当前 southbound 仅存在：

1. `DBNativeAdapter` skeleton

未发现多 adapter 并行或 MCP 通道已接入的事实。

---

## 4. 已存在的测试基线

当前已存在 8 个 Go 测试文件，覆盖以下主题：

1. API unified entry / error mapping
2. action request service skeleton
3. authorization aggregation
4. exact asset resolver guardrail
5. execute authorization guardrail
6. idempotency key format
7. policy model fields
8. skill contract mapping

这说明当前仓库不是“只有接口没有验证”的状态，但这些测试仍主要属于 Phase 01 级 guardrail 验证，不代表 Phase 02/03 链路已经完成。

---

## 5. 当前仍是 stub 的部分

以下内容仍不能被当作“已完成”：

1. `ApprovalService` 仍是占位实现，尚未和共享 order / plan 状态打通。
2. `ActionRequestService.ExecuteApprovedOrder(...)` 还没有形成真实 `re-validate -> router -> runtime` 执行闭环。
3. `DBNativeAdapter` 尚未连接真实 MySQL，也没有真实 `mysql.database.create`。
4. `persistence` 仍以 contract-first 为主，application service 尚未全面接线。
5. append-only audit repository 尚未形成真实实现。
6. `EvidencePack` 仍未覆盖成功 / 失败 / `PLAN_STALE` 的真实固化链路。
7. `ExecutionTask` 还没有真实创建、推进与查询闭环。

---

## 6. 与 Phase 01 的关系

基于上述事实，Phase 01 的正确进入方式应为：

1. 先盘点已有实现与测试。
2. 按 Phase 01 验收标准逐条比对是否缺项。
3. 只补骨架、contract、guardrail 和缺失测试，不提前进入真实 MySQL 执行。
4. 不把当前已有 skeleton 误报为“Phase 02 已完成”或“Phase 03 已就绪”。

因此，Phase 01 的定位应是：

> 对已有代码基线做结构化收口与门禁补齐，而不是重新搭一套新的骨架。

---

## 7. Phase 00.5 放行结论

本基线文档支持以下 Phase 00.5 closeout 结论：

1. “新路线图基于现有代码继续推进”这一判断成立。
2. Phase 01 可以开始，但第一步必须是盘点已有产物并做 gap analysis。
3. 当前仍不得进入真实 `mysql.database.create` 执行开发。
4. 当前不得把 Phase 01 的“ready to start”误写成“already implemented”。

最终结论：

> Phase 00.5 可以关闭；Phase 01 可以开始，但必须从基线盘点与门禁补齐开始。
