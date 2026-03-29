# Phase 03 - Codex Handoff

## 文档作用

由 Codex 在一个可评审轮次结束后填写，交给 Claude Code 做 review。

## 本轮交接

- 分支：`feat/p3-mysql-database-create`
- 提交范围：
  - `mysql.database.create` Phase 03 MVP 真实纵切
  - 真实 `DBNativeAdapter`
  - execute 前真实 revalidate / 最小幂等 / terminal audit-evidence 收口
  - approval actor 认证边界最小收敛
- 主要变更：
  - `ActionRequestService.Submit(...)` 现在会校验：
    - 仅支持 `mysql.database.create`
    - `database_name` 平台命名规范
    - `resource_selector` 必填
  - `ExecuteApprovedOrder(...)` 现在会：
    - 去掉 `simulate_plan_stale`
    - 重新 exact resolve asset
    - 构造真实 `AdapterExecutionRequest`
    - 调用 `DBNativeAdapter.DryRun()`
    - 处理 `PLAN_STALE` / `IDEMPOTENCY_CONFLICT` / idempotent replay / real execute
    - 将 order/task 推进到 `SUCCEEDED` / `FAILED`
  - `DBNativeAdapter` 现在会：
    - 解析 `connection_ref`
    - 校验引擎和命名规范
    - 检查数据库存在性
    - 执行 `CREATE DATABASE`
    - 再次验证数据库已创建
  - `EvidencePack.execution_success` 已切换为真实动作终态语义。
  - approval API 现在在认证上下文存在时优先信任 `X-Principal-ID`，并对 body/header 不一致做阻断。
- 关键文件：
  - [service.go](/Users/zqw/Desktop/Project/dba_ai_assistant/internal/application/actionrequest/service.go)
  - [adapter.go](/Users/zqw/Desktop/Project/dba_ai_assistant/internal/adapters/dbnative/adapter.go)
  - [stub_planner.go](/Users/zqw/Desktop/Project/dba_ai_assistant/internal/application/execution/stub_planner.go)
  - [memory.go](/Users/zqw/Desktop/Project/dba_ai_assistant/internal/persistence/memory.go)
  - [server.go](/Users/zqw/Desktop/Project/dba_ai_assistant/internal/api/server.go)
  - [26-phase-03-mysql-database-create-notes-v0.1.md](/Users/zqw/Desktop/Project/dba_ai_assistant/design_docs/26-phase-03-mysql-database-create-notes-v0.1.md)
  - [27-phase-03-manual-verification-runbook-v0.1.md](/Users/zqw/Desktop/Project/dba_ai_assistant/design_docs/27-phase-03-manual-verification-runbook-v0.1.md)
- 已执行验证：
  - `go test ./internal/application/actionrequest -run 'PhaseThree|RealTerminal|Idempotency|InvalidDatabaseName'`
  - `go test ./internal/application/actionrequest ./internal/application/execution ./internal/adapters/dbnative ./internal/api`
  - `go test ./...`
- 未覆盖项：
  - 默认自动化测试未直接连真实 MySQL；真实 southbound 通过可注入 admin 接口单测覆盖，另有手工 runbook。
  - approval actor 仍未完全禁止 body fallback；当前是“认证上下文优先 + mismatch 阻断 + 无认证时兼容旧 Phase 02 调用”。
  - 幂等记录仍为 in-memory，不具备进程重启后的恢复能力。
- 请求 review 重点：
  - `AuthorizationService` 是否仍是 request 路径唯一最终授权出口。
  - execute 是否仍必须显式触发，且 execute auth 仍独立。
  - approval actor / execute actor 是否都已向认证上下文收敛而未破坏现有主链路。
  - `AssetResolver` 是否仍只做 exact match。
  - `simulate_plan_stale` 是否已从正式产品路径移除。
  - `EvidencePack.execution_success` 是否已与真实动作终态一致。
  - `DBNativeAdapter` 是否仍严格只覆盖 `mysql.database.create`。
