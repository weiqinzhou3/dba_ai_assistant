# Phase 02 - Codex Plan

## 文档作用

由 Codex 在进入 Phase 02 编码前填写，明确最小控制链路的接线计划和验证路径。

## 当前 phase 目标

打通 request、authorization、order、approval、execute、audit/evidence 基础链路，但不做真实 MySQL 写操作。

## 必须回应事项

1. `RiskEngine.Evaluate()` 如何根据 asset environment 动态给出 `R1/R2`：
   - 继续由 `AuthorizationService` 统一编排；`RiskEngine` 本轮只依据已精确命中的 asset 快照动态给出最小风险结论：
   - `dev/test -> R1 + ALLOW`
   - `prod -> R2 + REQUIRE_APPROVAL`
   - 若 asset 带高敏标记，则至少提升到 `REQUIRE_APPROVAL`
   - 不允许由 handler、approval 或 execute 路径自行拼接最终授权结论。
2. `Plan Revalidate` 在本阶段是否会调用 `Adapter.DryRun()`，如果会，调用位置是什么；如果不会，stub 如何保留：
   - 本阶段保留最小 `Plan Revalidate` 占位逻辑，不强依赖 `Adapter.DryRun()`。
   - `Revalidate(...)` 仍发生在显式 `execute` 之后、任务启动之前；先做 plan/version/status 最小检查，并预留 stale reason。
   - `DBNativeAdapter.DryRun()` 继续保留为 southbound stub 能力，但不把真实 DB 探测接回主链路。
3. `PLAN_STALE` 路径的 `EvidencePack` 如何落地，尤其是 `task_id = null` 和 stale reason：
   - execute 路径若 `Revalidate(...)` 返回 invalid/stale，则先把 `plan_status=STALE`、`order.status=PLAN_STALE` 落到共享 repository。
   - 同步追加 `PLAN_STALE` 审计事件，并生成失败型 `EvidencePack`。
   - stale 证据包必须满足：
     - `task_id = null`
     - `execution_success = false`
     - `failure_detail.reason = <stale reason>`
     - `result_summary` 明确为“计划失效，未启动执行任务”

## 本轮计划

- 当前分支：
  - `feat/p2-min-control-flow-v2`
- 本轮目标：
  - 在不接 Deep Agent、不做真实 MySQL 执行的前提下，通过 HTTP API 手工跑通 request -> approval -> execute -> audit/evidence 的最小控制链路。
- 本轮范围：
  - 把 `ActionRequest`、`AssistantOrder`、`ExecutionPlan`、`ApprovalState`、`ExecutionTask`、`AuditEvent`、`EvidencePack` 切到共享 in-memory repository
  - 接线 `ApprovalService` 最小状态推进：`WAITING_APPROVAL -> APPROVED / REJECTED / EXPIRED`
  - 接线 `ExecuteApprovedOrder(...)`：独立 execute auth、plan revalidate、router/runtime/task skeleton、`PLAN_STALE` 阻断
  - 保持 northbound API 为独立的 request / approval / execute / query 接口
  - 更新 coordination / status / runbook / notes / implementation status 文档
- 明确不做：
  - 不实现真实 `mysql.database.create`
  - 不接 Deep Agent
  - 不接 MCP
  - 不新增多个 adapter
  - 不把 execute 塞回 `action-requests`
- 预计修改路径：
  - `cmd/server/main.go`
  - `internal/api/server.go`
  - `internal/application/actionrequest/*`
  - `internal/application/approval/*`
  - `internal/application/audit/*`
  - `internal/application/evidence/*`
  - `internal/application/execution/*`
  - `internal/application/authorization/*`
  - `internal/persistence/*`
  - `internal/domain/asset/types.go`
  - `internal/domain/order/types.go`
  - `internal/domain/plan/types.go`
  - `internal/domain/task/types.go`
  - `internal/adapters/dbnative/adapter.go`
  - `design_docs/24-phase-02-control-flow-notes-v0.1.md`
  - `design_docs/25-phase-02-manual-api-runbook-v0.1.md`
  - `design_docs/coordination/00-dashboard.md`
  - `design_docs/coordination/phase-02/codex-status.md`
  - `design_docs/coordination/phase-02/codex-handoff.md`
  - `IMPLEMENTATION_STATUS.md`
- 计划验证命令：
  - `gofmt -w $(find cmd internal -type f -name '*.go')`
  - `go test ./...`
  - 按 runbook 手工调用：
    - `POST /api/v1/action-requests`
    - `POST /api/v1/orders/{order_id}/approvals`
    - `POST /api/v1/orders/{order_id}/execute`
    - `GET /api/v1/orders/{order_id}`
    - `GET /api/v1/tasks/{task_id}`
    - `GET /api/v1/audit-ledger/{request_id}`
    - `GET /api/v1/evidence-packs/{order_id}`
- 请求 Claude review 的重点：
  - `AuthorizationService` 是否仍是 request 链路唯一最终授权出口
  - approval 与 execute 是否仍物理分离，且 execute 只能显式触发
  - `AssetResolver` 是否仍保持 exact match
  - `PLAN_STALE` 是否正确阻断任务创建，并生成 `task_id = null` 的 evidence
  - `DBNativeAdapter` 是否仍未做真实 `CREATE DATABASE`
