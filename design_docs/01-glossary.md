# 术语表（Glossary）

> 本文档用于保证人类读者与 coding agent 对关键术语的理解一致。

## 1. Assistant

本项目中的 Assistant，指一个面向 DBA / 平台工程师的企业级智能助手。

它不仅负责回答问题，还负责：

- 识别运维动作意图
- 校验权限与风险
- 发起审批
- 触发受控执行
- 记录审计和证据

## 2. Deep Agent

项目中选用的上层智能体运行时。

职责：

- 自然语言理解
- 多步计划
- 参数补全
- 调用高层 Skill
- 用自然语言回复用户

不负责：

- 最终权限裁定
- 最终执行签发
- 高风险绕过审批

## 3. Assistant Control Layer

位于 Deep Agent 和底层执行器之间的企业控制后端。

职责：

- 动作规范化
- 资产解析
- 权限判断
- 风险判断
- 审批编排
- 执行路由
- 审计记账
- 证据归档

一句话理解：

> 它是整个 Assistant 的“手刹、变速箱、仪表盘和黑匣子”。

## 4. Action

系统内部的标准动作名。

示例：

- `mysql.database.create`
- `mysql.user.create`
- `mysql.user.grant`
- `mysql.backup.create`
- `resource.cluster.register`

Action 是系统的**稳定接口语义**。

它不直接等于：

- 某个 HTTP path
- 某个 SQL 语句
- 某个 shell 脚本名
- 某个 MCP tool 名称

## 5. Skill

供 Deep Agent 调用的高层业务能力。

建议 Deep Agent 看到的是高层 Skill，而不是底层 Adapter。

示例：

- `request_mysql_database_create`
- `request_mysql_user_create`
- `request_mysql_backup`

Skill 的作用通常是：

> 帮 Deep Agent 以统一方式向 Control Layer 提交 ActionRequest。

## 6. Adapter

执行适配器。用于将统一 Action 真正落地到某个具体执行通道。

示例：

- `MCPToolAdapter`
- `DBNativeAdapter`
- `CRDAdapter`
- `GrpcCallAdapter`
- `K8sAdapter`
- `ShellAnsibleAdapter`
- `VMSSHAdapter`

Adapter 负责：

- 接收标准执行计划
- 调用底层目标系统
- 返回执行结果

Adapter 不负责：

- 理解自然语言
- 最终权限裁定
- 风险判定

## 7. Asset（资产）

系统中被纳管的对象。

示例：

- 项目（Project）
- 集群（Cluster）
- 节点组（NodeGroup）
- 节点（Node）
- 存储类（StorageClass）
- 服务组（ServiceGroup）
- 服务实例（ServiceInstance）
- 单元（Unit）

Asset 是执行动作时的目标对象基础。

## 8. Asset Resolver

资产解析器。

作用：

- 把用户自然语言中的目标对象，映射成系统内部的受控资产 ID。

例如：

- “A 项目的生产订单库”
- “华东 prod 主库”
- “上周创建的备份任务”

都要先被解析成明确的受控对象。

## 9. Principal

本次请求的主体身份。

表示“是谁在发起这个请求”。

Principal 通常包含：

- 用户 ID
- 显示名
- 角色集合
- 所属组
- 资源范围
- 审批角色

没有 Principal，就无法做权限判断。

## 10. Role

角色。

示例：

- `dba`
- `mysql_operator`
- `backup_operator`
- `prod_approver`
- `platform_admin`
- `readonly_auditor`

Role 用来表达“这个主体大概可以做哪类事情”。

## 11. Scope

资源范围。

用于表达：

- 哪些项目可操作
- 哪些环境可操作
- 哪些集群可操作
- 哪些服务组可操作

## 12. RBAC

Role-Based Access Control，基于角色的访问控制。

在本项目中，RBAC 主要用于：

- 判断某角色能否发起某类 Action

## 13. ABAC

Attribute-Based Access Control，基于属性的访问控制。

在本项目中，ABAC 主要用于：

- 根据环境、项目、目标实例、风险等级等属性做更精细的控制

## 14. ActionPolicy

动作策略。

定义：

- 哪些角色可以发起某个 Action
- 哪些角色可以执行某个 Action

## 15. ResourceScopePolicy

资源范围策略。

定义：

- 某个 Principal 或 Role 可以对哪些资源范围内的对象执行动作

## 16. Risk Level

风险等级。

建议最少分为：

- `R0`：只读
- `R1`：低风险写操作
- `R2`：中风险操作
- `R3`：高风险操作

## 17. Approval

审批。

当动作风险较高或命中敏感环境/资源时，系统不应直接执行，而应进入审批流程。

## 18. AssistantOrder

Assistant Control Layer 中的正式工单对象。

它不是自然语言本身，而是经过：

- 动作标准化
- 资产解析
- 风险判断

后的正式控制流对象。

## 19. ExecutionPlan

执行计划。

定义某个 Action 最终应该如何执行。

它通常包含：

- 选择哪个 Adapter
- 需要哪些步骤
- 是否需要回滚策略
- 是否需要幂等控制

## 20. ExecutionTask

执行任务。

表示某个工单/动作在运行时的实际执行对象。

## 21. ExecutionStep

执行步骤。

用于表示一个 ExecutionTask 被拆分后的具体步骤。

例如：

- Step 1：校验目标实例存在
- Step 2：获取数据库连接
- Step 3：执行 CREATE DATABASE
- Step 4：验证数据库创建成功

## 22. Audit Ledger

审计账本。

记录从用户原始请求到最终执行结果的全过程。

它比普通操作日志更完整。

## 23. Evidence Pack

证据包。

用于归档一次动作的关键证据，例如：

- 执行前状态
- 执行参数
- 执行结果
- 日志摘要
- 审批记录
- 回滚建议

## 24. MCP Server

Model Context Protocol Server。

它可以向 Agent 暴露标准化的：

- tools
- resources
- prompts

在本项目里，MCP Server 是**一种可接入的执行通道**，不是整个企业控制层的替代品。

## 25. Reference Platform（参考平台）

一个历史存在的企业数据库平台，其设计资料和逆向分析结果用于辅助本项目提炼：

- 资产建模
- 工单/任务语义
- 控制面/执行面分层
- 审计与适配思路

本项目不直接依赖该平台运行时。
