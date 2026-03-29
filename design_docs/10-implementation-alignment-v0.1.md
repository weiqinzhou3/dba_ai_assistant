# Implementation Alignment v0.1

## 0. 文档目的

本文档不是新的正式 spec，也不修改 `03` 到 `09` 的既有语义。

它只回答两件事：

1. 我将如何把已有正式文档翻译成第一阶段可编码的骨架。
2. 我会用哪些明确边界，避免在本轮提前写成“看起来能跑、实际上架构已经歪掉”的实现。

本文档中的内容分两类：

- **代码事实**：直接来自 `03-assistant-spec-v0.7.md` 到 `09-codex-phased-execution-manual-v0.1.md` 的明确要求。
- **实现判断**：在不改变正式语义前提下，为了本轮 skeleton/guardrails 落地做出的工程化选择。

---

## 1. 权威主链路

### 1.1 代码事实

正式文档已经关闭了主链路歧义，本项目本轮必须围绕如下链路展开：

```text
ActionRequest
  -> AuthorizationDecision
  -> AssistantOrder
  -> ExecutionPlan
  -> Approval / Execute
  -> ExecutionTask
  -> Audit / Evidence
```

更细粒度的控制顺序是：

```text
ActionRequest
  -> Action Normalizer
  -> Principal Resolver
  -> Asset Resolver (exact match only)
  -> Policy Engine
  -> Risk Engine
  -> AuthorizationService
  -> AssistantOrder
  -> ExecutionPlanner.Build
  -> ApprovalService / explicit Execute
  -> ExecutionPlanner.Revalidate
  -> ExecutionRouter
  -> TaskRuntime
  -> AuditService
  -> EvidenceService
```

### 1.2 实现判断

本轮不会尝试做完整执行闭环，但会先把这条链路的对象和接口按顺序立住。

具体做法：

- 北向统一入口只保留 `POST /api/v1/action-requests`
- 审批与执行分别保留独立入口：
  - `POST /api/v1/orders/{order_id}/approvals`
  - `POST /api/v1/orders/{order_id}/execute`
- `ActionRequestService` 负责统一收入口、串起上半段控制链路
- `ExecutionPlanner`、`ExecutionRouter`、`TaskRuntime`、`AuditService`、`EvidenceService` 先提供 contract 和 stub，使主链路方向不再依赖后续“补洞”

---

## 2. Deep Agent、Skill、Action、Adapter、Control Layer 的关系

### 2.1 代码事实

正式语义是：

```text
Deep Agent
  -> Skill
     -> Control Layer
        -> Action
           -> Adapter
```

约束包括：

- Deep Agent 负责理解自然语言、补全参数、组织多步计划。
- Skill 是给 Deep Agent 调用的高层业务能力入口。
- Action 是系统内部稳定动作语义，不等于 HTTP path、SQL、脚本名或 tool 名。
- Adapter 是 Control Layer 调用的 southbound 执行插件，不暴露给 Deep Agent。
- Control Layer 是权限、风险、审批、执行、审计、证据的唯一受控收口层。

### 2.2 实现判断

本轮代码里我会把它们切成三层：

1. **Northbound API / DTO**
   - 给 Skill / HTTP 调用
   - 只接收标准化 `ActionRequestDTO`、审批 DTO、execute DTO

2. **Application Services**
   - 承接 Control Layer 主链路
   - 这里定义 `ActionRequestService`、`AuthorizationService`、`ApprovalService`、`ExecutionPlanner` 等接口

3. **Southbound Adapter SPI**
   - 统一 `Adapter` contract
   - 只提供 `DBNativeAdapter` 骨架实现

这样做的目的，是避免上层直接看到 Adapter，也避免 southbound 细节反推上层模型。

---

## 3. 为什么审批通过不等于执行

### 3.1 代码事实

`03`、`04`、`08` 都明确规定：

- 审批通过只表示工单进入 `APPROVED`
- 审批通过不自动启动执行
- 执行只能由权威 `POST /api/v1/orders/{order_id}/execute` 显式触发
- execute 调用人必须来自认证上下文并满足 execute policy
- execute 前必须做 `Plan Re-validate`

### 3.2 实现判断

本轮即使所有内部实现仍是 stub，我也会把以下语义直接编码为 guardrail：

- `ApprovalService.Decide(...)` 只更新审批状态与工单状态
- `ActionRequestService.ExecuteApprovedOrder(...)` 是唯一执行触发接口
- `WAITING_APPROVAL` 状态下 execute 返回 `APPROVAL_REQUIRED`
- `PLAN_STALE`、`REJECTED`、`POLICY_REJECTED` 等状态下 execute 返回 `ORDER_NOT_EXECUTABLE`

这样下一轮补真实执行时，不需要再拆已经耦合错误的入口语义。

---

## 4. 为什么 Asset Resolver 不能做模糊匹配

### 4.1 代码事实

正式文档明确要求：

- `AssetResolver` 只允许 exact match
- 不允许 fuzzy match、best effort、自动猜测、宽松大小写命中
- 命中 0 个返回 `ASSET_NOT_FOUND`
- 命中多个返回 `ASSET_AMBIGUOUS`
- 用户自然语言别名应在上层 Agent 或未来独立 `Asset Search API` 解决，而不是进入核心执行链路

### 4.2 实现判断

本轮我会把 `AssetResolver` 接口定义为：

```text
ResolveExact(...)
```

不会提供：

- `ResolveFuzzy(...)`
- `ResolveBestEffort(...)`
- `ResolveOrGuess(...)`

并且 stub / in-memory 实现也只接受 `project + environment + service_instance` 的严格匹配。

这样可以把“自然语言检索”与“受控执行对象解析”彻底分层，避免执行链路从第一版开始就带不确定性。

---

## 5. 为什么 AuthorizationService 必须成为唯一权威授权出口

### 5.1 代码事实

正式要求已经明确：

- `PolicyEngine` 先做基础权限与 scope 判断
- `RiskEngine` 再做风险判断
- `AuthorizationService` 最终合并 Policy、Risk、豁免、审批要求，生成唯一 `AuthorizationDecision`
- 不允许业务调用方自行拼接 `Policy + Risk`

### 5.2 实现判断

本轮代码将通过两个层面的设计把这个原则钉死：

1. **接口层**
   - 对上只暴露 `AuthorizationService.Evaluate(...)`
   - `PolicyEngine`、`RiskEngine` 作为内部依赖存在，不作为 northbound 主接口暴露最终裁决能力

2. **对象层**
   - `AuthorizationDecision` 作为唯一可驱动 `AssistantOrder` 状态的对象
   - `ActionRequestService` 不读取零散的 `PolicyDecision` / `RiskDecision` 直接做状态推进

这样后续即使扩展豁免规则、审批 TTL、执行角色校验，也不会让授权语义分叉到 handler 或 adapter。

---

## 6. 为什么本轮只先做骨架和门禁

### 6.1 代码事实

`09-codex-phased-execution-manual-v0.1.md` 已经明确本轮应停在：

- Phase 0：对齐与门禁
- Phase 1：领域对象与接口骨架

并且显式禁止：

- 直接落完整 SQL 执行
- 同时实现多个 adapter
- 跳过审批与 execute 分离
- 跳过审计与证据预留

### 6.2 实现判断

本轮如果直接写完整 `mysql.database.create`，会导致几个高概率偏航：

1. handler 先于领域对象成型，最终由 transport 反推模型
2. `DBNativeAdapter` 被误写成事实上的授权边界
3. execute 被偷塞回 request 接口
4. 审计与证据因为“先跑通再说”被延后，后面很难再自然嵌回主链路

因此本轮的目标不是“数据库真的建出来”，而是让下一轮纵切时：

- 主链路位置已经正确
- 状态边界已经固定
- Northbound / Control / Southbound 三层不会再互相污染

---

## 7. 第一阶段代码结构切分

### 7.1 代码事实

正式文档建议目录为：

```text
internal/
  api/
  application/
    actionrequest/
    approval/
    authorization/
    execution/
    audit/
    evidence/
  domain/
    action/
    asset/
    principal/
    order/
    plan/
    task/
    policy/
    risk/
    authorization/
  adapters/
    dbnative/
  persistence/
```

### 7.2 实现判断

我会按 Go 模块化单体落地，并补一个小型 `domain/common` 用于统一错误码和时间/ID 级基础值对象。

第一阶段各目录职责如下：

- `internal/api/`
  - HTTP handler、路由、错误响应映射
  - 不承载控制规则

- `internal/application/actionrequest/`
  - 统一 request intake
  - 负责把主链路上半段串起来

- `internal/application/approval/`
  - 审批 contract 与 stub
  - 保证审批与执行分离

- `internal/application/authorization/`
  - `PrincipalResolver`
  - `AssetResolver`
  - `PolicyEngine`
  - `RiskEngine`
  - `AuthorizationService`

- `internal/application/execution/`
  - `ExecutionPlanner`
  - `ExecutionRouter`
  - `TaskRuntime`
  - Adapter SPI

- `internal/application/audit/`
  - `AuditService`

- `internal/application/evidence/`
  - `EvidenceService`

- `internal/domain/...`
  - 稳定领域对象、枚举、DTO 所依赖的核心类型

- `internal/adapters/dbnative/`
  - `DBNativeAdapter` contract + stub
  - 本轮不写完整 `CREATE DATABASE`

- `internal/persistence/`
  - repository interface
  - in-memory skeleton store

---

## 8. 本轮落地范围声明

### 8.1 会完成的内容

- `design_docs/10-implementation-alignment-v0.1.md`
- Go 模块骨架
- domain objects / DTO / contracts
- northbound API skeleton
- `AuthorizationService` 唯一授权出口
- `AssetResolver` exact match guardrail
- 审批与 execute 分离入口
- `AuditService` / `EvidenceService` 主链路占位
- `DBNativeAdapter` southbound skeleton
- `IMPLEMENTATION_STATUS.md`

### 8.2 不会完成的内容

- 完整 `mysql.database.create` 执行逻辑
- 真实 MySQL 连接与 `CREATE DATABASE`
- MCP / CRD / gRPC / K8s / Shell / VM 多 adapter 并行接入
- fuzzy asset search
- 自动审批后自动执行

---

## 9. 本轮编码门禁清单

本轮编码完成前，我会逐项自检以下门禁：

1. 是否已经存在 `ActionRequest -> AuthorizationDecision -> AssistantOrder -> ExecutionPlan -> Approval/Execute -> ExecutionTask -> Audit/Evidence` 的对象链路。
2. `AuthorizationService` 是否是唯一最终授权出口。
3. `AssetResolver` 是否只有 exact match contract。
4. 审批 API 与 execute API 是否已经物理分离。
5. `AuditService` 和 `EvidenceService` 是否已经进入主链路 contract。
6. 是否只实现了一个 southbound adapter skeleton：`DBNativeAdapter`。
7. 是否没有提前实现完整 `mysql.database.create`。

---

## 10. Alignment Addendum

本节逐条回应 [design_docs/11-claude-code-alignment-review-v0.1.md](/Users/zqw/Desktop/Project/dba_ai_assistant/design_docs/11-claude-code-alignment-review-v0.1.md) §2 的 11 项约束。

### 10.1 ExecutePolicy 是独立策略对象

**代码事实**

- `06-rbac-model-v0.1.md` 已把 `ExecutePolicy` 定义为独立于 `ActionPolicy` 的策略层。
- 它回答的是“谁能触发已批准工单执行”，不是“谁能发起动作请求”。

**实现判断**

- 我接受这条补充约束，并将其落为独立 domain object。
- `ExecuteAuthorizationService` 作为 execute 路径的独立鉴权接口存在，不复用 request 路径上的 `ActionPolicy` 判定。
- 本轮不会做完整 execute 流程，但会把 execute 权限门禁独立接到 `ActionRequestService.ExecuteApprovedOrder(...)`。

### 10.2 Adapter 幂等键

**代码事实**

- `07-adapter-interface-v0.1.md` 的示例把 `idempotency_key` 放在 execution controls 中。
- review 要求在 Phase 1 skeleton 中把该字段显式化，避免后续遗漏。

**实现判断**

- 我接受这条补充，Phase 1 直接把 `IdempotencyKey` 放入 `AdapterExecutionRequest` 顶层字段。
- 同时保留 `ExecutionControls`，用于后续 timeout / retry 等控制。
- 本轮只会补 contract 与 helper，不会伪实现真实幂等状态机。

### 10.3 Adapter DryRun 能力

**代码事实**

- 正式文档已要求 Adapter SPI 预留 `DryRun(...)`。
- `04-interface-design-v0.8.md` 同时明确：DryRun 不要求在首轮对所有 adapter 立即做真实实现，但接口必须先存在。

**实现判断**

- 这条约束我接受，但说明一点：我上一轮代码里已经预留了 `DryRun(...)`。
- 本轮不把 DryRun 硬塞进控制主链路伪装成“已实现”，只补 Phase 1 所需的接口和占位。
- Phase 2 再把 “build 前预检” 真实接线。

### 10.4 Approval TTL 与过期扫描

**代码事实**

- `03-assistant-spec-v0.7.md` 与 `04-interface-design-v0.8.md` 都把 `ApprovalPolicy.approval_ttl` 定为审批过期窗口的权威来源。
- 审批超时后，正式 order 终态是 `EXPIRED`。
- `APPROVAL_EXPIRED` 是审计事件名，不是第二个 order status。

**实现判断**

- 我接受 TTL 与过期扫描这两个约束，并在 domain 层补 `ApprovalPolicy.TTL time.Duration`。
- 这里对 review 做一个技术性保留：
  - 不会新增与正式文档冲突的 `AssistantOrder.APPROVAL_EXPIRED` 状态
  - 会继续使用 order `EXPIRED` + audit event `APPROVAL_EXPIRED`
- 代码里会用 `// REVIEW:` 标明这个分歧，避免后续实现者误读。

### 10.5 风险等级映射规则

**代码事实**

- 风险等级应由 `RiskEngine` 按上下文动态计算，而不是由静态常量决定。
- `mysql.database.create` 的 MVP 规则至少需要区分非 prod 与 prod。

**实现判断**

- 我接受这条约束。
- 当前 skeleton 中 `StaticRiskEngine` 已按 asset 环境把 test/dev 视为 `R1`，prod 提升到 `R2`。
- 这仍然只是最小骨架，不代表 Phase 2 已完成全部风险策略。

### 10.6 `control_executor` 角色

**代码事实**

- `06-rbac-model-v0.1.md` 把 `control_executor` 作为 execute 路径的合法角色之一。

**实现判断**

- 我接受这条约束。
- Phase 1 应至少把它固化为角色常量，并让 execute authorization stub 识别它。
- 这样后续自动链式执行不会被骨架层先天排除。

### 10.7 PLAN_STALE 路径的 Evidence Pack

**代码事实**

- `PLAN_STALE` 是独立终态，不等于 `FAILED`。
- Phase 4 才要求为其生成独立 evidence 逻辑，且 `task_id = null`。

**实现判断**

- 我接受这条约束，但它主要属于 Phase 4 gate，不会在本轮假装补成完整业务。
- 本轮只保证：
  - `PLAN_STALE` 是独立 order status
  - `EvidenceService` contract 已存在
  - evidence model 允许 `task_id` 为空

### 10.8 Skill 函数签名与北向 API 映射

**代码事实**

- 正式文档已定义两个 northbound skill 语义：
  - `request_mysql_database_create`
  - `execute_assistant_order`

**实现判断**

- 我接受这条约束。
- Phase 1 会把这两个 skill 的输入输出 struct 显式建模，并提供从 application result 到 skill output 的映射 helper。
- Skill contract 是 Agent 视角的契约，HTTP API 只是其底层 transport。

### 10.9 REQUEST_ACCEPTED 是第一个审计事件

**代码事实**

- `08-mysql-database-create-sequence.md` 已明确：进入控制层后的第一件事就是落 `REQUEST_ACCEPTED`。

**实现判断**

- 这条约束是有效的，我已在 `Submit(...)` 里按该顺序落账。
- 本轮会补测试把它锁成回归不变量，而不是只靠人工阅读代码判断。

### 10.10 UPM 逆向工程的设计启示

**代码事实**

- `design_docs/upm/*` 不是运行时依赖，但提供了资产层级、服务聚合、工单快照、任务/子任务等设计启示。

**实现判断**

- 我接受“用于扩展点预留”的含义，但不会把 UPM 专有对象名搬进正式 runtime。
- Phase 1 的字段设计会优先保证：
  - order / evidence 可承载快照类信息
  - task / step 可承载 timeout / heartbeat / priority
  - asset / adapter model 不把 target 退化成裸机器

### 10.11 文档路径修正

**代码事实**

- 仓库中的正式文档实际位于 `design_docs/`，不是 `docs/`。

**实现判断**

- 我接受这条修正。
- 后续实现说明、状态文档和引用链接都统一使用 `design_docs/`。

---

## 11. Phase Gate Additions

在原有 phase gate 基础上，再补以下检查项：

### 11.1 Phase 1 Gate Additions

1. `ExecutePolicy` 作为独立 domain object 存在。
2. `AdapterExecutionRequest` 显式包含 `IdempotencyKey`。
3. Adapter SPI 显式包含 `DryRun(...)`。
4. `ApprovalPolicy` 显式包含 `TTL`。
5. order status 保持 `PLAN_STALE` 与 `EXPIRED` 独立；不把 `APPROVAL_EXPIRED` 错建成第二个 order status。
6. 角色常量中包含 `control_executor`。
7. northbound skill contract 已定义：
   - `request_mysql_database_create`
   - `execute_assistant_order`

### 11.2 Phase 2 Gate Additions

1. execute 请求走独立的 `ExecuteAuthorizationService` / `ExecutePolicy` 检查。
2. `RiskEngine` 基于 asset 环境动态计算风险等级。
3. `REQUEST_ACCEPTED` 在主链路入口先于其他业务逻辑写入。
4. DryRun 在 plan/build 预检阶段被真实接线。
5. 审批过期扫描逻辑存在，至少是可执行 stub。

### 11.3 Phase 3 Gate Additions

1. `DBNativeAdapter.Execute()` 检查幂等键。
2. `DBNativeAdapter.DryRun()` 实现真实预检逻辑。
3. `control_executor` 可作为合法 execute 主体。

### 11.4 Phase 4 Gate Additions

1. `PLAN_STALE` 路径生成独立 `EvidencePack`，且 `task_id = null`。
2. `APPROVAL_EXPIRED` 路径生成审计事件。
3. `SUCCEEDED`、`FAILED`、`PLAN_STALE` 三条结束路径均有独立 evidence 逻辑。
