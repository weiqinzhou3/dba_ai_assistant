# RBAC / ABAC Model v0.1

## 0. 文档定位

本文档定义企业级 DBA Assistant 的权限模型。

本文档目标不是复刻历史平台的 URL/Method 权限，而是定义适合 Assistant 的：

- 主体身份模型
- 角色模型
- 动作权限模型
- 资源范围模型
- 风险策略模型
- 审批策略模型

---

## 1. 为什么需要独立 RBAC/ABAC 模型

如果没有独立权限模型，系统只能退化成：

- Agent 能调到哪个 tool，就相当于有权限
- 谁知道 order_id，谁就可能尝试 execute
- 底层 adapter 成为事实上的权限边界

这对企业级 DBA Assistant 是不可接受的。

本项目必须支持判断：

1. 谁能发起什么动作
2. 谁能对哪些资源发起动作
3. 哪些动作在哪些环境下可以直接执行
4. 哪些动作必须审批
5. 谁可以审批
6. 谁可以 execute 已批准工单

---

## 2. 权限判断总公式

```text
最终是否可执行 =
身份合法
× 动作允许
× 资源范围命中
× 风险策略放行
× 审批满足
× execute 触发人也合法
```

说明：

- “发起动作” 与 “执行动作” 是两层权限。
- “审批权限” 又是第三层权限。
- 因此系统至少要有三类能力：
  - Request 权限
  - Approval 权限
  - Execute 权限

---

## 3. 模型分层

## 3.1 身份层（Identity Layer）
回答“你是谁”。

核心对象：
- `Principal`
- `Role`
- `Group`

## 3.2 动作层（Action Authorization Layer）
回答“你能发起什么动作”。

核心对象：
- `ActionPolicy`

## 3.3 资源层（Resource Scope Layer）
回答“你能对哪些对象做这些动作”。

核心对象：
- `ResourceScopePolicy`

## 3.4 风险层（Risk Governance Layer）
回答“在当前环境/敏感度下，你能做到什么程度”。

核心对象：
- `RiskPolicy`

## 3.5 审批层（Approval Governance Layer）
回答“是否必须审批、谁来审批、审批是否过期”。

核心对象：
- `ApprovalPolicy`

---

## 4. 核心对象定义

## 4.1 Principal

### 作用
表示本次请求主体。

### 字段建议

| 字段 | 类型 | 说明 |
|---|---|---|
| `principal_id` | string | 主体 ID |
| `principal_type` | enum | `human` / `service` / `agent_proxy` |
| `user_id` | string | 用户唯一标识 |
| `display_name` | string | 展示名 |
| `is_active` | bool | 是否启用 |

### 说明
Principal 是权限判断的起点。  
Deep Agent 不能跳过 Principal 直接发执行请求。

---

## 4.2 Role

### 作用
表示一类职责边界。

### 首批建议角色

| 角色 | 说明 |
|---|---|
| `assistant_user` | 基础使用者 |
| `dba` | DBA 通用角色 |
| `mysql_operator` | MySQL 相关动作发起/执行角色 |
| `backup_operator` | 备份动作相关角色 |
| `prod_approver` | 生产审批角色 |
| `platform_admin` | 平台管理员 |
| `readonly_auditor` | 只读审计角色 |
| `control_executor` | 受控后台执行主体 |

### 说明
角色不是最终裁决器。  
角色只是输入，真正的允许/拒绝由策略计算得出。

---

## 4.3 Group

### 作用
表示组织归属，例如 DBA 团队、平台团队、项目团队。

### 用途
- 辅助 scope 管理
- 辅助审批人范围
- 辅助默认策略挂载

---

## 4.4 ActionPolicy

### 作用
定义谁可以发起哪个 Action。

### 推荐结构

| 字段 | 类型 | 说明 |
|---|---|---|
| `policy_id` | string | 策略 ID |
| `subject_type` | enum | `role` / `group` / `principal` |
| `subject_ref` | string | 角色名/组名/主体 ID |
| `action_name` | string | 标准动作名 |
| `effect` | enum | `allow` / `deny` |
| `conditions` | json | 可选条件 |
| `enabled` | bool | 是否启用 |

### 示例
- `mysql_operator` allow `mysql.database.create`
- `readonly_auditor` deny `mysql.database.create`

### 说明
ActionPolicy 只解决“是否允许发起该动作”，不解决是否能在 prod 上做。

---

## 4.5 ResourceScopePolicy

### 作用
定义动作可作用于哪些资源范围。

### 推荐结构

| 字段 | 类型 | 说明 |
|---|---|---|
| `scope_policy_id` | string | 策略 ID |
| `subject_type` | enum | `role` / `group` / `principal` |
| `subject_ref` | string | 主体引用 |
| `action_name` | string | 可选，允许按动作细化 |
| `project` | string[] | 允许的项目 |
| `environment` | string[] | 允许的环境 |
| `cluster` | string[] | 允许的集群 |
| `service_instance` | string[] | 允许的服务实例 |
| `effect` | enum | `allow` / `deny` |

### 示例
- `mysql_operator` 对 `project=order-platform`、`environment=dev/test` allow
- `mysql_operator` 对 `environment=prod` 可以 allow 发起，但不代表可直接 execute

---

## 4.6 RiskPolicy

### 作用
定义不同风险等级下的控制策略。

### 推荐结构

| 字段 | 类型 | 说明 |
|---|---|---|
| `risk_policy_id` | string | 策略 ID |
| `subject_type` | enum | `role` / `group` / `principal` |
| `subject_ref` | string | 主体引用 |
| `action_name` | string | 动作名 |
| `risk_level` | enum | `R0/R1/R2/R3` |
| `decision` | enum | `allow` / `require_approval` / `deny` |
| `conditions` | json | 环境/敏感度条件 |

### 示例
- `mysql_operator` + `mysql.database.create` + `R1` -> `allow`
- `mysql_operator` + `mysql.database.create` + `R2` -> `require_approval`
- `assistant_user` + `mysql.database.create` + `R2` -> `deny`

---

## 4.7 ApprovalPolicy

### 作用
定义审批规则。

### 推荐结构

| 字段 | 类型 | 说明 |
|---|---|---|
| `approval_policy_id` | string | 策略 ID |
| `action_name` | string | 动作名 |
| `environment` | string[] | 环境范围 |
| `risk_level` | string[] | 风险范围 |
| `approver_roles` | string[] | 可审批角色 |
| `min_approver_count` | int | 最少审批人数 |
| `forbid_self_approval` | bool | 是否禁止自我审批 |
| `approval_ttl_seconds` | int | 审批有效期 |
| `enabled` | bool | 是否启用 |

### MVP 建议
- 先做单人审批
- 默认禁止自我审批
- prod 审批 TTL 明确配置，不走隐式常量

---

## 4.8 ExecutePolicy（建议显式建模）

### 作用
定义谁可以触发已批准工单的 execute。

### 为什么建议单独建模
“可发起请求” 不等于 “可触发 execute”。

### 推荐结构

| 字段 | 类型 | 说明 |
|---|---|---|
| `execute_policy_id` | string | 策略 ID |
| `subject_type` | enum | `role` / `group` / `principal` |
| `subject_ref` | string | 主体引用 |
| `action_name` | string | 动作名 |
| `environment` | string[] | 环境范围 |
| `effect` | enum | `allow` / `deny` |

### 示例
- `mysql_operator` 可以 execute dev/test 上已批准工单
- `control_executor` 可以 execute 后台受控工单
- `assistant_user` 不可以直接 execute

---

## 5. 最小决策链路

## 5.1 提交请求时

```text
Principal
  -> ActionPolicy
  -> ResourceScopePolicy
  -> RiskPolicy
  -> ApprovalPolicy
  -> AuthorizationDecision
```

结论可能是：

- `DENY`
- `ALLOW_NO_APPROVAL`
- `ALLOW_WITH_APPROVAL`

## 5.2 审批时

校验：

- 审批人是否具备 `approver_roles`
- 是否自我审批
- 审批是否过期

## 5.3 execute 时

校验：

- execute 调用主体是否来自认证上下文
- 是否具备 ExecutePolicy
- 工单是否为 `APPROVED`
- Plan 是否仍然有效

---

## 6. MVP 推荐规则（可直接实现）

## 6.1 角色
- `assistant_user`
- `mysql_operator`
- `prod_approver`
- `platform_admin`
- `control_executor`

## 6.2 动作
先只覆盖：
- `mysql.database.create`

## 6.3 默认动作权限
- `mysql_operator` allow `mysql.database.create`
- `assistant_user` deny `mysql.database.create`
- `platform_admin` allow `*`

## 6.4 默认资源范围
- `mysql_operator` 仅限指定 project / env / service_instance
- `platform_admin` 可全局
- `prod_approver` 不等于 execute 权限

## 6.5 默认风险策略
- dev/test + `R1` -> allow
- prod + `R2` -> require approval
- 高敏实例 -> 至少 require approval

## 6.6 默认审批策略
- prod 必须审批
- 自我审批禁止
- 审批通过不自动 execute

## 6.7 默认 execute 策略
- dev/test 可由 `mysql_operator` execute
- prod 建议由 `mysql_operator` 或 `control_executor` execute
- `assistant_user` 不得 execute

---

## 7. 建表建议

建议至少拆这些表：

- `principals`
- `roles`
- `groups`
- `principal_roles`
- `principal_groups`
- `action_policies`
- `resource_scope_policies`
- `risk_policies`
- `approval_policies`
- `execute_policies`

---

## 8. 典型判断示例

## 8.1 dev/test 创建数据库
主体：`mysql_operator`  
动作：`mysql.database.create`  
资源：test 环境  
风险：`R1`

结果：
- Action allow
- Scope allow
- Risk allow
- Approval not required
- 可进入 `APPROVED`
- 但仍需显式 execute

## 8.2 prod 创建数据库
主体：`mysql_operator`  
动作：`mysql.database.create`  
资源：prod 环境  
风险：`R2`

结果：
- Action allow
- Scope allow
- Risk require approval
- 创建 `WAITING_APPROVAL`
- 审批通过后进入 `APPROVED`
- 仍需合法 execute 主体触发执行

## 8.3 非 DBA 用户创建数据库
主体：`assistant_user`  
动作：`mysql.database.create`

结果：
- Action deny
- 直接 `POLICY_REJECTED`

## 8.4 审批人就是发起人
结果：
- `SELF_APPROVAL_FORBIDDEN`

---

## 9. 对 Coding Agent 的直接要求

1. 不要把 URL/Method 权限模型直接搬进来。
2. 必须围绕 Action/Resource/Risk/Approval/Execute 这五层建模。
3. Role 只是输入，不是最终决策。
4. `AuthorizationDecision` 必须是唯一权威输出。
5. execute 权限不要隐含在 request 权限里。
6. MVP 先做最小可用模型，不要一开始做超级复杂的多级审批。
