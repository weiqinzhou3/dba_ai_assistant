# Codex 分段执行手册 v0.1

## 0. 文档定位

本文档是给 Coding Agent（Codex / Claude Code）使用的**执行手册**。

它不是正式领域规格，也不是代码接口文档。  
它的作用是：

- 把已有 spec / interface design / schema 文档，转换成可执行的阶段任务
- 避免 Coding Agent 一上来就自由发挥
- 配合 superpowers / phased execution 方式，按门禁推进

---

## 1. 为什么需要分段执行手册

因为本项目不是单一 CRUD 服务，而是：

- 上层有 Deep Agent
- 中间有 Control Layer
- 下层有可插拔 Adapter
- 还包含权限、风险、审批、审计、证据等受控边界

如果没有分段执行手册，Coding Agent 很容易出现：

1. 直接先写 controller
2. 跳过核心对象和状态机
3. 把审批和 execute 混成一个接口
4. 把 DB 直写逻辑塞进 handler
5. 不写失败审计和失败证据

所以，必须按阶段推进。

---

## 2. 总体阶段规划

建议采用 6 个阶段：

- Phase 0：对齐与门禁
- Phase 1：领域对象与接口骨架
- Phase 2：控制主链路
- Phase 3：DB-native Adapter 与执行闭环
- Phase 4：审计与证据闭环
- Phase 5：验收与收口

每个阶段都要求：
- 明确输入
- 明确输出
- 明确不做什么
- 明确通过门禁后才能进入下一阶段

---

## 3. Phase 0：对齐与门禁

### 目标
确认 Coding Agent 对文档理解一致，避免错方向。

### 必读文档顺序
1. `docs/00-project-context.md`
2. `docs/01-glossary.md`
3. `docs/02-reference-platform-background.md`
4. `docs/03-assistant-spec-v0.7.md`
5. `docs/04-interface-design-v0.8.md`
6. `docs/05-control-layer-schema-v0.1.md`
7. `docs/06-rbac-model-v0.1.md`
8. `docs/07-adapter-interface-v0.1.md`
9. `docs/08-mysql-database-create-sequence.md`

### 必须输出
- `docs/implementation-alignment.md`
- `docs/IMPLEMENTATION_STATUS.md`

### alignment 文档必须回答
1. 主链路是什么
2. Deep Agent 与 Control Layer 的边界是什么
3. Skill / Action / Adapter 的关系是什么
4. 为什么审批通过不自动执行
5. 为什么 Resolver 不能模糊匹配
6. 一期为什么只做 `mysql.database.create`

### 门禁
如果 alignment 文档里没有准确写出：
- `ActionRequest -> AuthorizationDecision -> AssistantOrder -> ExecutionPlan -> Approval/Execute -> ExecutionTask -> Audit/Evidence`
则不得进入 Phase 1。

---

## 4. Phase 1：领域对象与接口骨架

### 目标
先固化对象模型和接口，不写完整业务逻辑。

### 必做内容
1. 建模块化单体目录
2. 定义领域对象
3. 定义 service interface
4. 定义 repository interface
5. 定义 northbound API contract
6. 定义 southbound adapter SPI

### 推荐目录
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

### 本阶段禁止
- 不要直接落完整 SQL 执行
- 不要实现全部 Adapter
- 不要做模糊资产搜索
- 不要上来做 UI

### 产物
- 代码骨架可编译
- 核心接口存在
- 结构与文档一致

### 门禁
必须满足：
1. 先有 domain / application / adapter 边界
2. 存在 `AuthorizationService`
3. 存在 `ExecutionPlanner`
4. 存在 `DBNativeAdapter` 接口占位
5. 所有返回对象带 `trace_id`

---

## 5. Phase 2：控制主链路

### 目标
打通 request -> authorization -> order -> plan 的上半段链路。

### 必做内容
1. `POST /api/v1/action-requests`
2. Action Normalizer
3. Principal Resolver
4. Asset Resolver（严格精确匹配）
5. Policy Engine
6. Risk Engine
7. AuthorizationService
8. Order Service
9. Plan Builder

### 本阶段不要求
- 不要求真正执行 SQL
- 不要求真正连接数据库
- 不要求证据制品丰富化

### 验收要点
1. dev/test 路径可返回 `APPROVED`
2. prod 路径可返回 `WAITING_APPROVAL`
3. 明确 `POLICY_REJECTED`
4. 能正确生成 `ExecutionPlan`
5. 审批与 execute 仍然分开

### 门禁
只有当：
- `AuthorizationDecision` 已成为唯一权威结果
- `AssistantOrder` 和 `ExecutionPlan` 可持久化
- `Asset Resolver` 无模糊匹配
才允许进入 Phase 3。

---

## 6. Phase 3：DB-native Adapter 与执行闭环

### 目标
打通 execute -> task -> dbnative 的最小执行链路。

### 必做内容
1. `POST /api/v1/orders/{order_id}/execute`
2. Execute 权限校验
3. Plan Re-validate
4. Execution Router（MVP 固定到 db_native）
5. Task Runtime
6. DB-native Adapter
7. create database 最小闭环

### DB-native Adapter 最小步骤
1. validate_target
2. check_database_not_exists
3. create_database
4. verify_database_created

### 本阶段禁止
- 不要同时做 MCP/CRD/gRPC
- 不要自动重试写操作
- 不要让 execute 跳过 re-validate

### 门禁
必须满足：
1. 审批通过不自动执行
2. execute 必须显式调用
3. `PLAN_STALE` 可见
4. `EXECUTING` 状态重复 execute 幂等
5. create database 能形成真实结果

---

## 7. Phase 4：审计与证据闭环

### 目标
把成功和失败两条路径都做成可追踪闭环。

### 必做内容
1. `AuditEvent` append-only
2. `AuditLedgerView`
3. `EvidencePack`
4. 失败路径证据
5. `PLAN_STALE` 路径证据

### 最小审计事件
- `REQUEST_ACCEPTED`
- `AUTHORIZATION_DECIDED`
- `ORDER_CREATED`
- `PLAN_FROZEN`
- `APPROVAL_CREATED/APPROVED/REJECTED/EXPIRED`
- `EXECUTE_TRIGGERED`
- `PLAN_REVALIDATED`
- `PLAN_STALE`
- `EXECUTION_STARTED`
- `EXECUTION_SUCCEEDED/FAILED`
- `EVIDENCE_WRITTEN`

### 最小证据
- 请求摘要
- 目标实例
- before/after 状态
- SQL 摘要
- 审批引用
- 失败详情
- 回滚建议

### 门禁
只有当：
- 成功路径有审计和证据
- 失败路径有审计和证据
- `PLAN_STALE` 路径也有证据
才允许进入 Phase 5。

---

## 8. Phase 5：验收与收口

### 目标
把 MVP 做成可评审、可演示、可继续扩展的基线。

### 必做内容
1. 整理 `IMPLEMENTATION_STATUS.md`
2. 增补 API 示例
3. 增补最小运行说明
4. 增补失败路径说明
5. 列出后续动作扩展清单：
   - `mysql.user.create`
   - `mysql.user.grant`
   - `mysql.password.change`

### 验收问题
1. 是否真正遵守了主链路？
2. 是否真的没有让 Agent 直接持有执行权？
3. 是否真的把审批和 execute 分开？
4. 是否真的没有模糊资产匹配？
5. 是否真的把失败路径落账落证据？

---

## 9. implementation alignment 是什么

implementation alignment 不是正式 spec。  
它是 Coding Agent 在开始写代码前写的一份**实现对齐说明**。

它的作用是：

- 证明它真的读懂了文档
- 证明它没有误解边界
- 让人类在它写很多代码前，先发现偏差

### 为什么不是正式设计文档
因为它不定义新规则，只复述和落地已有规则。

### 为什么仍然有价值
因为 Coding Agent 很容易“自以为懂了”，然后直接开写。  
alignment 文档就是在正式编码前加一道门禁。

---

## 10. 给 Coding Agent 的工作方式建议

如果已经接入 superpowers skill，建议这样使用：

### 推荐模式
- 我们（人类/ChatGPT）先提供权威 spec 和分段执行手册
- Coding Agent 按 phase 工作
- 每个 phase 结束后输出：
  - 本阶段产物
  - 差异说明
  - 未实现项
  - 下一阶段建议

### 不推荐模式
- 直接让 Coding Agent“从文档出发把整个 MVP 全写了”
- 不设门禁
- 不做阶段验收

---

## 11. 对 Coding Agent 的最终要求

1. 必须按 phase 推进，不得跳阶段。
2. 每阶段结束必须写 `IMPLEMENTATION_STATUS.md`。
3. 若发现 spec/interface design/sequence 冲突，先停下来标注，不要擅自改语义。
4. 如果要新增文档，新增的是“实现说明”而不是重写正式 spec。
