# Phase 03 - Codex Plan

## 文档作用

由 Codex 在进入 Phase 03 编码前填写，明确 `mysql.database.create` MVP 纵切的实施顺序与验证方案。

## 当前 phase 目标

接入真实 `DBNativeAdapter`，交付 `mysql.database.create` 可执行 MVP。

## 必须回应事项

1. `DBNativeAdapter.DryRun()` 的真实预检范围是什么：
   - 连接引用必须能解析到受控 MySQL DSN。
   - 目标引擎必须显式收敛为 `mysql`。
   - `database_name` 必须满足平台命名规范 `^[A-Za-z][A-Za-z0-9_]{0,63}$`。
   - 连接必须可达，且 `information_schema.schemata` 查询成功。
   - 目标数据库当前是否存在必须返回给上层，用于区分 `PLAN_STALE`、幂等成功和继续执行。
2. 幂等三种情况如何处理：
   - 前次已成功
     - 若同一 `idempotency_key` 已有成功记录，且当前目标数据库仍存在，则本次 execute 返回受控幂等成功，不重复执行 `CREATE DATABASE`。
   - 前次进行中
     - 若同一 `idempotency_key` 已有运行中记录，则阻断本次 execute，返回 `IDEMPOTENCY_CONFLICT`，且不新建任务。
   - 前次已失败
     - 若同一 `idempotency_key` 的最近记录为失败，则允许受控重试，并生成新的任务终态与审计/证据。
3. 哪些验证将证明 `PLAN_STALE`、幂等冲突和成功路径都已覆盖：
   - 单元测试覆盖参数校验、真实 revalidate 结果分类、幂等记录三种状态分支、终态 audit/evidence 语义。
   - adapter 级测试覆盖 dry-run 预检、`CREATE DATABASE` 成功、已有数据库、连接解析失败和执行后校验失败。
   - 可选手工 runbook 使用真实 MySQL 验证 dev/test 与 prod 审批后 execute 的成功 / 失败 / `PLAN_STALE` 路径。

## 本轮计划

- 当前分支：`feat/p3-mysql-database-create`
- 本轮目标：
  - 在不改变 request / approval / execute 主链路门禁的前提下，让 `mysql.database.create` 首次完成真实 southbound 执行，并把成功/失败证据收敛到真实动作终态。
- 本轮范围：
  - `mysql.database.create` 提交参数校验。
  - execute 前真实 revalidate：plan version、exact asset、connection ref、目标不存在、最小幂等冲突。
  - `DBNativeAdapter` 最小 MySQL 实现：`validate_target` / `check_database_not_exists` / `create_database` / `verify_database_created`。
  - `TaskRuntime` 从 skeleton 升级为同步真实执行，并把 task/order 推进到 terminal state。
  - 成功 / 失败 / `PLAN_STALE` / 幂等冲突的真实 audit / evidence 收口。
  - approval actor 读取认证上下文优先，保持 approval / execute 物理分离。
- 明确不做：
  - 不接 Deep Agent。
  - 不接 MCP。
  - 不增加第二个 adapter。
  - 不扩展 `mysql.user.create` / `grant` / `backup`。
  - 不把 request 授权与 execute 授权合并。
- 预计修改路径：
  - `internal/application/actionrequest/`
  - `internal/application/execution/`
  - `internal/adapters/dbnative/`
  - `internal/application/evidence/`
  - `internal/application/audit/`
  - `internal/api/`
  - `internal/persistence/`
  - `cmd/server/`
  - `design_docs/coordination/phase-03/`
  - `design_docs/coordination/00-dashboard.md`
  - `IMPLEMENTATION_STATUS.md`
- 计划验证命令：
  - `go test ./internal/application/actionrequest ./internal/application/execution ./internal/adapters/dbnative ./internal/api`
  - `go test ./...`
- 请求 Claude review 的重点：
  - `AuthorizationService` 是否仍是 request 路径唯一最终授权出口。
  - execute 是否仍必须显式触发，且 execute auth 仍独立。
  - approval actor / execute actor 是否都优先收敛到认证上下文。
  - `AssetResolver` 是否仍严格 exact match。
  - `simulate_plan_stale` 是否已从正式链路移除。
  - `EvidencePack.execution_success` 是否已切换为真实动作终态语义。
  - `DBNativeAdapter` 是否仍只覆盖 `mysql.database.create`。
