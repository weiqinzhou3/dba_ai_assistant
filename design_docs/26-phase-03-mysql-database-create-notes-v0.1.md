# Phase 03 - mysql.database.create Notes v0.1

## 作用

记录 Phase 03 `mysql.database.create` MVP 的实现约束、运行方式和当前取舍，供 review / handoff / 后续 phase 继续演进时参考。

## 本轮实现收口

1. `mysql.database.create` 现在已经从 `ActionRequest -> AuthorizationDecision -> AssistantOrder -> explicit execute -> DBNativeAdapter` 走到真实 southbound。
2. `AuthorizationService` 仍只用于 request 路径的最终授权聚合；execute 仍单独走 `ExecuteAuthorizationService`。
3. approval / execute 仍物理分离；审批通过不会自动执行。
4. `AssetResolver` 仍然只有 `ResolveExact(...)`，没有引入 fuzzy match。
5. `DBNativeAdapter` 仍只支持 `mysql.database.create`，没有扩散到其他动作或其他 adapter。

## 参数与命名规范

- 当前只接受 `action_hint = mysql.database.create`
- `database_name` 必须满足平台命名规范：

```text
^[A-Za-z][A-Za-z0-9_]{0,63}$
```

- `resource_selector.project/environment/service_instance` 不能为空

## 真实 revalidate

execute 前现在会做以下真实检查：

1. 冻结 plan 版本仍与 order 匹配
2. 目标资产仍能被 `ResolveExact(...)` 唯一命中
3. 命中的资产仍与冻结的 `resolved_asset_ids` 一致
4. `ExecutionRouter` 仍能绑定 `db_native`
5. `DBNativeAdapter.DryRun()` 能完成：
   - 连接引用解析
   - 目标引擎校验
   - MySQL 可达性检查
   - 数据库当前存在性检查
6. 幂等键不存在运行中冲突

## 最小幂等语义

幂等键仍采用：

```text
mysql.database.create:<target_asset_id>:<database_name>
```

当前行为：

1. 若同 key 已有 `RUNNING` 记录，本次 execute 返回 `IDEMPOTENCY_CONFLICT`，不新建任务。
2. 若同 key 已有 `SUCCEEDED` 记录，且当前数据库确实已存在，本次 execute 返回受控幂等成功，不重复执行 `CREATE DATABASE`。
3. 若同 key 已有 `FAILED` 记录，则允许继续执行一次新的受控重试。
4. 若数据库在 execute 前已经存在、但没有对应成功幂等记录，则视为目标状态漂移，order 进入 `PLAN_STALE`。

## DBNativeAdapter 运行配置

当前 `DBNativeAdapter` 通过受控 env 配置解析 `connection_ref`：

1. 推荐：

```text
DBNATIVE_CONNECTIONS_JSON
```

示例：

```json
{
  "secret://db-targets/mysql-order-main-test": "root:root@tcp(127.0.0.1:3306)/mysql?parseTime=true",
  "secret://db-targets/mysql-order-main-prod": "root:root@tcp(127.0.0.1:3307)/mysql?parseTime=true"
}
```

2. 备选：

```text
DBNATIVE_DSN_<SANITIZED_CONNECTION_REF>
```

其中 `sanitize` 规则是把非字母数字字符转成 `_` 并大写。

## 审计 / 证据语义变化

Phase 02：

- `execution_success=true` 只表示 task skeleton 已启动

Phase 03：

- `execution_success=true` 只在真实动作终态成功时写入
- 成功路径会追加：
  - `EXECUTION_STARTED`
  - `EXECUTION_SUCCEEDED`
  - `EVIDENCE_WRITTEN`
- 失败路径会追加：
  - `EXECUTION_STARTED`（若已真正进入 southbound）
  - `EXECUTION_FAILED`
  - `EVIDENCE_WRITTEN`
- `PLAN_STALE` 仍不创建 task，`task_id = null`

## 当前仍未覆盖

1. 审批 actor 还没有完全强制只信任认证中间件；当前实现是“认证上下文优先，若缺失则保留 body 兼容”。
2. 幂等记录仍是 in-memory，不具备跨进程 / 重启恢复能力。
3. 真实 MySQL 集成验证仍依赖手工环境，不在默认单测里自动拉起数据库。
4. adapter artifact 目前只产出引用，不做持久化 artifact store。
