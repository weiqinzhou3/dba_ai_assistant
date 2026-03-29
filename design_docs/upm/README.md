# Assistant Analysis 总览

本目录的目标不是解释 `upm-api-server` 怎么用，而是回答一个更直接的问题：

> 如果未来要基于 Deep Agent 设计一个企业级 DBA Assistant，并且不直接依赖 Java platform，那么 `upm-api-server` 中有哪些控制面设计值得借鉴？哪些能力必须重建？哪些动作适合抽象为统一 Action？哪些地方适合未来兼容 MCP？

结论先行：`upm-api-server` 最有价值的不是 Spring Boot 代码，而是它已经把企业级控制面拆出了几层清晰语义：

- 资产目录
- 服务聚合
- 工单变更
- 异步任务
- 子任务编排
- 适配器执行
- 基础审计

未来 assistant 不应把这些 Java 模块当运行时依赖，但非常值得把它们提炼为新的 `Assistant Control Layer blueprint`。

## 1. upm-api-server 最值得借鉴的 5 个设计

### 1.1 资产先注册、动作再执行

`Cluster / Zone / NodeGroup / Node / StorageClass / StorageS3` 都不是运行时临时拼出来的，而是先注册成内部对象，再被服务编排引用。

关键依据：

- `ClusterServiceImpl.save(...)`
- `NodeGroupServiceImpl.save(...)`
- `NodeServiceImpl.save(...)`
- `StorageClassServiceImpl.save(...)`
- `StorageS3ServiceImpl.save(...)`

这说明未来 assistant 也必须先有资产解析层，不能让 agent 直接拿裸集群、裸 IP、裸账号执行。

### 1.2 `ServGroup -> Serv -> Unit` 的服务聚合模型

`ServGroupDO`、`ServDO`、`UnitDO` 明确区分了：

- 逻辑服务组
- 服务角色
- 实际实例

这比“一个集群一个对象”或“一个 deployment 一个对象”更适合 DBA assistant，因为数据库操作经常需要在 `service` 级和 `unit` 级来回切换。

### 1.3 `OrderGroup -> Order` 的变更固化模型

当前系统把“我想做什么”先固化成工单对象，再进入执行阶段。

关键依据：

- `OrderGroupDO`
- `OrderDO`
- `OrderGroupCoreService.save/approve/execute(...)`

这给 future assistant 一个很重要的启发：  
 高风险动作不应由 Deep Agent 直接触发执行，而应先固化为标准变更对象。

### 1.4 `Task -> Subtask` 的执行计划模型

`TaskSchedulerCoreService` 已经把复杂执行拆成：

- 任务
- 子任务
- 优先级
- 并发执行
- 超时
- 状态跟踪

这是 future Control Layer 最容易直接继承的一层。

### 1.5 “控制面对象”与“执行面对象”分离

当前系统的服务对象不是直接等于 K8s 对象。

控制面对象：

- `ServGroup`
- `Serv`
- `Unit`
- `OrderGroup`
- `Task`

执行面对象：

- `UnitSet`
- `GrpcCall`
- `ConfigMap`
- `Pod`
- MySQL schema/user

这意味着未来 assistant 完全可以保留控制面模型，同时自由更换执行器：

- MCP
- CRD
- gRPC
- DB-native
- shell/ansible/VM

## 2. 哪些能力不能直接复用

### 2.1 URL/菜单型权限模型

当前权限核心是 `Role -> App(url, method)`，本质是平台前后端权限，不是 assistant 权限。

future assistant 需要的是：

- action 权限
- resource 权限
- risk 权限
- approval 权限

### 2.2 Spring controller 风格的动作组织

比如 MySQL 扩缩容被拆成：

- `computer_scaling`
- `storage_expansion`
- `unit_expansion`

这对页面友好，但对 agent 不够统一。future assistant 更适合一个标准动作 `mysql.service.scale`，再用参数区分维度。

### 2.3 Java 微服务强耦合

`BaseCoreService` 里通过 openfeign 聚合了大量平台内部依赖。未来 assistant 不能建立在“这些 Java 服务必须原样存在”的前提上。

### 2.4 当前审批不是完整审批系统

当前仓库里能看到审批状态位，但 `OrderGroupCoreService.approve(...)` 本质只是改状态。没有发现完整审批编排、审批策略引擎、多级审批链。

因此可以明确判断：

> 审批可能主要在 platform/UI/workflow 层，而非当前代码完整承载。

### 2.5 软件目录依赖 K8s PodTemplate

`SoftwareServiceImpl` 直接把 `PodTemplate` 当 software catalog。这个思想可参考，但实现不应直接照搬。

## 3. 创建 MySQL 数据库这个动作，未来 assistant 最可能怎么落地

当前代码里，这是最清晰的一条同步直写链路：

- `MysqlDatabaseController.save(...)`
- `MysqlDatabaseServiceImpl.save(...)`
- `MysqlDatabaseCheck.checkSave(...)`
- `FeatureService.buildJdbcTemplate(...)`
- `JdbcTemplate.execute("CREATE DATABASE ...")`

因此 future assistant 最合理的落地方式不是 CRD，也不是工单系统强包裹，而是：

```text
Deep Agent
  -> 生成标准动作 mysql.database.create
  -> Control Layer 做资产解析/风控/审计
  -> DB-native adapter 执行 SQL
  -> 返回结果并写入审计账本
```

只有在以下情况才应该升级为审批或工单：

- 目标是生产高敏实例
- 库名命中关键模式
- 与授权动作绑定执行
- 涉及批量租户变更

## 4. MCP 模式和非 MCP 模式如何兼容

核心不是“选 MCP 还是不用 MCP”，而是建立一个统一的动作控制层。

推荐结构：

```text
Deep Agent
  -> Action Normalizer
  -> Control Layer
     -> Policy / Risk / Approval
     -> Execution Router
        -> MCP Tool Adapter
        -> CRD Adapter
        -> GrpcCall Adapter
        -> DB-native Adapter
        -> K8s Adapter
        -> Shell/Ansible/VM Adapter
```

兼容原则：

1. Deep Agent 只产出标准动作，不直接决定最终执行器。
2. Control Layer 根据动作、资产、环境、风险来路由。
3. 若某能力天然是 MCP tool，则走 MCP。
4. 若某能力需要强审计、强约束、强幂等、强审批，则走自研受控 adapter。
5. 两条路径最终都写入同一套 task/audit/evidence 体系。

换句话说，MCP 不是控制层的替代品，而只是执行器家族之一。

## 5. Assistant Control Layer 为什么必须存在

这是这次分析里最重要的判断。

如果没有 Control Layer，Deep Agent 最终只能：

- 直接调工具
- 自己做资源判断
- 自己决定风险
- 自己拼执行路径

这在企业级 DBA 场景里是不可接受的，因为会立刻丢失：

- 权限边界
- 风险边界
- 审批边界
- 审计边界
- 幂等边界
- 证据边界

而 `upm-api-server` 代码已经反向证明了这层是必要的，只是它今天散落在不同 Java service 里：

- 一部分在 `resource`
- 一部分在 `service-common`
- 一部分在 `auth/user`
- 一部分在 `mysql` 等具体模块

future assistant 需要做的，不是调用这些 Java controller，而是把这些分散语义收敛成一个独立的 Control Layer。

## 6. 最终判断

### 6.1 可直接借鉴

- 资产目录
- 服务聚合模型
- 工单对象边界
- 任务/子任务状态模型
- K8s/CRD/gRPC/DB-native 适配思路
- 基础审计字段

### 6.2 可参考但需重构

- URL 权限模型
- 页面型 REST 设计
- Java 微服务调用方式
- 轻量审批状态
- PodTemplate 驱动的软件目录

### 6.3 必须自建

- Assistant Control Layer
- 意图规范化
- 统一动作字典
- 风险分级
- 审批策略
- 审计账本
- 证据包
- 执行路由
- MCP 兼容层
- 失败补偿/回滚建议

## 7. 建议阅读顺序

1. `01-upm-control-plane-reverse-engineering.md`
2. `02-assistant-reusable-capabilities.md`
3. `03-action-to-execution-mapping.md`

如果只看一份，优先看 `03-action-to-execution-mapping.md`，因为它最接近后续 coding 时真正要落地的动作字典和执行路由。
