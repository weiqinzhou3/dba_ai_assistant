# Phase 03 - mysql.database.create MVP

## 目标

在既有控制链路上接入真实 `DBNativeAdapter`，交付一个真正可执行、可审计、可留证据的 MVP 动作：`mysql.database.create`。

## 范围

1. `ExecutionRouter` 将 `mysql.database.create` 路由到 `db_native`。
2. `Plan Revalidate` 接入真实检查：
   - 目标 Asset 仍唯一命中
   - 连接引用仍可解析
   - 数据库仍不存在
   - 幂等键无运行中冲突
   - 计划版本匹配
3. `DBNativeAdapter` 最小执行步骤：
   - `validate_target`
   - `check_database_not_exists`
   - `create_database`
   - `verify_database_created`
4. `DBNativeAdapter.DryRun()` 实现真实预检，至少检查：
   - 连接可达
   - 数据库名符合平台命名规范
   - 目标数据库当前不存在
5. 幂等处理必须明确区分三种情况：
   - 前次已成功 -> 返回受控幂等成功，不重复执行
   - 前次进行中 -> 返回幂等冲突，不创建新任务
   - 前次已失败 -> 允许受控重试
6. `AdapterExecutionResult`、统一错误码、SQL 摘要与 artifact 归档。
7. `EvidencePack` 真实采集：
   - before/after snapshot
   - failure detail
   - rollback suggestion

## 禁止事项

1. 不扩展到 `mysql.user.create`、`mysql.user.grant`、`mysql.password.change`。
2. 不同时做多数据库引擎适配。
3. 不让 Agent 或 handler 直接拼 SQL 执行。
4. 不自动重试 `CREATE DATABASE` 这类写操作。
5. 不把真实凭证暴露给 Deep Agent。

## 产物

1. 可工作的 `DBNativeAdapter`。
2. `mysql.database.create` 真实纵切链路。
3. 幂等键检查与冲突处理。
4. 成功、失败、`PLAN_STALE` 三条路径的真实证据。
5. 最小集成测试与运行说明。

## 验收标准

1. 受控 MySQL 目标可以真实创建数据库。
2. 已存在数据库时返回受控幂等结论或冲突结论，而不是重复副作用。
3. prod 路径仍必须先审批，审批通过后仍需显式 execute。
4. execute 前数据库状态变化能触发 `PLAN_STALE`。
5. 成功、失败、`PLAN_STALE` 三条路径均有审计与证据。
6. 返回给上层的是统一错误码，而不是裸驱动异常。
7. `DBNativeAdapter.DryRun()` 已实现并纳入验证。
8. 幂等成功、幂等冲突、失败后受控重试三种情况均有明确行为。

## 风险点

1. 本地和 CI 中稳定准备 MySQL 测试环境有额外成本。
2. `CREATE DATABASE` 的幂等语义容易和“数据库已存在”混淆。
3. 连接引用、密钥管理和执行凭证权限若不严，会绕过 Control Layer 设计初衷。

## 进入下一阶段条件

1. `mysql.database.create` 已经是可演示动作，不再是 stub。
2. dev/test 与 prod 路径都已验证。
3. 审计和证据在真实执行路径中得到验证。
4. `PLAN_STALE` 路径已有实际测试或演示证据。

## 推荐 branch 名

`phase/03-mysql-database-create-mvp`

## 推荐 commit message 模式

1. `feat(mysql): implement mysql.database.create action flow`
2. `feat(adapter): add dbnative create-database execution`
3. `feat(evidence): capture before after snapshots for mysql create`
4. `test(mysql): add integration coverage for create database flow`
