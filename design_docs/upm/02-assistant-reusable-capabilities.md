# 面向企业级 DBA Assistant 的可复用能力盘点

## 0. 前提

本文件的立场不是“如何复用 upm-api-server 代码”，而是“从 upm-api-server 中提炼什么控制面设计，供未来 `Deep Agent + Assistant Control Layer + MCP/受控适配器` 复用”。

结论口径：

- `可直接借鉴`：优先复用对象边界、状态语义、控制流抽象。
- `可参考但不宜直接复用`：思想可借，但代码结构或耦合方式不适合 assistant。
- `必须新增建设`：当前代码没有、或对 AI 场景明显不够。

## 1. 能直接借鉴的能力

### 1.1 资源注册与资产目录语义

#### 代码事实

- `ClusterServiceImpl.save(...)`  
  文件：`upm-resource/upm-resource-ms/src/main/java/io/syntropycloud/upm/resource/ms/service/impl/ClusterServiceImpl.java`
- `StorageClassServiceImpl.save(...)`  
  文件：`upm-resource/upm-resource-ms/src/main/java/io/syntropycloud/upm/resource/ms/service/impl/StorageClassServiceImpl.java`
- `NodeGroupServiceImpl.save(...)`  
  文件：`upm-resource/upm-resource-ms/src/main/java/io/syntropycloud/upm/resource/ms/service/impl/NodeGroupServiceImpl.java`
- `NodeServiceImpl.save(...)`  
  文件：`upm-resource/upm-resource-ms/src/main/java/io/syntropycloud/upm/resource/ms/service/impl/NodeServiceImpl.java`
- `StorageS3ServiceImpl.save(...)`  
  文件：`upm-resource/upm-resource-ms/src/main/java/io/syntropycloud/upm/resource/ms/service/impl/StorageS3ServiceImpl.java`

#### 可借鉴点

- 先注册资产，再引用资产。
- 内部对象持有：
  - 内部 ID
  - 展示名
  - `relateName` 或外部真实名
  - enabled 状态
  - creator / created time
- 这是 assistant 的资产解析层 blueprint。

#### 建议

- 未来 Control Layer 必须保留这类资产注册中心。
- Deep Agent 只能消费解析后的受控资产，不能直接面向裸 IP / 裸 kubeconfig / 裸账号执行。

### 1.2 服务聚合模型：`ServGroup -> Serv -> Unit`

#### 代码事实

- `ServGroupDO`  
  文件：`upm-service/upm-service-common/upm-service-common-core/src/main/java/io/syntropycloud/upm/service/common/core/domain/ServGroupDO.java`
- `ServDO`  
  文件：`upm-service/upm-service-common/upm-service-common-core/src/main/java/io/syntropycloud/upm/service/common/core/domain/ServDO.java`
- `UnitDO`  
  文件：`upm-service/upm-service-common/upm-service-common-core/src/main/java/io/syntropycloud/upm/service/common/core/domain/UnitDO.java`

#### 可借鉴点

- 一个逻辑服务组可以包含多个服务角色，例如 MySQL + ProxySQL。
- 一个服务角色再包含多个 unit。
- 这是比“一个集群一个对象”更适合 DBA assistant 的粒度。

#### 对 future assistant 的价值

- 很适合承载：
  - 服务创建/扩容/缩容
  - 组件级版本升级
  - unit 级恢复/重建
  - topology 感知操作

### 1.3 工单模型：`OrderGroup -> Order`

#### 代码事实

- `OrderGroupDO`
- `OrderDO`
- `OrderGroupCoreService.save/approve/execute(...)`
- `OrderGroupCoreCheck.checkSave/checkApprove/checkExecute(...)`

文件：

- `upm-service/upm-service-common/upm-service-common-core/src/main/java/io/syntropycloud/upm/service/common/core/domain/OrderGroupDO.java`
- `upm-service/upm-service-common/upm-service-common-core/src/main/java/io/syntropycloud/upm/service/common/core/domain/OrderDO.java`
- `upm-service/upm-service-common/upm-service-common-core/src/main/java/io/syntropycloud/upm/service/common/core/service/OrderGroupCoreService.java`
- `upm-service/upm-service-common/upm-service-common-core/src/main/java/io/syntropycloud/upm/service/common/core/service/check/OrderGroupCoreCheck.java`

#### 可借鉴点

- 目标状态和当前状态被清楚分开。
- `OrderDO` 里有大量 `pre*` 字段，天然适合作为变更 diff。
- 先固化变更，再执行。

#### 对 assistant 的价值

- 非常适合作为未来 `ActionRequest` / `PlannedChange` / `ApprovalSubject` 的原型。

### 1.4 执行模型：`Task -> Subtask`

#### 代码事实

- `TaskDO`
- `SubtaskDO`
- `TaskSchedulerCoreService.buildTask/buildSubTask/saveTaskAndExecutor(...)`

文件：

- `upm-service/upm-service-common/upm-service-common-core/src/main/java/io/syntropycloud/upm/service/common/core/domain/TaskDO.java`
- `upm-service/upm-service-common/upm-service-common-core/src/main/java/io/syntropycloud/upm/service/common/core/domain/SubtaskDO.java`
- `upm-service/upm-service-common/upm-service-common-core/src/main/java/io/syntropycloud/upm/service/common/core/service/TaskSchedulerCoreService.java`

#### 可借鉴点

- 子任务带 `priority`，相同优先级可并发执行。
- 子任务带 `timeout`、状态、错误信息。
- 任务心跳写入 Redis。

#### 对 assistant 的价值

- 几乎可以直接翻译成未来的 execution plan runtime。
- 特别适合多适配器并排执行和有序依赖执行。

### 1.5 K8s/CRD/GrpcCall 的受控适配模式

#### 代码事实

- `KubeResourceCoreService.buildGrpcCall/createGrpcCall/pollingSubTaskGrpcCallStatus(...)`
- `SubtaskExecuteService.backup/restore/cloneSourceData/gtidPurged(...)`
- `NodeServiceImpl.save(...)` / `StorageClassServiceImpl.save(...)` 直接操作 K8s API

#### 可借鉴点

- 控制面动作不直接嵌死实现，而是统一转到适配层。
- `GrpcCall` 本质上已经是一个受控执行适配器。

#### 对 assistant 的价值

- 未来完全可以抽象成统一 adapter 契约：
  - `K8s adapter`
  - `CRD adapter`
  - `GrpcCall adapter`
  - `DB-native adapter`
  - `MCP tool adapter`

### 1.6 审计字段设计

#### 代码事实

- `OperateLogDO` 字段已经覆盖：
  - operator
  - objType
  - action
  - objName
  - description
  - body
  - success
  - errMsg
  - traceId / spanId

文件：

- `upm-user/upm-user-ms/src/main/java/io/syntropycloud/upm/user/ms/domain/OperateLogDO.java`

#### 可借鉴点

- 字段粒度足够作为基础审计 event schema。
- 未来可以直接扩展，而不是从零设计。

### 1.7 输入校验模式

#### 代码事实

- 各模块普遍先走 `*Check`，返回 `ChkRs`
- 例如：
  - `MysqlUserCheck`
  - `MysqlDatabaseCheck`
  - `ClusterCheck`
  - `OrderGroupCheck`

#### 可借鉴点

- 把“是否合法”与“如何执行”拆开。
- 很适合作为 future Control Layer 的：
  - policy check
  - preflight check
  - safety gate

## 2. 能参考但不宜直接复用的能力

### 2.1 URL/菜单型权限模型

#### 代码事实

- `UserServiceImpl.checkApp(...)` 最终依据 `AppDO.urls + method` 做权限判断。
- `AppDO` 同时承担菜单节点和 API 权限节点。

文件：

- `upm-user/upm-user-ms/src/main/java/io/syntropycloud/upm/user/ms/domain/AppDO.java`
- `upm-user/upm-user-ms/src/main/java/io/syntropycloud/upm/user/ms/service/impl/UserServiceImpl.java`

#### 不宜直接复用的原因

- 这更适合平台前后端，不适合 agent。
- assistant 需要的是：
  - action 权限
  - 资源权限
  - 风险等级权限
  - 是否允许直执 / 是否必须审批

### 2.2 基于 Spring controller 的 API 组织方式

#### 代码事实

- 现有接口是典型面向前端页面的 REST 分组。
- 例如 MySQL 扩缩容拆成 `computer_scaling` / `storage_expansion` / `unit_expansion`。

#### 不宜直接复用的原因

- assistant 更适合统一 action 字典，而不是直接沿用 UI API path。
- 现有 API 的颗粒度有时偏页面操作，不够 agent-native。

### 2.3 Java 服务间的强耦合 openfeign 方式

#### 代码事实

- `BaseCoreService` 通过 `AuthClient/UserClient/ProjectClient/ClusterClient/...` 聚合所有外部依赖。

文件：

- `upm-service/upm-service-common/upm-service-common-core/src/main/java/io/syntropycloud/upm/service/common/core/service/BaseCoreService.java`

#### 不宜直接复用的原因

- 未来 assistant 不能建立在“多个 Java 微服务必须同时在线”的前提上。
- 这些依赖更适合转译成：
  - 资产目录 API
  - policy API
  - execution API
  - audit API

### 2.4 审批实现的轻量状态化

#### 代码事实

- `OrderGroupCoreService.approve(...)` 只是更新 `status` 与 `msg`。
- 未发现完整多级审批流引擎。

#### 不宜直接复用的原因

- 对企业级 DBA assistant 来说，审批通常是核心能力。
- 当前实现更像“审批结果回写”，而不是审批系统本身。

### 2.5 `Software` 依赖 PodTemplate 的实现方式

#### 代码事实

- `SoftwareServiceImpl` 直接从 K8s `PodTemplate` 暴露软件对象。

文件：

- `upm-resource/upm-resource-ms/src/main/java/io/syntropycloud/upm/resource/ms/service/impl/SoftwareServiceImpl.java`

#### 不宜直接复用的原因

- 未来 assistant 的软件目录可能来自：
  - CMDB
  - 镜像仓库
  - 发布目录
  - 制品库
  - Operator capability registry
- 不应把“软件版本目录”绑定死在 K8s PodTemplate 上。

### 2.6 请求签名和网关语义

#### 代码事实

- 当前系统高度依赖：
  - `x-upm-signature`
  - `x-upm-timestamp`
  - gateway filter

#### 不宜直接复用的原因

- future assistant 内部组件之间未必通过 HTTP gateway 协调。
- 其思想可以保留，但实现需要转成：
  - 内部 identity
  - action signature
  - execution authorization token

## 3. 必须新增建设的能力

以下能力当前仓库没有完整解决，未来 assistant 必须自建。

### 3.1 Assistant Control Layer

必须存在一个独立控制层，负责：

- 接收 agent 输出的意图
- 归一化动作
- 解析资产
- 做风控、审批、审计
- 规划执行路线
- 调度 MCP 或自研 adapter

没有这一层，Deep Agent 只能直接调工具，无法形成企业级控制闭环。

### 3.2 意图规范化

当前仓库没有“自然语言意图 -> 标准动作”的层。

未来必须有：

- intent parser
- action normalizer
- 参数补全与歧义消解
- 资源指代解析

### 3.3 统一动作字典

当前系统动作存在于：

- controller path
- `ActionCoreConsts`
- `TaskActionConsts`
- `MysqlTaskActionConsts`

但并没有一个对 assistant 友好的统一动作词典。

未来必须沉淀：

- 标准动作名
- 参数 schema
- 风险等级
- 默认执行器
- 是否必须审批
- 是否必须证据包

### 3.4 资产解析层

当前代码有资产目录，但没有 agent 友好的 asset resolution。

未来必须支持：

- “prod 华东主库集群”
- “A 项目的 mysql 订单库”
- “带代理的 MySQL 服务组”
- “最近一次全量备份”

解析后再映射到受控 ID。

### 3.5 风险分级

当前代码中风控主要体现为 `check*` 与“是否允许操作”，缺少统一风险层。

未来必须新增：

- `R0` 只读
- `R1` 低风险写
- `R2` 中风险
- `R3` 高风险

并与审批、执行方式绑定。

### 3.6 审批流

当前只有 `approve(status,msg)`。

未来必须补：

- 审批策略
- 多级审批
- 风险驱动审批
- 紧急变更模式
- agent 建议与人工确认分离

### 3.7 审计账本

现有 `OperateLog` 不是完整账本。

未来必须补：

- assistant 原始意图
- 标准动作
- 风险判定
- 审批决策
- 计划版本
- 执行路线
- 结果摘要
- 证据引用

### 3.8 证据包

当前系统有 `MysqlBackupFileDO` 这类产物元数据，但没有统一 evidence package。

未来应统一沉淀：

- SQL / CRD / gRPC 请求摘要
- 前后状态快照
- 运行日志
- 结果对象
- 人工审批记录
- 关联 task/subtask

### 3.9 执行路由

当前代码里“怎么执行”隐含在 Java service 里。

未来必须显式化为：

- `route(action, asset, risk, environment) -> adapter`

例如：

- `mysql.database.create -> DB-native`
- `mysql.service.create -> CRD/K8s`
- `mysql.backup.create -> GrpcCall + S3`
- `resource.cluster.register -> K8s adapter`

### 3.10 MCP 兼容层

当前系统没有 MCP 概念。

未来需要一层统一抽象：

- 若目标能力可由 MCP tool 表达，则走 MCP
- 若需更强受控执行，则走自研 adapter
- 两者都由同一个 action dictionary 驱动

### 3.11 失败补偿与回滚建议

当前只有局部补偿，如用户创建失败后删用户。

未来必须补：

- 失败分类
- 自动补偿
- 回滚建议
- 人工接管建议
- 二次执行建议

## 4. 对未来架构的建议

## 4.1 哪些能力适合由 Deep Agent 承担

- 用户意图理解
- 参数补全与歧义澄清
- 变更方案草拟
- 风险说明生成
- 证据包阅读与总结
- 执行后报告生成

Deep Agent 应该负责“认知与规划”，而不是直接握有执行权。

## 4.2 哪些能力绝不能交给 Deep Agent 直接承担

- 最终权限判定
- 资产 ID 解析的最终落锤
- 风险等级裁定
- 审批绕过判断
- adapter 直接凭证管理
- 审计账本写入
- 执行动作的最终签发

这些都必须沉到 Control Layer。

## 4.3 哪些能力必须沉在 Control Layer

- action normalization
- policy enforcement
- risk gating
- approval orchestration
- execution routing
- idempotency / dedupe
- task/subtask runtime
- audit/evidence ledger

这是未来架构中最关键的一层。

## 4.4 哪些能力应该做成 adapter

- `K8s adapter`
- `CRD adapter`
- `GrpcCall adapter`
- `DB-native adapter`
- `Shell/Ansible adapter`
- `MCP tool adapter`
- `VM/SSH adapter`

adapter 的职责是受控执行，不负责意图理解。

## 4.5 建议的目标架构

```text
User
  -> Deep Agent
     -> Assistant Control Layer
        -> Asset Resolver
        -> Policy / Risk / Approval
        -> Action Planner
        -> Task / Evidence Ledger
        -> Execution Router
           -> MCP Tool Adapter
           -> CRD Adapter
           -> GrpcCall Adapter
           -> DB-native Adapter
           -> K8s Adapter
           -> Shell/Ansible Adapter
```

## 5. 结论表

| 分类 | 内容 | 代表依据 |
| --- | --- | --- |
| 能直接借鉴 | 资产注册模型 | `ClusterServiceImpl`、`StorageClassServiceImpl`、`NodeServiceImpl` |
| 能直接借鉴 | `ServGroup -> Serv -> Unit` 聚合 | `ServGroupDO`、`ServDO`、`UnitDO` |
| 能直接借鉴 | `OrderGroup -> Order` 变更模型 | `OrderGroupDO`、`OrderDO`、`OrderGroupCoreService` |
| 能直接借鉴 | `Task -> Subtask` 执行模型 | `TaskDO`、`SubtaskDO`、`TaskSchedulerCoreService` |
| 能直接借鉴 | `K8s / GrpcCall / DB-native` 适配思想 | `KubeResourceCoreService`、`SubtaskExecuteService`、`MysqlDatabaseServiceImpl` |
| 可参考但需重构 | URL/菜单权限模型 | `AppDO`、`UserServiceImpl.checkApp(...)` |
| 可参考但需重构 | REST path 级动作组织 | 各 `*Controller` |
| 可参考但需重构 | Java 微服务强耦合调用 | `BaseCoreService` 的 openfeign 依赖 |
| 可参考但需重构 | 轻量审批状态位 | `OrderGroupCoreService.approve(...)` |
| 可参考但需重构 | PodTemplate 驱动的软件目录 | `SoftwareServiceImpl` |
| 必须自建 | Assistant Control Layer | 当前仓库缺失 |
| 必须自建 | 意图规范化与动作字典 | 当前仓库缺失 |
| 必须自建 | 风险分级与审批策略 | 当前仓库仅有校验，不成体系 |
| 必须自建 | 证据包与审计账本 | 当前仓库只有操作日志和部分备份产物 |
| 必须自建 | MCP 兼容层 | 当前仓库缺失 |
| 必须自建 | 失败补偿/回滚建议 | 当前仓库只有零散补偿逻辑 |

## 6. 最终判断

`upm-api-server` 最值得未来 assistant 复用的是“控制面结构化思维”，不是 Java 运行时。

可直接拿走的是：

- 资源对象边界
- 服务聚合边界
- 工单/任务/子任务状态模型
- 审计基础字段
- 适配器分层思想

必须新建的是：

- agent 友好的动作体系
- control layer
- MCP/adapter 路由
- 风控审批账本

如果这几层补齐，future DBA Assistant 就能既继承 UPM 的控制面语义，又摆脱对 Java platform 的直接依赖。
