# 企业级 DBA Assistant 规格说明书 v0.7

## 0. 文档定位

本文档是本项目的**正式规格说明书**。

本文档面向：

- 产品设计
- 架构设计
- Coding Agent（如 Codex / Claude Code）
- 后续开发与评审

本文档必须满足以下要求：

1. 自解释。
2. 自包含。
3. 不依赖读者认识任何历史平台产品。
4. 允许在无额外背景知识的前提下继续做设计和编码。

### 0.1 v0.7 修订摘要

相较于 v0.6，本版补上了上一版仍可能让实现分叉的三个点：

1. 修正修订摘要与版本基线：本版摘要明确只描述 v0.6 → v0.7 的真实增量，而不再混入更早版本的变更。
2. 为 `ApprovalPolicy` 增加 `approval_ttl`（审批有效期）语义：审批过期不再依赖隐含全局常量，推荐按动作/环境粒度配置；若未配置，可回退到平台全局默认 TTL。
3. 保持显式 execute 语义不变的前提下，继续明确配套运行机制：`REQUEST_ACCEPTED` 的边界、审批超时调度、Deep Agent 自动串联 `request -> execute` 的行为指引仍然有效。

---

## 1. 项目目标

建设一个面向 DBA / 平台工程师的企业级 Assistant。

该 Assistant 需要具备以下能力：

1. 理解自然语言请求。
2. 识别出标准运维动作。
3. 将动作绑定到受控资产对象。
4. 判断当前请求主体是否有权限。
5. 判断动作风险等级。
6. 在必要时发起审批。
7. 通过受控执行器完成真实执行。
8. 全程记录审计与证据。

本项目不是一个单纯的“聊天机器人”，而是一个面向企业运维场景的**受控执行系统**。

---

## 2. 系统边界

### 2.1 包含内容

本项目包含：

- 基于 Deep Agent 的上层智能体接入
- Assistant Control Layer
- 动作字典（Action Dictionary）
- 资产解析能力（Asset Resolver）
- 权限、风险、审批统一控制
- 审计账本与证据包
- 可插拔执行适配器

### 2.2 不包含内容

v0.7 不包含：

- 自动无人审批执行高风险动作
- 完整企业 UI
- 替代所有既有平台
- 一次性覆盖所有数据库和中间件
- 自然语言资产搜索引擎本身

---

## 3. 总体架构

```text
User
  -> Deep Agent
     -> High-level Skill
        -> Assistant Control Layer
           -> Action Normalizer
           -> Principal Resolver
           -> Asset Resolver
           -> Authorization Service
              -> Policy Engine
              -> Risk Engine
           -> Approval Runtime
           -> Order Service
           -> Plan Builder
           -> Execution Router
           -> Task Runtime
           -> Audit Ledger
           -> Evidence Pack
           -> Adapter Layer
              -> MCP Tool Adapter
              -> DB-native Adapter
              -> CRD Adapter
              -> GrpcCall Adapter
              -> K8s Adapter
              -> Shell/Ansible Adapter
              -> VM/SSH Adapter
```

### 3.1 各层职责

#### User
发起自然语言请求。

#### Deep Agent
负责：

- 理解意图
- 参数补全
- 多步计划
- 调用高层 Skill
- 结果解释

#### High-level Skill
暴露给 Deep Agent 的稳定业务能力入口。

#### Assistant Control Layer
负责：

- 动作标准化
- 主体识别
- 资产解析
- 基础权限判断
- 风险判断
- 审批控制
- 工单与计划管理
- 执行路由
- 审计记账
- 证据归档

#### Adapter Layer
负责：

- 真正调用底层目标系统
- 返回统一执行结果

### 3.2 控制流主链路

v0.7 规定的权威主链路如下：

```text
Action Request
  -> Action Normalizer
  -> Principal Resolver
  -> Asset Resolver (exact match only)
  -> Policy Engine (basic authorization)
  -> Risk Engine
  -> Authorization Service (combine policy + risk + exemption)
  -> AssistantOrder
  -> Plan Builder
  -> if approval required:
         WAITING_APPROVAL
         -> approval approved
         -> APPROVED
     else:
         APPROVED (approval_status = NOT_REQUIRED)
  -> explicit execute trigger
  -> Plan Re-validate
  -> Execution Router
  -> Task Runtime
  -> Audit Ledger
  -> Evidence Pack
```

强制要求：

1. `Policy Engine` 与 `Risk Engine` 不是平级终态裁决器。
2. 所有最终放行/拒绝/需审批结论都必须由 `AuthorizationService` 统一输出。
3. 审批通过不自动执行。
4. 执行必须由显式 `execute` 触发，且触发人必须是已认证并满足执行角色要求的受控 Principal。
5. `Plan Re-validate` 失败时，Order 必须进入 `PLAN_STALE`，而不是被归类为 `FAILED`。
6. EvidencePack 必须覆盖成功与失败两条结束路径。

---

## 4. 核心设计原则

### 4.1 Agent 与执行权分离

Deep Agent 负责理解与规划，不直接拥有最终执行权。

### 4.2 控制层统一收口

所有与企业执行安全相关的能力必须统一收敛在 Assistant Control Layer 中，包括：

- 权限
- 风险
- 审批
- 审计
- 幂等
- 证据

### 4.3 动作统一，高于具体工具

系统内部必须使用统一的 Action，而不是让 Agent 直接依赖：

- 某个 HTTP path
- 某个 SQL
- 某个脚本名
- 某个 MCP tool 名字

### 4.4 执行器可插拔

同一个 Action，未来可以路由到不同执行器。

例如：

- 走 MCP Tool Adapter
- 走 DB-native Adapter
- 走 CRD Adapter
- 走 gRPC Adapter

### 4.5 资产先于执行

所有执行必须先命中受控资产对象。

系统不允许默认面向：

- 裸数据库实例
- 裸主机
- 裸 K8s 集群
- 裸脚本入口

### 4.6 授权结论必须统一产生

Policy、Risk、Exemption、Approval Requirement 的组合结论必须通过 `AuthorizationService` 统一生成。

不允许：

- 调用方自己拼 Policy 结果和 Risk 结果
- 不同模块各自约定先后顺序
- 上层 Skill 自行绕过主链路判断

### 4.7 审批不等于执行

审批通过只表示“允许进入执行候选状态”，不表示“已经开始执行”。

### 4.8 执行触发也必须受控

执行触发人本身也是受控身份，不是一个可以由客户端随意伪造的字符串。

系统必须满足：

- `execute` 调用人来自认证上下文。
- `order_id` 本身不构成执行授权。
- 只有满足执行触发角色要求的 Principal 才能触发已批准工单执行。

### 4.9 Plan 必须可审批且可复核

ExecutionPlan 必须在审批前生成并冻结成审批附件。

审批通过后，执行前只允许做轻量 `re-validate`，不默认生成另一份新计划。

若 `re-validate` 失败，则：

- `ExecutionPlan` 进入 `STALE`
- `AssistantOrder` 进入 `PLAN_STALE`
- 旧工单不继续执行
- 若用户仍希望完成动作，应重新发起新的请求

### 4.10 审计必须不可变追加

AuditLedger 的底层写入模型必须是不可变追加型事件日志。

系统允许提供便于查询的聚合视图，但该视图不是审计源记录。

### 4.11 证据必须对称

成功执行与失败执行都必须留下可审计、可追溯的 EvidencePack。

---

## 5. 核心概念模型

## 5.1 Action

Action 是系统内部的标准动作语义。

示例：

- `mysql.database.create`
- `mysql.user.create`
- `mysql.user.grant`
- `mysql.password.change`
- `mysql.backup.create`
- `resource.cluster.register`

Action 必须满足：

- 可读
- 可稳定引用
- 与底层实现解耦

## 5.2 Skill

Skill 是面向 Deep Agent 的高层能力入口。

建议使用“request_动作名”形式，例如：

- `request_mysql_database_create`
- `request_mysql_user_create`
- `request_mysql_backup`

Skill 的作用是：

> 让 Deep Agent 以稳定方式向 Control Layer 提交动作申请。

## 5.3 Adapter

Adapter 是执行层插件。

类型包括：

- MCP Tool Adapter
- DB-native Adapter
- CRD Adapter
- GrpcCall Adapter
- K8s Adapter
- Shell/Ansible Adapter
- VM/SSH Adapter

Adapter 不直接暴露给 Deep Agent。

## 5.4 Asset

Asset 指系统中被纳管的对象。

首批资产类型建议：

- Project
- Cluster
- Environment
- NodeGroup
- Node
- StorageClass
- ServiceGroup
- ServiceInstance
- Unit
- DatabaseTarget

## 5.5 Principal

Principal 表示本次请求的主体。

包括：

- 用户 ID
- 角色集合
- 所属组
- 资源范围
- 审批角色
- 豁免标记

## 5.6 AuthorizationDecision

AuthorizationDecision 是控制层对外唯一权威的授权结论对象。

至少应包含：

- `allowed`
- `deny_reason`
- `risk_level`
- `approval_required`
- `approval_policy_ref`
- `exemption_flags`
- `decision_source`

## 5.7 AssistantOrder

正式的控制流对象。

当一个 ActionRequest 进入控制层并完成资产解析、权限和风险判定后，应转化为 AssistantOrder。

## 5.8 ExecutionPlan

ExecutionPlan 表示可审阅、可冻结、可在执行前复核的执行计划对象。

## 5.9 ExecutionTask / ExecutionStep

ExecutionTask 表示一次真实执行任务。

ExecutionStep 表示任务中的某个具体步骤。

## 5.10 AuditLedger

记录完整动作链路的审计账本。

## 5.11 EvidencePack

记录执行证据的归档对象。

必须覆盖：

- 执行前状态
- 执行后状态
- 成功/失败结论
- 失败原因
- 审批引用
- 日志引用

---

## 6. Assistant Control Layer 设计要求

Assistant Control Layer 是本项目的核心。

### 6.1 它必须完成的事情

1. 接收标准动作请求。
2. 识别本次请求的 Principal。
3. 将目标描述解析成具体 Asset。
4. 做基础动作权限判断。
5. 做资源范围判断。
6. 做风险等级判断。
7. 通过 `AuthorizationService` 统一合并最终控制结论。
8. 创建 AssistantOrder。
9. 生成可审批的 ExecutionPlan。
10. 在需要时发起审批。
11. 在显式 execute 时做 Plan Re-validate。
12. 选择具体 Adapter。
13. 创建 ExecutionTask / Step。
14. 写入 AuditLedger。
15. 在成功或失败后都生成 EvidencePack。

### 6.2 它不应该做的事情

1. 不直接承担自然语言生成。
2. 不取代 Deep Agent 的上下文理解。
3. 不把高危执行器直接暴露给 Agent。
4. 不跳过权限与审批。
5. 不在 Asset Resolver 内部做模糊猜测。
6. 不在审批通过后默认自动执行。

---

## 7. 权限 / 风险 / 审批模型

## 7.1 为什么需要独立权限模型

如果没有独立权限模型，Deep Agent 只能根据“能不能调到某个工具”来做事，这是不安全的。

企业场景需要判断：

- 谁能发起什么动作
- 谁能对哪些资源发起动作
- 哪些动作在哪些环境可以免审批进入可执行状态
- 哪些动作必须审批
- 谁能审批
- 哪些角色具备豁免资格

因此，本项目必须建设独立的 RBAC / ABAC 数据模型与策略体系。

## 7.2 授权主链路

v0.7 明确规定：

```text
Policy Engine (basic)
  -> Risk Engine
  -> AuthorizationService
  -> AuthorizationDecision
```

说明：

1. Policy 先解决“是否有基础权限、是否命中 scope、是否有显式 deny”。
2. Risk 再解决“该动作在该资产和参数条件下属于什么风险等级”。
3. AuthorizationService 最后合并：
   - Policy 决定
   - Risk 决定
   - 豁免规则
   - 是否需要审批
   - 是否允许进入执行候选状态

## 7.3 执行放行公式

```text
是否能进入执行 =
身份合法
× 资产精确命中
× 基础权限允许
× 风险策略允许或审批满足
× 显式 execute 触发
```

### 7.4 最小权限对象

#### Principal
字段建议：

- `principal_id`
- `principal_type`
- `user_id`
- `display_name`
- `roles[]`
- `groups[]`
- `scopes`
- `approval_roles[]`
- `exemption_flags[]`

#### Role
角色建议：

- `assistant_user`
- `dba`
- `mysql_operator`
- `backup_operator`
- `prod_approver`
- `platform_admin`
- `readonly_auditor`

#### ActionPolicy
定义某角色可否发起某个 Action。

#### ResourceScopePolicy
定义某角色可操作的资源范围。

#### RiskPolicy
定义某角色面对某风险等级时的策略：

- allow
- deny
- require_approval

#### ApprovalPolicy
定义：

- 哪些动作需要审批
- 谁可以审批
- 是否需要双人审批
- 是否允许紧急模式
- 是否禁止自我审批
- 审批有效期 `approval_ttl`

说明：

- `approval_ttl` 是审批等待窗口的权威来源，建议按动作 / 环境粒度配置。
- 若未命中更细粒度 `ApprovalPolicy`，一期可回退到平台全局默认 TTL。
- 后台过期扫描与 `APPROVAL_EXPIRED` 事件都应基于该 TTL 语义执行，而不是依赖实现内硬编码常量。

## 7.5 明确拒绝的几类情况

以下情况应直接拒绝，而不是进入执行：

- Principal 身份非法
- Asset 未命中
- Asset 命中多个对象
- Role / Scope 不允许
- 显式 deny
- 审批人等于发起人
- 工单未获批却尝试执行

---

## 8. 审批模型

## 8.1 审批触发条件

以下场景默认触发审批：

- 生产环境
- 高敏实例
- 中高风险动作（R2/R3）
- 密码/授权类敏感操作
- 恢复、扩缩容、重建、删除类操作

## 8.2 审批对象

审批对象应是 `AssistantOrder + 冻结的 ExecutionPlanSnapshot`，而不是某条底层 SQL 或某次工具调用。

### 8.3 审批运行时规则

1. `approver_id` 不允许等于 `created_by`。
2. 审批通过只改变审批状态和工单状态为 `APPROVED`，不自动执行。
3. 执行必须通过单独的 `execute` 触发。
4. `execute` 的触发主体必须来自认证上下文，并满足执行触发角色要求，例如 `mysql_operator`、`platform_admin` 或受控系统主体 `control_executor`。
5. 不得把客户端提交的 `executor_id` 当作可信授权依据；如接口需要传该字段，也只能作为与认证上下文做一致性比对的审计镜像。
6. 执行前必须重新校验 Plan 是否仍然可执行。
7. 若 `Plan Re-validate` 失败，则 `ExecutionPlan` 必须进入 `STALE`，`AssistantOrder` 必须进入 `PLAN_STALE`。
8. `PLAN_STALE` 不属于执行失败，不得与 `FAILED` 混用。
9. 审批记录必须保存审批时看到的风险快照和计划版本。
10. 推荐由后台调度任务周期性扫描 `WAITING_APPROVAL` 且已超过审批 TTL（优先取 `ApprovalPolicy.approval_ttl`，否则回退平台全局默认值）的工单，并将其标记为 `EXPIRED`。
11. 审批过期落账时必须同时追加 `APPROVAL_EXPIRED` 审计事件；若调度任务临时失败，后续补偿扫描仍必须能够收敛到同一终态。

## 8.4 Approval 状态机

```text
INIT -> NOT_REQUIRED
INIT -> WAITING_APPROVAL -> APPROVED
                        -> REJECTED
                        -> EXPIRED
```

状态语义补充：

1. `NOT_REQUIRED` 与 `WAITING_APPROVAL` 是两条互斥路径，不存在 `NOT_REQUIRED -> APPROVED` 的隐式转移。
2. `APPROVED` / `REJECTED` / `EXPIRED` 只能从 `WAITING_APPROVAL` 进入。
3. `APPROVED` 仅表示审批结束且通过，不表示执行已开始。
4. 自我审批在进入状态机前即被拒绝。

## 8.5 AssistantOrder 状态机

```text
DRAFT -> POLICY_REJECTED
DRAFT -> WAITING_APPROVAL -> APPROVED -> EXECUTING -> SUCCEEDED / FAILED
DRAFT -> APPROVED -> EXECUTING -> SUCCEEDED / FAILED
DRAFT -> CANCELLED
WAITING_APPROVAL -> REJECTED
WAITING_APPROVAL -> EXPIRED
APPROVED -> PLAN_STALE
```

状态语义补充：

1. `EXECUTING` 只能由显式 `execute` 触发进入。
2. `WAITING_APPROVAL` 状态若审批超时，工单必须进入 `EXPIRED` 终态，不得无限停留在等待态。
3. `APPROVED` 状态下若 `Plan Re-validate` 失败，工单进入 `PLAN_STALE`，而不是 `FAILED`。
4. `PLAN_STALE` 表示“冻结的计划已不再适用于当前资产状态”，此时不得继续执行旧工单。
5. `PLAN_STALE` 的推荐处理路径是：保留旧工单用于审计，若仍需完成目标动作，则重新发起新的 `ActionRequest`。
6. `EXECUTING` 状态下重复 `execute` 不应产生第二个任务。
7. `FAILED` 状态默认不等于可自动重试；是否重试由后续版本单独定义。

## 8.6 ApprovalRecord 字段建议

- `approval_id`
- `order_id`
- `approver_id`
- `decision`
- `comment`
- `decided_at`
- `risk_snapshot`
- `plan_id`
- `plan_version`

---

## 9. 审计与证据模型

## 9.1 AuditLedger 写入模型

v0.7 中，AuditLedger 的底层模型必须是**不可变追加型事件日志**。

这意味着：

1. 每个关键生命周期事件都追加一条新的 `AuditEvent`。
2. 已落盘的审计事件不得原地修改。
3. 同一个 `request_id` 或 `order_id` 与审计事件之间是 **1:N** 关系。
4. 对外可提供 `AuditLedgerView` 作为聚合视图，但该视图是派生读模型，不是审计源记录。

最小事件集合建议包括：

- `REQUEST_ACCEPTED`
- `AUTHORIZATION_DECIDED`
- `ORDER_CREATED`
- `PLAN_FROZEN`
- `APPROVAL_CREATED`
- `APPROVAL_APPROVED`
- `APPROVAL_REJECTED`
- `APPROVAL_EXPIRED`
- `EXECUTE_TRIGGERED`
- `PLAN_REVALIDATED`
- `PLAN_STALE`
- `EXECUTION_STARTED`
- `EXECUTION_SUCCEEDED`
- `EXECUTION_FAILED`
- `EVIDENCE_WRITTEN`

## 9.2 AuditEvent

字段建议：

- `event_id`
- `request_id`
- `order_id`
- `task_id`
- `event_type`
- `principal_id`
- `approval_actor_id`
- `execute_actor_id`
- `raw_user_prompt`
- `normalized_action`
- `resolved_asset_ids`
- `risk_level`
- `policy_decision`
- `authorization_decision`
- `approval_status`
- `order_status`
- `selected_adapter`
- `execution_summary`
- `success`
- `error_code`
- `error_message`
- `trace_id`
- `created_at`

写入语义约束：

1. `AuditEvent` 采用**按事件增量记录**语义，而不是“每条事件都携带完整链路快照”。
2. 请求级不变字段，例如 `raw_user_prompt`、`normalized_action`、`resolved_asset_ids`，建议只在 `REQUEST_ACCEPTED` 或最早可确定的事件中写入一次。
3. 阶段级字段，例如 `approval_status`、`order_status`、`task_id`、`selected_adapter`、`execution_summary`，由各阶段事件按需写入。
4. 对于与当前事件无关的字段，可以置为 `null` 或在序列化层省略；聚合展示由 `AuditLedgerView` 完成，不要求单条事件始终自包含全部状态。
5. `AuditEvent` 的设计目标是忠实反映“本阶段发生了什么变化”，而不是替代读模型。
6. 建议 `REQUEST_ACCEPTED` 只记录受理阶段字段；`risk_level`、`policy_decision`、`authorization_decision` 等授权结论应由 `AUTHORIZATION_DECIDED` 事件写入。
7. 在 v0.7 中，`REQUEST_ACCEPTED` 的推荐边界定义为：**请求受理 + 动作归一化完成 + 资产精确解析完成**；因此 `resolved_asset_ids` 可以在该事件首次写入。若该时点尚未可靠解析出资产，也可以保持 `null`，并在后续首个可确定事件中补写。

## 9.3 AuditLedgerView

字段建议：

- `request_id`
- `latest_order_id`
- `latest_task_id`
- `latest_order_status`
- `latest_approval_status`
- `latest_execution_summary`
- `latest_success`
- `latest_error_code`
- `latest_error_message`
- `event_count`
- `last_event_at`

说明：

- `AuditLedgerView` 只用于便捷查询与页面展示。
- 它可以通过追加事件流实时聚合得到。
- 它不能替代底层 `AuditEvent` 作为审计依据。

## 9.4 EvidencePack

字段建议：

- `evidence_id`
- `order_id`
- `task_id`
- `artifact_refs[]`
- `request_summary`
- `before_state_snapshot`
- `after_state_snapshot`
- `log_refs[]`
- `approval_refs[]`
- `result_summary`
- `execution_success`
- `failure_detail`
- `rollback_suggestion`

强制要求：

1. EvidencePack 在成功和失败两条路径都必须生成。
2. `execution_success=false` 时，`failure_detail` 不得为空。
3. 若执行尚未真正开始、但因 `PLAN_STALE` 被阻断，也应生成对应该工单的失败证据包，并明确“未启动执行任务”的原因。
4. `PLAN_STALE` 场景下允许 `task_id=null`，因为此时不应创建 `ExecutionTask`。
5. `PLAN_STALE` 场景下，`before_state_snapshot` / `after_state_snapshot` 可为 `null` 或空对象，取决于 `re-validate` 阶段是否成功采集到足够快照。
6. EvidencePack 与 AuditLedger 必须在语义上对称，不允许“有失败审计但无失败证据包”。

---

## 10. 核心运行时对象

## 10.1 ActionRequest

表示 Deep Agent 提交的标准动作申请。

字段建议：

- `request_id`
- `principal_id`
- `action_name`
- `resource_selector`
- `parameters`
- `context`
- `source`
- `created_at`

### resource_selector 约束

v0.7 最小字段建议：

- `project`
- `environment`
- `service_instance`

说明：

1. `service_instance` 必须是平台受控目录中的**规范名（canonical name）**。
2. Asset Resolver 只做严格精确匹配。
3. 用户自然语言别名不直接进入 Resolver 主链路。
4. 如存在“订单库主库”“order 主实例”这类表达，应由上层 Agent 或独立 Asset Search 能力先转换为规范名。

## 10.2 AssistantOrder

表示正式进入控制流后的工单对象。

字段建议：

- `order_id`
- `request_id`
- `action_name`
- `resolved_assets`
- `risk_level`
- `approval_required`
- `status`
- `plan_id`
- `plan_version`
- `created_by`
- `last_execute_triggered_by`
- `created_at`

字段语义约束：

1. `plan_id` 是 Order 绑定的冻结 `ExecutionPlan` 主键引用。
2. `plan_version` 是该绑定计划版本号的只读镜像，源自 `ExecutionPlan.plan_version`。
3. 执行前必须先通过 `plan_id` 取回对应 Plan，再将 `AssistantOrder.plan_version` 与 `ExecutionPlan.plan_version` 做一致性校验。
4. `PLAN_STALE` 状态下，Order 继续保留其 `plan_id` / `plan_version`，以支持后续审计与排障。

## 10.3 ExecutionPlan

字段建议：

- `plan_id`
- `order_id`
- `plan_version`
- `plan_status`
- `selected_route`
- `adapter_chain`
- `steps`
- `rollback_strategy`
- `idempotency_strategy`
- `snapshot_frozen`
- `validated_at`
- `stale_reason`

字段语义约束：

1. `ExecutionPlan.plan_version` 是 Plan 聚合根自己的版本戳，也是审批快照引用的源版本。
2. v0.7 默认一个 Order 只绑定一个冻结 Plan；若资产状态变化导致计划失效，不在原 Order 内重建新 Plan，而是把旧 Plan 标记为 `STALE`。
3. `ExecutionPlan` 的推荐生命周期为：`DRAFT -> FROZEN`；执行前复核失败时 `FROZEN -> STALE`；复核通过时 `FROZEN -> REVALIDATED`；正式启动任务后 `REVALIDATED -> CONSUMED`；若在复核通过后、任务真正启动前又被判定不可继续执行，可使用 `REVALIDATED -> STALE` 作为阻断回退。
4. 若仍需继续执行相同业务目标，应创建新的 Order 与新的 Plan，而不是修改旧 Plan 覆盖审计痕迹。

## 10.4 ExecutionTask

字段建议：

- `task_id`
- `order_id`
- `action_name`
- `status`
- `heartbeat_at`
- `started_at`
- `ended_at`

## 10.5 ExecutionStep

字段建议：

- `step_id`
- `task_id`
- `priority`
- `adapter_type`
- `operation`
- `status`
- `timeout_seconds`
- `error_message`

---

## 11. 动作分层

首版动作建议分为三类。

## 11.1 同步动作

特点：

- 可快速返回结果
- 执行路径较短
- 常用于数据库逻辑对象操作

示例：

- `mysql.database.create`
- `mysql.user.create`
- `mysql.password.change`

## 11.2 任务型动作

特点：

- 有异步执行过程
- 需要轮询状态
- 可能无审批，但必须有任务跟踪

示例：

- `mysql.backup.create`
- `mysql.restore.create`
- `mysql.unit.rebuild`

## 11.3 工单 + 任务型动作

特点：

- 必然进入审批与执行流程
- 通常涉及基础设施与服务实例变更
- 常带多个步骤与子任务

示例：

- `mysql.service.create`
- `mysql.service.scale`
- `resource.cluster.register`

---

## 12. 执行适配器设计

## 12.1 适配器类型

### MCP Tool Adapter
用于调用 MCP Server 提供的 tools。

### DB-native Adapter
用于直接连接数据库执行受控动作。

### CRD Adapter
用于对 Kubernetes 自定义资源执行 apply/query/update 等操作。

### GrpcCall Adapter
用于调用企业内部标准化 gRPC 执行接口。

### K8s Adapter
用于查询或操作 Kubernetes 原生资源。

### Shell/Ansible Adapter
用于兼容传统脚本与自动化执行方式。

### VM/SSH Adapter
用于兼容非 K8s 的虚拟机执行场景。

## 12.2 适配器设计原则

1. 只做执行，不做自然语言理解。
2. 不负责最终权限判断。
3. 不直接暴露给 Deep Agent。
4. 必须支持返回统一执行结果。
5. 必须支持幂等控制、日志引用和前后状态证据采集。

---

## 13. MCP 兼容策略

MCP 不是控制层替代品，而是一类执行通道。

同一个 Action 可以路由到不同通道，例如：

- `mysql.database.create` -> DB-native Adapter 或 MCP Tool Adapter
- `mysql.backup.create` -> MCP Tool Adapter 或 gRPC Adapter
- `mysql.service.create` -> CRD Adapter

系统必须保证：

- 不管走哪条执行路径，权限、审批、审计、证据都走同一个 Control Layer。

---

## 14. MVP 范围

## 14.1 一期范围

v0.7 最小落地范围：

1. Principal 输入与识别
2. 最小 RBAC / Risk / Approval 能力
3. `AuthorizationService`
4. Action Dictionary 首批动作
5. Asset Resolver 最小实现（严格精确匹配）
6. `mysql.database.create`
7. 审计账本
8. 成功/失败对称的简版证据包
9. 基础任务状态查询
10. execute 触发链路

## 14.2 一期默认策略

- dev/test：R1 无需审批，但仍须通过显式 `execute` 触发
- prod：默认 require approval
- 密码/高权限授权：默认 require approval
- 恢复/扩缩容/重建：默认 require approval + evidence required

## 14.3 一期平台命名规范

以 `mysql.database.create` 为例，建议平台命名规范为：

- `database_name` 以字母开头
- 仅允许字母 / 数字 / 下划线
- 最大 64 个 ASCII 字符

建议 regex：

```text
^[A-Za-z][A-Za-z0-9_]{0,63}$
```

重要说明：

1. 这是**平台命名规范强约束**，不是 MySQL 语法完整边界。
2. MySQL 中某些通过 quoting 可成立的名称，平台 v0.7 不接受。
3. 中文数据库名即使底层理论上可表达，平台 v0.7 也不接受。
4. 当前约束是 ASCII-only，因此这里的“最大 64 个 ASCII 字符”与 MySQL 的“最大 64 字节”在一期实现中恰好等价；若未来放开 Unicode，校验器必须改为按字节长度而不是按字符数判断。
5. 返回给用户的错误语义应是“违反平台命名规范”，而不是“数据库语法非法”。

---

## 15. MVP 动作示例：mysql.database.create

### 15.1 业务语义

在一个已纳管 MySQL 目标上创建逻辑数据库。

### 15.2 期望调用链

```text
User
  -> Deep Agent
     -> request_mysql_database_create
        -> Assistant Control Layer
           -> normalize to mysql.database.create
           -> resolve principal
           -> resolve target asset (exact match)
           -> policy basic allow/deny
           -> evaluate risk
           -> build AuthorizationDecision
           -> create AssistantOrder
           -> build and freeze ExecutionPlan
           -> if approval required:
                -> create approval order
                -> WAITING_APPROVAL
                -> approval approved
                -> APPROVED
                -> explicit execute by authorized executor
              else:
                -> APPROVED (approval_status = NOT_REQUIRED)
                -> explicit execute by authorized executor
           -> plan re-validate
           -> if stale: mark order PLAN_STALE and append audit/evidence
           -> else choose DB-native or MCP adapter
           -> execute create database
           -> append audit events
           -> build evidence pack (success/failure)
  -> Deep Agent returns final message
```

交互补充说明：

- 控制层语义仍然要求 `request` 与 `execute` 是两次独立的受控调用。
- 但在 `approval_required = false` 且风险等级允许的场景下，Deep Agent 可以在收到 `APPROVED` 响应后，**自动紧接着调用** `execute_assistant_order`。
- 这种“自动串联两次 Skill 调用”的行为属于 Agent 层编排优化，不改变控制层“执行必须由显式 execute 触发”的原则。

### 15.3 必要输入

- 请求主体
- 目标项目/环境/实例规范名
- 数据库名

### 15.4 最小校验

- 目标实例存在且唯一命中
- Principal 有动作权限
- Principal 在该资源范围内
- 动作未重复执行
- 若命中 prod/high-sensitive 策略，则必须审批
- 审批通过后，仍需由具备执行角色的受控主体显式 execute
- execute 前必须完成 Plan Re-validate
- 若 Re-validate 失败，工单进入 `PLAN_STALE`，需重新发起新请求

---

## 16. 数据存储要求

系统至少需要一套数据库，保存以下内容：

- Principal
- Role
- ActionPolicy
- ResourceScopePolicy
- RiskPolicy
- ApprovalPolicy
- Asset Catalog
- ActionRequest
- AssistantOrder
- ExecutionPlan / ExecutionPlanSnapshot
- ExecutionTask / Step
- AuditEvent / AuditLedgerView
- EvidencePack

数据库类型可选：

- MySQL
- PostgreSQL

如果希望后续策略表达更灵活，可优先考虑 PostgreSQL；如果团队更熟悉 MySQL，也可先用 MySQL 落地 MVP。

---

## 17. 后续文档拆分建议

本 spec 之后，建议继续补充以下文档：

1. `04-interface-design-v0.8.md`
2. `05-action-dictionary-v0.1.md`
3. `06-control-layer-schema-v0.1.md`
4. `07-rbac-model-v0.1.md`
5. `08-adapter-interface-v0.1.md`
6. `09-mysql-database-create-sequence.md`

---

## 18. 对 Coding Agent 的直接要求

当 Coding Agent 读取本 spec 时，应遵循以下要求：

1. 默认以本文档定义的概念为准。
2. 不假设存在任何历史平台运行时。
3. 不把历史平台专有对象名当作实现前提。
4. 优先先做 Control Layer 的核心对象和控制流程。
5. 不允许让 Deep Agent 直接持有高危底层凭证与执行权。
6. 所有执行能力必须经过 `Action -> Asset -> AuthorizationService -> Approval -> Execute -> Adapter` 这条主链路。
7. `Asset Resolver` 不允许模糊匹配和自动猜测。
8. 审批通过不得自动触发执行。
9. 必须同时实现成功和失败路径的审计与证据固化。
