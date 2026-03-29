# Execution Roadmap v0.1

## 0. 文档定位

本文档基于以下正式文档整理：

- `design_docs/03-assistant-spec-v0.7.md`
- `design_docs/04-interface-design-v0.8.md`
- `design_docs/05-control-layer-schema-v0.1.md`
- `design_docs/06-rbac-model-v0.1.md`
- `design_docs/07-adapter-interface-v0.1.md`
- `design_docs/08-mysql-database-create-sequence.md`
- `design_docs/09-codex-phased-execution-manual-v0.1.md`
- `design_docs/10-implementation-alignment-v0.1.md`

本文档的目标不是新增领域规则，而是把现有正式约束整理成后续多轮编码可执行的阶段路线图。

本路线图默认前提：

1. 不假设任何历史 Java platform runtime 存在。
2. 所有实现都围绕本项目自己的 Control Layer / Action / RBAC / Audit / Evidence / Adapter 模型展开。
3. 审批通过不自动执行，执行必须走显式 `execute`。
4. Asset Resolver 只允许 exact match，不允许 fuzzy match。

---

## 1. 当前基线判断

基于当前仓库与 `IMPLEMENTATION_STATUS.md`，可以做出以下工程判断：

1. 当前仓库已经存在 Go 模块、模块化单体目录、领域对象、application contract、northbound API skeleton、southbound `DBNativeAdapter` skeleton。
2. 当前仓库已经具备较多 `Phase 1` 产物，但还没有把持久化、审批共享状态、execute 真正启动任务、append-only 审计、真实证据固化打通。
3. 因此，后续编码工作的**推荐切入点**不是重新做一遍 skeleton，而是按本路线图先补 `Phase 1` 验收清点，然后直接进入 `Phase 2`。

一句话判断：

> 当前项目处于“Phase 1 基本有骨架，但尚未进入真正可运行控制链路”的状态。

---

## 2. 推荐实现语言

### 2.1 拍板结论

推荐实现语言：**Go**

### 2.2 理由

1. `04-interface-design-v0.8.md` 与当前仓库代码都已经以 Go 接口风格表达 northbound service、authorization、planner、adapter SPI，继续使用 Go 可以避免接口语义二次翻译。
2. 当前项目最核心的是一个**受控控制后端**，需要清晰的领域对象、显式接口边界、并发安全和单二进制可部署性。Go 对模块化单体、HTTP API、后台调度、adapter 接口和轻量运行时都很合适。
3. Go 的静态类型和显式错误处理更适合本项目这种“状态机 + 权限 + 审计 + 证据”的控制系统，能减少把审批、执行、失败路径写散的风险。
4. 当前仓库已经有 `go.mod`、`cmd/`、`internal/` skeleton。此时改语言会把 roadmap 变成迁移项目，而不是交付 MVP。

结论：

> 后续路线图应以 Go 模块化单体为唯一实现语言，不再提供多语言候选。

---

## 3. 分阶段实施总览

| Phase | 名称 | 目标摘要 | 当前建议 |
|---|---|---|---|
| Phase 1 | 骨架与门禁 | 固化对象、接口、状态机、错误模型与 guardrails | 以“补验收和收口”为主 |
| Phase 2 | 最小控制链路可跑通 | 打通 request -> authorization -> order -> approval -> execute 的受控链路 | 当前最优先 |
| Phase 3 | `mysql.database.create` MVP 纵切片 | 接入真实 `DBNativeAdapter`，完成一个真实动作闭环 | Phase 2 后立即进入 |
| Phase 4 | Deep Agent 最小接入 | 通过 Skill/HTTP 把 Agent 接到 Control Layer，而不是反过来 | 在 Phase 3 稳定后进入 |
| Phase 5 | MCP 兼容接入 | 增加第二条 southbound 执行通道，但不改变 Control Layer 主语义 | 最后进入 |

说明：

1. 本路线图只定义 5 个核心实施阶段。
2. “读文档、做 alignment、写 roadmap” 视为进入编码前的预备门禁，不单列为 Phase 0。
3. 任何阶段都不得跳过前序门禁直接推进。

---

## 4. Phase 1：骨架与门禁

### 4.1 目标

把所有后续编码必须依赖的边界钉死，确保后续实现不会把 handler、adapter、审批、execute 混写。

### 4.2 范围

1. 模块化单体目录结构。
2. 核心领域对象：
   - `ActionRequest`
   - `AuthorizationDecision`
   - `AssistantOrder`
   - `ExecutionPlan`
   - `ApprovalRecord`
   - `ExecutionTask`
   - `AuditEvent`
   - `EvidencePack`
3. 核心 service / repository / adapter SPI contract。
4. northbound API contract 与 skill contract。
5. `AuthorizationService` 唯一权威授权出口。
6. `ExecuteAuthorizationService` 独立于 request 权限。
7. `AssetResolver.ResolveExact(...)` guardrail。
8. 审批与 execute 物理分离。
9. 统一错误码、状态枚举、`trace_id` 传递。

### 4.3 禁止事项

1. 不落真实 MySQL 执行。
2. 不实现多个 adapter 并行。
3. 不引入 fuzzy asset search。
4. 不让审批通过后自动执行。
5. 不把 request 权限和 execute 权限合并。
6. 不为了“先跑通”而跳过 `ExecutionPlan`。

### 4.4 产物

1. 可编译的 Go skeleton。
2. 领域对象、application contract、repository contract、adapter SPI。
3. 最小 northbound API skeleton。
4. `DBNativeAdapter` contract + stub。
5. guardrail tests。
6. `IMPLEMENTATION_STATUS.md` 更新。

### 4.5 验收标准

1. `go test ./...` 通过。
2. 代码中存在唯一 `AuthorizationService` 最终授权出口。
3. execute 路径存在独立的 execute authorization contract。
4. `AssetResolver` 只有 exact match contract，没有 fuzzy/best-effort 入口。
5. request / approval / execute 三条 northbound 入口已经分离。
6. northbound 返回对象能稳定携带 `trace_id`。

### 4.6 风险点

1. 只做骨架、没有共享持久化时，容易形成“接口正确、状态没接通”的假完成。
2. 如果错误码、状态机和领域对象没有一次固化，后续 Phase 2/3 会反复返工。
3. 当前仓库的 git/worktree 布局不标准，后续协作时容易误加内层 `.git` 目录。

### 4.7 进入下一阶段的门槛

1. 所有 guardrail 已落为测试，而不是只存在文档里。
2. 所有后续阶段需要的核心对象和接口已存在。
3. 团队确认以 Go 模块化单体继续推进，不再讨论换语言或先拆微服务。

### 4.8 当前仓库判断

当前仓库已基本具备本阶段的大部分产物，但仍需按本阶段验收标准做一次清点，不建议口头跳过。

---

## 5. Phase 2：最小控制链路可跑通

### 5.1 目标

让控制主链路在不依赖真实 MySQL 执行结果的前提下真正“跑通”，包括请求受理、授权决策、工单生成、审批流转、execute 门禁和任务启动骨架。

### 5.2 范围

1. `POST /api/v1/action-requests` 可真实落库或持久化到统一 repository。
2. `PrincipalResolver`、`AssetResolver`、`PolicyEngine`、`RiskEngine`、`AuthorizationService` 真正接线。
3. `AssistantOrder` 与 `ExecutionPlan` 共享持久化。
4. `ApprovalService` 真正驱动：
   - `WAITING_APPROVAL -> APPROVED`
   - `WAITING_APPROVAL -> REJECTED`
   - `WAITING_APPROVAL -> EXPIRED`
5. `POST /api/v1/orders/{order_id}/execute` 接入：
   - execute 身份校验
   - `ExecutePolicy`
   - `Plan Revalidate`
   - 任务启动骨架
6. append-only 审计仓储的最小实现。
7. 简版 `EvidencePack` 结构与查询接口。

### 5.3 禁止事项

1. 不接真实 MySQL 写操作。
2. 不扩散到 `mysql.user.create`、`mysql.user.grant` 等其他动作。
3. 不接入 MCP、CRD、gRPC、K8s 多通道。
4. 不把审批 TTL 写成代码里的隐式常量。
5. 不把 `PLAN_STALE` 当成普通 `FAILED`。

### 5.4 产物

1. 统一 repository 驱动的控制链路。
2. request/approval/execute/query API 可运行。
3. append-only `AuditEvent` 基础实现与 `AuditLedgerView` 聚合。
4. `PLAN_STALE`、`REJECTED`、`EXPIRED` 可见状态。
5. execute 幂等规则的最小实现。

### 5.5 验收标准

1. dev/test 场景能走到：
   - request
   - `APPROVED`
   - explicit execute
   - 创建任务骨架
2. prod 场景能走到：
   - request
   - `WAITING_APPROVAL`
   - 审批通过
   - `APPROVED`
   - explicit execute
3. `WAITING_APPROVAL` 状态调用 execute 返回 `APPROVAL_REQUIRED`。
4. `EXECUTING` 状态重复 execute 不产生第二个任务。
5. `Plan Revalidate` 能产出 `PLAN_STALE`，且此路径不创建任务。
6. 审批过期由 `ApprovalPolicy` TTL 或其最小实现驱动。
7. 成功、拒绝、过期、计划失效都能查到对应审计事件。

### 5.6 风险点

1. 若 repository 形状现在定义不稳，Phase 3 接真实执行时会再次重构 application service。
2. 若 Phase 2 只用内存对象“模拟通过”，团队会误判真实链路已经完成。
3. execute auth、approval auth、request auth 混用会直接破坏后续审计边界。

### 5.7 进入下一阶段的门槛

1. request / approval / execute 三条链路已经共享同一批 order/plan/task/audit repository。
2. `AuthorizationDecision` 与 `ExecuteAuthorizationDecision` 已成为各自路径的唯一权威结果。
3. `PLAN_STALE`、`EXPIRED`、`REJECTED` 都有真实状态推进与审计记录。
4. 不依赖真实数据库的最小控制链路已经能演示。

---

## 6. Phase 3：`mysql.database.create` MVP 纵切片

### 6.1 目标

在现有控制链路上接入真实 `DBNativeAdapter`，交付一个真正可执行、可审计、可留证据的 MVP 动作：`mysql.database.create`。

### 6.2 范围

1. `ExecutionRouter` 固定将 `mysql.database.create` 路由到 `db_native`。
2. `Plan Revalidate` 接入真实检查：
   - 目标 Asset 仍唯一命中
   - 连接引用仍可解析
   - 数据库仍不存在
   - 幂等键无运行中冲突
   - 计划版本仍匹配
3. `DBNativeAdapter` 最小执行流程：
   - `validate_target`
   - `check_database_not_exists`
   - `create_database`
   - `verify_database_created`
4. 执行结果统一映射成 `AdapterExecutionResult`。
5. 真实 `EvidencePack` 采集：
   - before/after snapshot
   - SQL 摘要
   - 失败详情
   - rollback suggestion
6. 动作参数只覆盖 `mysql.database.create` 所需最小集。

### 6.3 禁止事项

1. 不扩展到其他动作。
2. 不同时做多数据库引擎适配。
3. 不让 Agent 或 northbound handler 拼 SQL 并直接执行。
4. 不自动重试 `CREATE DATABASE` 这种写操作。
5. 不把真实凭证暴露给 Deep Agent。

### 6.4 产物

1. 可工作的 `DBNativeAdapter`。
2. `mysql.database.create` 真实纵切链路。
3. 幂等键检查。
4. 成功/失败/`PLAN_STALE` 三条路径的真实证据。
5. 面向 MVP 的最小集成测试和运行说明。

### 6.5 验收标准

1. 对受控 MySQL 目标能真实创建数据库。
2. 已存在数据库时能返回幂等成功或受控冲突结论，而不是重复副作用。
3. prod 路径仍必须先审批，审批通过后也必须走显式 execute。
4. execute 前数据库状态变化能触发 `PLAN_STALE`。
5. `AuditEvent` 与 `EvidencePack` 在成功、失败、`PLAN_STALE` 三条路径上都完整。
6. 返回给上层的是统一错误码，而不是裸驱动报错。

### 6.6 风险点

1. 本地或 CI 中可重复的 MySQL 测试环境准备成本较高。
2. `CREATE DATABASE` 的幂等语义如果设计不稳，会导致“已存在”和“重复执行”混淆。
3. 连接引用、密钥管理、执行凭证权限需要与平台安全边界一致，否则会绕过 Control Layer 初衷。

### 6.7 进入下一阶段的门槛

1. `mysql.database.create` 已是一个真实可演示动作，而不是 stub。
2. prod 审批路径、dev/test 无审批路径都已验证。
3. 审计与证据在真实执行路径中得到验证。
4. 失败和 `PLAN_STALE` 路径都不是靠注释描述，而是有实际测试或演示证据。

---

## 7. Phase 4：Deep Agent 最小接入

### 7.1 目标

让 Deep Agent 通过 Skill/HTTP 安全接入已经成形的 Control Layer，但不把任何控制权下放给 Agent。

### 7.2 范围

1. 固化两个最小 skill：
   - `request_mysql_database_create`
   - `execute_assistant_order`
2. Deep Agent 到 Control API 的输入输出映射。
3. Agent 侧最小编排规则：
   - `approval_required=false` 且 order 状态为 `APPROVED` 时，允许自动串联第二次 execute 调用
   - `approval_required=true` 时只返回工单等待审批
4. auth context 与 `principal_id` 的可信传递。
5. user-facing message 模板与错误解释。

### 7.3 禁止事项

1. 不让 Deep Agent 直接调用 Adapter。
2. 不让 Deep Agent 直接接触凭证引用。
3. 不把 Policy / Risk / Approval 逻辑搬到 Agent 侧。
4. 不在 approval 通过后由 Agent 默认自动执行。
5. 不把自然语言 alias 直接送入核心 `AssetResolver`。

### 7.4 产物

1. Skill contract 与 HTTP 映射实现。
2. Deep Agent 最小调用示例。
3. request/execute 两次独立调用的演示链路。
4. 常见失败路径的用户消息模板。

### 7.5 验收标准

1. Agent 能成功提交 `request_mysql_database_create`。
2. dev/test 无审批场景下，Agent 可以自动串联 execute，但底层仍体现为两次受控调用。
3. prod 场景下，Agent 只能返回等待审批，不会偷跑 execute。
4. `order_id`、`task_id`、`trace_id` 能被上层会话安全引用。
5. Agent 接入没有改变任何 Control Layer 授权、审批、审计语义。

### 7.6 风险点

1. Agent 自动串联 execute 可能与重试机制叠加，造成重复触发风险。
2. auth context 若在 skill 层处理不严，会出现“会话身份”和“执行身份”不一致。
3. 如果用户消息模板过度简化，可能掩盖 `APPROVAL_REQUIRED`、`PLAN_STALE` 等重要状态。

### 7.7 进入下一阶段的门槛

1. Deep Agent 已经是 northbound client，而不是事实上的控制器。
2. request/execute 分离语义在 Agent 接入后仍保持清晰。
3. 自动串联 execute 只发生在正式允许的低风险路径。

---

## 8. Phase 5：MCP 兼容接入

### 8.1 目标

增加 `MCPToolAdapter` 作为第二条 southbound 路线，验证本项目的 Control Layer 能对不同执行通道保持同一套权限、审批、审计、证据语义。

### 8.2 范围

1. 实现 `MCPToolAdapter` 的最小 SPI。
2. 实现 capability check / route selection / result normalization。
3. 在不改变现有 northbound 入口的前提下，把至少一个已存在动作接到 MCP 路线。
4. `AuditEvent.selected_adapter`、`EvidencePack`、错误码映射对 MCP 路线保持一致。
5. 保留 `DBNativeAdapter` 作为基线对照通道。

### 8.3 禁止事项

1. 不把 MCP 当成 Control Layer 替代品。
2. 不让 Deep Agent 直接绕过 Control Layer 调 MCP tool。
3. 不为了迁就 MCP tool 而修改核心状态机或审批语义。
4. 不在 Phase 5 顺手扩散到多动作、多 server、大量工具接入。

### 8.4 产物

1. `MCPToolAdapter` 最小实现。
2. route 配置与 capability 探测。
3. 同一动作的多通道路由能力或等价兼容验证。
4. MCP 兼容测试与使用说明。

### 8.5 验收标准

1. 同一个 northbound 请求仍然先经过 Control Layer，再路由到 MCP。
2. MCP 路线同样要求：
   - `AuthorizationDecision`
   - 审批
   - explicit execute
   - 审计
   - 证据
3. adapter 切换后，上层 skill / API 契约不变。
4. 路由结果、适配器类型、失败原因可在审计中看见。
5. MCP 路线不会削弱现有 `DBNativeAdapter` 已验证的控制门禁。

### 8.6 风险点

1. 企业内现成 MCP tool 的输入输出契约可能不稳定。
2. MCP tool 返回结果的可审计性和证据密度未必与 DB-native 一致。
3. 若 route 规则设计不好，可能在不同 adapter 之间引入不可解释的行为差异。

### 8.7 进入下一阶段的门槛

1. 至少一个 MCP 路线经过完整 control chain 验证。
2. DB-native 与 MCP 两条路径在核心治理语义上保持一致。
3. 团队确认 MCP 接入增加的是执行通道，而不是新的控制面。

---

## 9. Git 工作流建议

### 9.1 总体原则

1. 一阶段一分支，不要用单个超长生命周期分支承载全部 Phase。
2. 每阶段分支从当时的 `main` 或上一阶段合并后的主干拉出。
3. commit message 使用 Conventional Commits + bounded context scope。
4. 每阶段 PR 前都更新自检文档，不允许只贴口头说明。

### 9.2 每阶段推荐 branch 命名

| Phase | 推荐分支名 |
|---|---|
| Phase 1 | `phase/01-skeleton-guardrails` |
| Phase 2 | `phase/02-minimal-control-chain` |
| Phase 3 | `phase/03-mysql-database-create-mvp` |
| Phase 4 | `phase/04-deep-agent-minimal` |
| Phase 5 | `phase/05-mcp-compat` |

### 9.3 每阶段推荐 commit message 风格

| Phase | 推荐风格 |
|---|---|
| Phase 1 | `feat(domain): ...` / `feat(api): ...` / `test(guardrail): ...` / `docs(status): ...` |
| Phase 2 | `feat(control): ...` / `feat(approval): ...` / `feat(audit): ...` |
| Phase 3 | `feat(mysql): ...` / `feat(adapter): ...` / `test(mysql): ...` |
| Phase 4 | `feat(skill): ...` / `feat(agent): ...` / `docs(agent): ...` |
| Phase 5 | `feat(mcp): ...` / `test(mcp): ...` / `refactor(router): ...` |

建议：

1. 一个 commit 只解决一个边界问题，不要把 policy、approval、adapter、test 混成单次提交。
2. Phase 2 以后所有 commit 都至少配一个回归测试或集成测试。
3. 文档更新单独提交，避免和执行逻辑大块混杂。

### 9.4 PR / merge 前是否建议做自检文档

建议：**必须做，而且应该轻量但固定。**

推荐最小做法：

1. 更新根目录 `IMPLEMENTATION_STATUS.md`。
2. 为当前阶段追加一份短自检记录，建议命名：
   - `design_docs/status/phase-0X-gate-check.md`
3. 自检记录至少回答：
   - 本阶段目标是否达成
   - 哪些验收标准已验证
   - 哪些禁止事项没有触碰
   - 哪些风险仍未关闭

如果团队不想新增过多文件，最低要求也应把上述四项并入 `IMPLEMENTATION_STATUS.md`。

---

## 10. 当前最合理的近期工作顺序

### 10.1 先做什么

1. 先把当前仓库按 `Phase 1` 验收标准清点一遍，确认哪些 guardrail 已经真正由测试锁住，哪些还只是 stub。
2. 立刻进入 `Phase 2`，优先接通统一 repository、审批共享状态、execute authorization、plan revalidate、append-only audit repository。
3. 在 `Phase 2` 可演示后，直接切 `Phase 3`，只做 `mysql.database.create` 一个真实纵切。

### 10.2 后做什么

1. `mysql.database.create` MVP 稳定后，再做 `Phase 4` 的 Deep Agent 最小接入。
2. Deep Agent 接入稳定后，再做 `Phase 5` 的 MCP 兼容路径。
3. 只有在 Phase 5 完成后，才考虑扩展到其他动作，如：
   - `mysql.user.create`
   - `mysql.user.grant`
   - `mysql.password.change`

### 10.3 哪些事情现在不要做

1. 不要先做 UI。
2. 不要先做 fuzzy asset search。
3. 不要先做微服务拆分。
4. 不要先做多 adapter 并行实现。
5. 不要先扩展多个 MySQL 动作。
6. 不要把审批、execute、adapter 逻辑塞回一个 handler。
7. 不要把当前 Go skeleton 推倒重来换成其他语言。

---

## 11. 已发现的文档漂移与处理建议

这些问题属于**文档漂移**，目前不足以阻塞本路线图，但应在后续文档治理时统一处理：

1. `design_docs/09-codex-phased-execution-manual-v0.1.md` 中的示例路径仍写成 `docs/...`，而当前仓库正式文档实际位于 `design_docs/...`。
2. `design_docs/03-assistant-spec-v0.7.md` 第 17 节的“后续文档拆分建议”仍保留较早版本的文档命名，如 `05-action-dictionary-v0.1.md`，与当前仓库实际文档集合不完全一致。
3. 审批 TTL 命名存在轻微漂移：
   - `03` / `04` 更偏向 `approval_ttl`
   - `06` 更偏向 `approval_ttl_seconds`
   推荐在代码里统一成一个持久化字段名，并在接口层做兼容解释。
4. 当前仓库已存在 `design_docs/11-claude-code-alignment-review-v0.1.md`，本次新增 `design_docs/11-execution-roadmap-v0.1.md` 会形成同号前缀并列。它是命名碰撞，不是领域语义冲突。

结论：

> 当前没有发现足以改变主架构原则的硬冲突；发现的是路径、命名和编号层面的漂移，需要在后续文档治理中统一。

