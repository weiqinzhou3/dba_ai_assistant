# Phase 02 - Codex Handoff

## 文档作用

由 Codex 在一个可评审轮次结束后填写，交给 Claude Code 做 review。

## 本轮交接

- 分支：
  - `feat/p2-min-control-flow`
- 提交范围：
  - 最小控制链路从 request 贯通到 approval / execute / audit / evidence 查询
  - 共享 in-memory persistence 正式接线
  - Phase 02 notes / manual runbook / dashboard / implementation status 收口
- 主要变更：
  - `ActionRequestService` 已不再用内部 map，改走共享 `MemoryStore`
  - `ApprovalService` 已接线最小状态机与过期扫描 stub
  - `ExecuteApprovedOrder(...)` 已接线：
    - execute auth
    - plan revalidate
    - `PLAN_STALE` 阻断
    - task skeleton 创建
    - audit / evidence 写入
  - `AuditLedgerView` 已能聚合 `latest_order_status` / `latest_task_id` / `latest_execution_summary`
  - `EvidencePack` 已支持：
    - execute 后 task skeleton 证据
    - `PLAN_STALE` 失败证据
  - 新增 `simulate_plan_stale` 占位开关，仅用于 Phase 02 手工演示 revalidate/stale 路径
- 关键文件：
  - `cmd/server/main.go`
  - `internal/application/actionrequest/service.go`
  - `internal/application/approval/service.go`
  - `internal/application/audit/contracts.go`
  - `internal/application/audit/memory_service.go`
  - `internal/application/evidence/contracts.go`
  - `internal/application/evidence/memory_service.go`
  - `internal/application/execution/stub_planner.go`
  - `internal/persistence/contracts.go`
  - `internal/persistence/memory.go`
  - `design_docs/24-phase-02-control-flow-notes-v0.1.md`
  - `design_docs/25-phase-02-manual-api-runbook-v0.1.md`
- 已执行验证：
  - `gofmt -w $(find cmd internal -type f -name '*.go')`
  - `go test ./...`
  - 本地 HTTP smoke：
    - `POST /api/v1/action-requests` prod -> `WAITING_APPROVAL`
    - `POST /api/v1/orders/{order_id}/approvals` -> `APPROVED`
    - `POST /api/v1/orders/{order_id}/execute` -> `EXECUTING`
    - `GET /api/v1/audit-ledger/{request_id}` 有闭环事件
    - `GET /api/v1/evidence-packs/{order_id}` 可查询
    - stale 模拟路径返回 `PLAN_STALE` 且 `task_id = null`
- 未覆盖项：
  - 未实现真实 `mysql.database.create`
  - execute policy 仍是静态角色判断，不是持久化策略求值
  - approval actor 目前仍由 northbound body 提供，尚未收敛到正式认证上下文
  - task runtime 只起 `RUNNING` 骨架，不进入真实 terminal outcome
- 请求 review 重点：
  - `AuthorizationService` 是否仍是 request 路径唯一最终授权出口
  - approval 与 execute 是否仍物理分离，且 execute 必须显式触发
  - `PLAN_STALE` 是否严格不创建任务，并正确生成 failure evidence
  - `AuditLedgerView` / `EvidencePack` 的最小语义是否足够支撑 Phase 02
  - `DBNativeAdapter` 是否仍完全停留在 skeleton-only
