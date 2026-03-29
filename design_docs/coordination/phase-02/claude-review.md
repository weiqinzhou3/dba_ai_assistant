# Phase 02 - Claude Review

## 文档作用

由 Claude Code 填写，记录对本轮 handoff 的评审结论和 gate 判断。

## Review 结论

- 版本：**final closeout**
- 总结：**PASS**
- Gate 决定：**允许进入 Phase 03。**

---

## 审查范围

本次审查覆盖：

1. 分支 `feat/p2-min-control-flow-v2` 相对 `main` 的全部 diff（PR #5，+2500/-490，25 files changed）
2. 正式文档 `design_docs/03~08`
3. Phase 02 规范 `design_docs/phases/phase-02-min-control-flow.md`
4. Codex 协调文档 `codex-plan.md` / `codex-status.md` / `codex-handoff.md`
5. 新增文档 `24-phase-02-control-flow-notes-v0.1.md` / `25-phase-02-manual-api-runbook-v0.1.md`
6. `IMPLEMENTATION_STATUS.md`
7. 所有改动的 Go 源码与测试
8. `go test ./...` 执行结果（12 个测试包全部通过）

---

## 8 项重点审查

### 1. 是否真正实现了最小控制链路可跑通

**结论：是，已实现。**

证据：

- `ActionRequestService.Submit(...)` 现在完整走通：request 保存 → principal 解析 → asset 精确匹配 → 授权决策 → order 创建 → plan 冻结 → approval 状态创建 → 审计事件写入。所有对象通过共享 `MemoryStore` 持久化。
- `ApprovalService` 已实现真实状态机：`WAITING_APPROVAL -> APPROVED / REJECTED / EXPIRED`，且与 `AssistantOrder` 共享状态同步更新。
- `ExecuteApprovedOrder(...)` 已完整接线：execute auth → plan revalidate → router → task runtime → order 状态推进 → audit/evidence 写入。
- `TestServerSupportsManualControlFlowOverHTTP` 是一个端到端 HTTP 集成测试，覆盖 submit(prod) → approval → execute → audit ledger 查询 → evidence pack 查询的完整链路。
- `TestServiceExecuteApprovedOrderStartsTaskAndWritesEvidence` 验证 execute 后 order 状态变为 `EXECUTING`、task 被创建、audit ledger 包含完整事件链、evidence pack 包含 task_id 和 trace_id。
- 本地 HTTP smoke 已验证 prod 路径和 stale 路径。

### 2. 是否仍未越界到真实 mysql.database.create 执行

**结论：未越界。**

证据：

- `internal/adapters/dbnative/adapter.go` 和 `internal/adapters/dbnative/adapter_test.go` 在 Phase 02 diff 中 **零改动**。
- `DBNativeAdapter.Execute(...)` 继续返回 `NOT_IMPLEMENTED_IN_PHASE_1`。
- `NoopTaskRuntime.Start(...)` 只创建 `RUNNING` 状态的 task skeleton，不调用任何 adapter execute 方法。
- `go.mod` 未新增 MySQL driver 依赖。
- `IMPLEMENTATION_STATUS.md` 明确声明："execute 仍只起 task skeleton，不调用真实 adapter execute"。

### 3. AuthorizationService 是否仍是唯一最终授权出口

**结论：是。**

证据：

- `internal/application/actionrequest/service.go:100-106`（Submit 路径）仍然只通过 `s.authorization.Evaluate(...)` 获取授权决策。
- `ApprovalService.Decide(...)` 不做任何授权判断，只推进审批状态。
- `ExecuteApprovedOrder(...)` 中的 execute auth 走独立的 `s.executeAuth.Authorize(...)`（`service.go:250-260`），与 request 授权物理隔离。
- `RiskEngine.Evaluate(...)` 被 `AuthorizationService` 统一编排，调用方不能绕过（`stub_resolvers.go` 的 `StaticRiskEngine` 只被 `AuthorizationService` 的 constructor 接受）。
- 新增 `risk_engine_test.go` 验证了 prod → R2 + REQUIRE_APPROVAL 和 high sensitivity → REQUIRE_APPROVAL 两条规则。

### 4. Approval 与 Execute 是否仍分离

**结论：是，且物理分离更加明确。**

证据：

- `internal/api/server.go` 保持独立路由：
  - `POST /api/v1/orders/{order_id}/approvals` → `handleDecideApproval` → `s.approvals.Decide(...)`
  - `POST /api/v1/orders/{order_id}/execute` → `handleExecuteOrder` → `s.actionRequests.ExecuteApprovedOrder(...)`
- Phase 01 的 `TestServerRoutesApprovalsWithoutCallingExecute` 继续存在并通过。
- `ApprovalService.Decide(...)` 把 order 状态推到 `APPROVED`，但不触发任何 execute 逻辑。execute 必须通过独立 API 显式触发。
- `ExecuteApprovedOrder(...)` 第一步检查 order 状态，`WAITING_APPROVAL` 直接返回 `APPROVAL_REQUIRED` 错误。
- `TestServiceExecuteApprovedOrderRejectsWaitingApprovalOrder` 锁定此行为。

### 5. AssetResolver 是否仍是 exact match

**结论：是。**

证据：

- `internal/application/authorization/contracts.go` 仍只暴露 `ResolveExact(...)`。
- Phase 02 diff 未触碰 `AssetResolver` interface。
- `stub_resolvers.go` 的 `InMemoryExactAssetResolver` 未新增 fuzzy/alias 方法。
- `Asset` 结构新增了 `Sensitivity` 字段（`internal/domain/asset/types.go`），但这是资产元数据扩展，不是搜索方式变更。

### 6. Audit / Evidence 是否已经形成最小查询闭环

**结论：是。**

证据：

- **Audit**:
  - `AuditEvent` 已升级为 append-only 语义，新增 `event_id`、`created_at`、`approval_actor_id`、`execute_actor_id`、`success` 字段。
  - `AuditLedgerView` 已可聚合 `latest_order_status` / `latest_task_id` / `latest_execution_summary` / `event_count` / `last_event_at`。
  - submit → authorization → order → plan → approval → execute → evidence 全链路均有对应审计事件。
  - `REQUEST_ACCEPTED` 仍是第一条审计事件（验证标准 7）。
  - `TestServerSupportsManualControlFlowOverHTTP` 验证了 `GET /api/v1/audit-ledger/{request_id}` 返回完整 trace_id、latest_order_status、latest_task_id。

- **Evidence**:
  - execute 成功路径：生成 `task_id != ""` 的成功证据包。
  - `PLAN_STALE` 路径：生成 `task_id = ""` 且 `execution_success = false` 的失败证据包，failure_detail 包含 stale reason。
  - `TestServiceExecuteApprovedOrderMarksPlanStaleWithoutCreatingTask` 验证了 stale 路径的 evidence 满足 `task_id = null` + `execution_success = false` + `failure_detail.reason != ""`。
  - `GET /api/v1/evidence-packs/{order_id}` 在集成测试中已验证可查。

### 7. DBNativeAdapter 是否仍只停留在 stub / dry-run 占位层

**结论：是。**

证据：

- Phase 02 diff 对 `internal/adapters/dbnative/` 目录零改动。
- `Execute(...)` 仍返回 `NOT_IMPLEMENTED_IN_PHASE_1`。
- `DryRun(...)` 仍返回 `dry_run_stub`。
- Phase 01 的 adapter guardrail tests 继续通过。

### 8. 是否允许进入 Phase 03

**结论：允许进入 Phase 03。**

依据（对照 Phase 02 验收标准逐项确认）：

| # | 验收标准 | 状态 | 证据 |
|---|---|---|---|
| 1 | dev/test 场景可走到 APPROVED 并通过 execute 进入任务骨架 | PASS | `TestServiceExecuteApprovedOrderStartsTaskAndWritesEvidence` 使用 test 环境 |
| 2 | prod 场景可走到 WAITING_APPROVAL -> APPROVED -> explicit execute | PASS | `TestServerSupportsManualControlFlowOverHTTP` 使用 prod 环境 |
| 3 | WAITING_APPROVAL 状态下 execute 返回 APPROVAL_REQUIRED | PASS | `TestServiceExecuteApprovedOrderRejectsWaitingApprovalOrder` |
| 4 | EXECUTING 状态重复 execute 不产生第二个任务 | PASS | `TestServiceExecuteApprovedOrderReturnsExistingTaskWhenAlreadyExecuting` |
| 5 | Plan Revalidate 可触发 PLAN_STALE 且不创建任务 | PASS | `TestServiceExecuteApprovedOrderMarksPlanStaleWithoutCreatingTask` |
| 6 | 审批过期扫描至少以 stub 方式可执行 | PASS | `TestServiceExpireStaleApprovalsTransitionsOrderToExpired` |
| 7 | REQUEST_ACCEPTED 是第一条审计事件 | PASS | submit 路径第一个 audit event 即 `EventRequestAccepted` |
| 8 | PLAN_STALE 路径 EvidencePack 含 stale reason 且 task_id = null | PASS | stale 测试验证 `pack.TaskID == ""` + `pack.FailureDetail["reason"] != ""` |

Phase 02 进入下一阶段的四个条件：

1. **三条链路共享同一批 repository** — PASS（`MemoryStore` 同时被 actionrequest/approval/audit/evidence/execution 共享）
2. **AuthorizationDecision 与 execute auth decision 各自是唯一权威结果** — PASS（两条独立路径，无混用）
3. **PLAN_STALE、EXPIRED、REJECTED 均有真实状态推进与审计记录** — PASS（PLAN_STALE 和 EXPIRED 均有测试覆盖并写入审计；REJECTED 在 approval service 中已接线）
4. **最小控制链路已可演示** — PASS（HTTP 集成测试 + 本地 smoke 已验证）

---

## 阻塞问题

无。

---

## 非阻塞问题（建议性）

### NB-1: `simulate_plan_stale` 占位开关应在 Phase 03 移除

`service.go` 中通过 `request_context.simulate_plan_stale` 注入 stale reason 的逻辑是 Phase 02 演示用占位。Phase 03 实现真实 adapter DryRun 集成后，应移除此开关，改用真实 plan revalidate 结果驱动。

严重级别：**low**，Phase 03 前置清理。

### NB-2: approval actor 仍由 body 传入

`DecisionInput.ApproverID` 当前从请求体传入，未与认证上下文 (`X-Principal-ID` header) 绑定。这意味着审批者身份可以伪造。Phase 03 或之前应将审批者身份收敛到认证中间件。

严重级别：**medium**，安全边界问题，建议 Phase 03 优先处理。

### NB-3: `EvidencePack.execution_success=true` 的 Phase 02 语义

当前 execute 成功路径的 `execution_success=true` 表示"控制链路成功启动 task skeleton"，不是"数据库已创建成功"。`IMPLEMENTATION_STATUS.md` 已正确标注此语义差异。Phase 03 需要将此切回真实终态。

严重级别：**informational**，已正确标注。

### NB-4: 部分 Phase 01 测试被重写而非保留

Phase 01 的 `TestServiceSubmitCreatesApprovedOrderWithoutExecuting` 等测试被重写为 Phase 02 集成测试。原有的 stub-based 单元测试（如 `stubAuthorizationService`、`stubExecutionPlanner`）被移除。这不影响覆盖率（新测试覆盖更完整），但意味着 Phase 02 测试现在依赖更多基础设施（`MemoryStore`、real static resolvers）。若后续需要精确定位回归，可能需要补回独立的 service-level 单元测试。

严重级别：**low**，不阻塞。

### NB-5: `RiskEngine` 对 `Sensitivity` 的判断顺序

`StaticRiskEngine.Evaluate(...)` 先检查 `sensitivity == "high" || "critical"` 再检查 `environment == "prod"`。这意味着 prod + normal sensitivity 和 test + high sensitivity 都会返回 R2。这个行为符合 Phase 02 规范要求（"高敏目标至少触发 require approval"），但需要在 Phase 03 确认这是否是最终业务意图。

严重级别：**informational**。

---

## Phase 02 验收标准对照表

| # | 验收标准 | 结论 |
|---|---|---|
| 1 | dev/test -> APPROVED -> execute -> task skeleton | PASS |
| 2 | prod -> WAITING_APPROVAL -> APPROVED -> explicit execute | PASS |
| 3 | WAITING_APPROVAL + execute -> APPROVAL_REQUIRED | PASS |
| 4 | EXECUTING + repeat execute -> 不产生第二个任务 | PASS |
| 5 | Plan Revalidate -> PLAN_STALE 不创建任务 | PASS |
| 6 | 审批过期扫描可执行 | PASS |
| 7 | REQUEST_ACCEPTED 是第一条审计事件 | PASS |
| 8 | PLAN_STALE evidence 含 stale reason 且 task_id = null | PASS |

---

## 最终决定

```
Gate:    PASS
Action:  允许进入 Phase 03
```

Phase 03 启动前建议关注：

1. **NB-2** 的 approval actor 认证收敛（安全边界）。
2. **NB-1** 的 `simulate_plan_stale` 开关清理。
3. Phase 03 的重点应是：`TaskRuntime` 从 skeleton 升级为真实 southbound 执行 → `DBNativeAdapter` 最小纵切 `mysql.database.create` → execute policy 持久化 → terminal task state → 真实 evidence 固化。
4. 旧 PR #3 建议关闭，以 PR #5 作为正式 Phase 02 合并入口。

---

## 审查元数据

- 审查人：Claude Code (Opus 4.6)
- 审查时间：2026-03-29
- 审查分支：`feat/p2-min-control-flow-v2` (PR #5)
- 基线分支：`main`
- 验证命令：`go test ./...` — 12 个测试包全部通过
