# Adapter Interface v0.1

## 0. 文档定位

本文档定义 Assistant Control Layer 对下游执行器暴露的统一 Southbound Adapter SPI。

本文档只回答：

1. Adapter 在系统里到底是什么
2. Adapter 的统一输入输出长什么样
3. Adapter 和 Skill / Action 的关系是什么
4. MVP 阶段为什么先只实现 `DB-native Adapter`

---

## 1. 概念边界

## 1.1 Skill 不是 Adapter
- Skill 是给 Deep Agent 看的高层业务能力入口。
- Adapter 是给 Control Layer 用的底层执行插件。

关系是：

```text
Deep Agent
  -> Skill
     -> Control Layer
        -> Action
           -> Adapter
```

## 1.2 Action 不是 Adapter
Action 是系统内部标准动作名，如：

- `mysql.database.create`
- `mysql.user.create`

Adapter 是 Action 的执行实现之一，如：

- `DBNativeAdapter`
- `MCPToolAdapter`
- `CRDAdapter`

## 1.3 Adapter 不负责什么
Adapter 不负责：

- 自然语言理解
- 最终权限裁定
- 风险判断
- 审批判断
- 工单状态推进

这些都属于 Control Layer。

---

## 2. Adapter 设计原则

1. 对 Control Layer 呈现统一接口。
2. 隔离底层执行差异。
3. 返回统一结果模型。
4. 支持幂等 key、超时、日志引用、artifact 引用。
5. 支持前置探测或 dry-run。
6. 不允许让 Deep Agent 直接拿到 Adapter。

---

## 3. Adapter 类型总览

首版预留这些类型：

- `mcp_tool`
- `db_native`
- `crd`
- `grpc`
- `k8s`
- `shell_ansible`
- `vm_ssh`

MVP 只实现：

- `db_native`

---

## 4. 统一 Adapter SPI

建议接口：

```go
type Adapter interface {
    Type() AdapterType
    Supports(ctx context.Context, req AdapterCapabilityRequest) (bool, error)
    DryRun(ctx context.Context, req AdapterExecutionRequest) (AdapterDryRunResult, error)
    Execute(ctx context.Context, req AdapterExecutionRequest) (AdapterExecutionResult, error)
}
```

---

## 5. 通用请求对象

## 5.1 AdapterCapabilityRequest

### 作用
用于路由前判断某个 Adapter 是否支持当前 Action + Target 组合。

### 字段建议

| 字段 | 类型 | 说明 |
|---|---|---|
| `action_name` | string | 标准动作名 |
| `target_asset_type` | string | 目标资产类型 |
| `target_adapter_hints` | string[] | 资产建议路线 |
| `environment` | string | 环境 |
| `parameters` | json | 参数摘要 |

---

## 5.2 AdapterExecutionRequest

### 作用
Control Layer 给 Adapter 的正式执行请求。

### 字段建议

| 字段 | 类型 | 说明 |
|---|---|---|
| `trace_id` | string | trace ID |
| `order_id` | string | 工单 ID |
| `task_id` | string | 任务 ID |
| `step_id` | string | 步骤 ID |
| `action_name` | string | 动作名 |
| `target` | json | 目标对象与连接引用 |
| `parameters` | json | 动作参数 |
| `execution_controls` | json | 超时、重试、幂等控制 |
| `evidence_requirements` | json | 证据采集要求 |

### 示例

```json
{
  "trace_id": "trace_01H...",
  "order_id": "ord_01H...",
  "task_id": "task_01H...",
  "step_id": "step_01H...",
  "action_name": "mysql.database.create",
  "target": {
    "asset_id": "dbt_1001",
    "asset_type": "DatabaseTarget",
    "connection_ref": "secret://db-targets/mysql-order-main"
  },
  "parameters": {
    "database_name": "order_center"
  },
  "execution_controls": {
    "timeout_seconds": 30,
    "retry_policy": "none",
    "idempotency_key": "mysql.database.create:dbt_1001:order_center"
  },
  "evidence_requirements": {
    "capture_before_state": true,
    "capture_after_state": true,
    "capture_sql_summary": true
  }
}
```

---

## 6. 通用返回对象

## 6.1 AdapterDryRunResult

### 作用
返回执行前探测结果。

### 建议字段

| 字段 | 类型 | 说明 |
|---|---|---|
| `supported` | bool | 是否支持 |
| `ready` | bool | 是否准备就绪 |
| `issues` | string[] | 问题清单 |
| `rendered_preview` | json | 预览内容 |

### 说明
MVP 可以先只做简单实现：
- DB-native 只检查连接可用、数据库名规范、数据库不存在
- 其他 adapter 先不实现 dry-run 细节

---

## 6.2 AdapterExecutionResult

### 作用
统一返回执行结果。

### 建议字段

| 字段 | 类型 | 说明 |
|---|---|---|
| `success` | bool | 是否成功 |
| `status` | enum | `SUCCEEDED` / `FAILED` / `TIMEOUT` |
| `provider_task_id` | string | 下游任务 ID |
| `provider_step_ref` | string | 下游步骤引用 |
| `summary` | string | 执行摘要 |
| `outputs` | json | 输出字段 |
| `artifacts` | json[] | artifact 引用 |
| `started_at` | datetime | 开始时间 |
| `ended_at` | datetime | 结束时间 |
| `error` | json | 错误详情 |

### 失败也必须返回
失败时也要尽量返回：
- `summary`
- `artifacts`
- `started_at`
- `ended_at`
- 统一 `error.code` / `error.message`

---

## 7. DB-native Adapter 设计

## 7.1 适用动作
MVP 先支持：
- `mysql.database.create`

后续可扩展：
- `mysql.user.create`
- `mysql.user.grant`
- `mysql.password.change`

## 7.2 输入要求
- 受控连接引用，不是裸密码
- 数据库名
- 可选 charset/collation

## 7.3 最小执行流程
1. 建立受控连接
2. 校验数据库不存在
3. 生成 SQL 摘要
4. 执行 `CREATE DATABASE`
5. 再次验证数据库已存在
6. 返回结果与 artifact

## 7.4 Artifact 最小要求
建议输出：
- SQL 摘要
- 执行前状态快照
- 执行后状态快照
- 错误摘要（若失败）

## 7.5 幂等要求
幂等键建议：

```text
mysql.database.create:<target_asset_id>:<database_name>
```

规则：
- 已成功则可返回幂等成功
- 运行中则返回幂等冲突
- 不默认自动重试写操作

---

## 8. MCP Tool Adapter 设计（预留）

## 8.1 用途
在企业内部已有 MCP Server 且 tool 定义清晰时接入。

## 8.2 注意点
MCP Tool Adapter 也必须经过：
- AuthorizationDecision
- Approval
- Execute
- Task/Audit/Evidence

MCP 只是执行通道，不是控制面替代品。

## 8.3 MVP 不实现原因
- 现阶段先打通 Control Layer 主链路
- 先避免多种 adapter 并行带来的复杂度
- 后续再把 MCP 作为第二条执行路线接入

---

## 9. CRD / gRPC / K8s / Shell / VM Adapter（预留）

这些 adapter 的接口都服从同一个 SPI。  
区别只在 `target` 和 `artifacts` 的具体内容不同。

### 9.1 CRD Adapter
用于 apply/query/update 自定义资源。

### 9.2 gRPC Adapter
用于内部标准化执行接口。

### 9.3 K8s Adapter
用于原生资源读写。

### 9.4 Shell/Ansible Adapter
用于传统运维脚本执行。

### 9.5 VM/SSH Adapter
用于非 K8s 的虚拟机场景。

---

## 10. 路由建议

Control Layer 不应把执行器写死在 Action 上。  
建议由 `ExecutionRouter` 根据以下信息路由：

- `action_name`
- 目标资产类型
- 资产的 adapter hints
- 环境
- 风险等级
- 平台配置优先级

MVP 中可先做简单路由：
- `mysql.database.create` -> `db_native`

---

## 11. 错误模型要求

Adapter 层错误必须被归一化，不允许把原始驱动异常直接抛给 Agent。

建议至少统一成：

- `ADAPTER_NOT_AVAILABLE`
- `TARGET_CONNECTION_FAILED`
- `PRECHECK_FAILED`
- `EXECUTION_FAILED`
- `IDEMPOTENCY_CONFLICT`
- `TIMEOUT`

---

## 12. 对 Coding Agent 的直接要求

1. 先定义 SPI，再写 `DBNativeAdapter`。
2. 不要让 Adapter 直接读自然语言参数。
3. 不要把权限判断塞进 Adapter。
4. 不要让 Agent 直接调用 Adapter。
5. 失败路径也要返回统一结果对象。
6. MVP 只做一个能工作的 `DBNativeAdapter`，其余类型先建接口与占位。
