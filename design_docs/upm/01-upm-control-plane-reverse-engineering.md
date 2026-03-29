# upm-api-server 控制面逆向建模文档

## 0. 分析说明

### 0.1 结论口径

- `代码事实`：来自本地仓库代码扫描，优先引用真实类、方法、文件路径。
- `架构判断`：基于代码事实，对未来 `Deep Agent + Control Layer + Adapter/MCP` 架构的提炼。

### 0.2 资料边界

- 已扫描本地仓库主模块：`upm-common`、`upm-user`、`upm-auth`、`upm-gateway`、`upm-resource`、`upm-service`。
- 已重点跟踪 MySQL、认证、权限、资源注册、任务/工单、审计日志、K8s/CRD/gRPC 执行链。
- 本地未发现与 `UPM Platform / UPM Engine / Kauntlet / Tesseract Cube` 明确对应的设计文档；相关内容以代码推断为主，统一标记为 `文档待补充`。
- 本地 `.md` 文档极少，能作为设计依据的主要是代码与 OpenAPI 注解；未发现完整平台架构说明。

### 0.3 本地未发现的文档

- `README.md` 仅为仓库模板，几乎不提供架构信息。
- `upm-auth/upm-auth-ms/README.md` 主要是 Helm/部署说明。
- 未发现 `UPM Platform / Engine / Kauntlet / Tesseract Cube` 架构文档。
- 结论：本次分析以代码逆向为主，相关概念文档需后续补充。

## 1. 项目总体结构概览

### 1.1 根模块

根 `pom.xml` 定义了 6 个一级模块：

- `upm-common`
- `upm-user`
- `upm-auth`
- `upm-gateway`
- `upm-resource`
- `upm-service`

文件依据：

- `pom.xml`

### 1.2 一级模块职责

| 模块 | 主要职责 | 关键证据 |
| --- | --- | --- |
| `upm-common` | 公共模型、返回结构、注解、工具、springdoc/openapi 配置 | `upm-common/upm-springdoc/src/main/java/io/syntropycloud/upm/doc/configuration/SwaggerOpenApiConfiguration.java` |
| `upm-user` | 用户、角色、权限菜单、组、License、OperateLog 落库 | `upm-user/upm-user-ms/src/main/java/io/syntropycloud/upm/user/ms/service/impl/UserServiceImpl.java` |
| `upm-auth` | 登录、JWT、Redis token、登录态校验、对下游暴露 auth check | `upm-auth/upm-auth-ms/src/main/java/io/syntropycloud/upm/auth/ms/service/impl/AuthServiceImpl.java` |
| `upm-gateway` | 网关入口、签名校验、鉴权转发、统一错误处理 | `upm-gateway/upm-gateway-ms/src/main/java/io/syntropycloud/upm/gateway/ms/filter/AuthorizationFilter.java` |
| `upm-resource` | 资源控制面：project/cluster/zone/node/node group/storage class/s3/software | `upm-resource/upm-resource-ms/src/main/java/io/syntropycloud/upm/resource/ms/controller` |
| `upm-service` | 各中间件/数据库服务控制面；`upm-service-common` 提供抽象控制流和执行框架 | `upm-service/upm-service-common/upm-service-common-core/src/main/java/io/syntropycloud/upm/service/common/core/service` |

### 1.3 `upm-service` 子模块图谱

`upm-service` 下存在以下服务家族：

- `upm-service-common`
- `upm-service-mysql`
- `upm-service-redis`
- `upm-service-redis-cluster`
- `upm-service-mongodb`
- `upm-service-milvus`
- `upm-service-kafka`
- `upm-service-elasticsearch`
- `upm-service-postgresql`
- `upm-service-innodb-cluster`
- `upm-service-zookeeper`

目录依据：

- `upm-service/*`

### 1.4 分层理解

按控制面职责可粗分为：

| 层次 | 主要模块/类 | 说明 |
| --- | --- | --- |
| 网关层 | `AuthorizationFilter`、`GlobalRequestSignFilter` | 请求签名、token、URL 级权限拦截 |
| 鉴权层 | `AuthController`、`AuthServiceImpl`、`UserServiceImpl.checkApp` | 登录态和路由权限判定 |
| 资源层 | `ProjectController`、`ClusterController`、`NodeController`、`StorageClassController` 等 | 把 K8s/S3/节点等外部资产注册为平台受控对象 |
| 业务层 | `Mysql*Controller`、`ServGroupController`、`OrderGroupController`、`UnitController` | 面向数据库/中间件的服务编排与管理 |
| 任务层 | `OrderGroupCoreService`、`TaskSchedulerCoreService`、`TaskCoreService` | 工单、任务、子任务、异步执行 |
| 执行适配层 | `KubeResourceCoreService`、`SubtaskExecuteCoreService`、各模块 `KubeResourceService` / `SubtaskExecuteService` | 把控制意图下发到 K8s CRD、GrpcCall、数据库直连等执行面 |
| 日志审计层 | 各模块 `OperateLogAspect`、`OperateLogController` | 请求级操作审计 |
| 事件层 | `EventCoreService`、`saveEvent(...)`、备份定时任务 | 面向服务对象的事件记录，不等同于完整审计账本 |

### 1.5 模块依赖关系

可逆向理解为：

```text
Client / UI
  -> upm-gateway
     -> upm-auth
        -> upm-user
     -> upm-resource
        -> K8s / S3 / Prometheus
     -> upm-service-<middleware>
        -> upm-service-common
        -> upm-resource (openfeign)
        -> upm-user / upm-auth (openfeign)
        -> K8s CRD / gRPC / DB-native / S3
```

### 1.6 对未来 Assistant 的第一判断

- `upm-api-server` 的真正价值不在 Java/Spring 本身，而在它已经沉淀出一套较完整的控制面语义：
  - 资源注册
  - 服务聚合
  - 变更申请
  - 异步任务
  - 子任务编排
  - 操作审计
  - 执行适配
- 未来 Assistant 不应依赖这些 Java 模块运行，但非常值得复用其对象边界、状态模型和动作分层。

## 2. 核心领域模型

### 2.1 文字版关系图

```text
User --belongs to--> Role --grants--> App(url/method/menu permission)
User --visible by--> Group / visibleDataScope
User --produces--> OperateLog

Project --contains--> Service Groups
Cluster --contains--> Zone --contains--> NodeGroup --contains--> Node
Cluster --contains--> StorageClass
S3Storage --stores--> BackupFile
Software(PodTemplate) --supplies--> deployable versions/capabilities

ServiceGroup --contains--> Service --contains--> Unit
ServiceGroup change request --> OrderGroup --contains--> Order
Execution runtime --> Task --contains--> Subtask
Task/Subtask --> downstream executor (K8s CRD / GrpcCall / DB-native)

MySQL BackupStrategy --targets--> ServiceGroup
MySQL BackupFile --belongs to--> Unit / Service / ServiceGroup / S3Storage / Task
```

### 2.2 关键对象清单

| 对象 | 所在模块 | 关键类 / 文件路径 | 作用 | 关系 | 对 Assistant 的建模价值 |
| --- | --- | --- | --- | --- | --- |
| `Project` | `upm-resource` | `upm-resource/upm-resource-ms/src/main/java/io/syntropycloud/upm/resource/ms/domain/ProjectDO.java` | 业务租户/命名空间抽象 | 挂在 `Cluster` 中创建 namespace；承载 `ServGroup` | 未来资产解析必须保留 `project` 维度，避免 assistant 直接面向裸集群执行 |
| `Cluster` | `upm-resource` | `upm-resource/upm-resource-ms/src/main/java/io/syntropycloud/upm/resource/ms/domain/ClusterDO.java` | K8s 集群注册对象，持有 `relateName` 和连接配置 | 向下包含 zone/storageclass/software/node | 是受控执行的顶层基础设施边界 |
| `Zone` | `upm-resource` | `upm-resource/upm-resource-ms/src/main/java/io/syntropycloud/upm/resource/ms/domain/ZoneDO.java` | 集群下的部署域 | `Cluster -> Zone -> NodeGroup` | 可作为 placement / failure domain |
| `NodeGroup` | `upm-resource` | `upm-resource/upm-resource-ms/src/main/java/io/syntropycloud/upm/resource/ms/domain/NodeGroupDO.java` | 主机组/节点组抽象 | 包含 `Node` | 未来 `hostgroup`/`nodegroup` 应统一建模 |
| `Node` | `upm-resource` | `upm-resource/upm-resource-ms/src/main/java/io/syntropycloud/upm/resource/ms/domain/NodeDO.java` | 已注册主机/节点 | 从属 `NodeGroup`，映射 K8s node | 资产定位和执行选路的重要输入 |
| `StorageClass` | `upm-resource` | `upm-resource/upm-resource-ms/src/main/java/io/syntropycloud/upm/resource/ms/domain/StorageClassDO.java` | K8s storage class 的内部注册对象 | 从属 `Cluster` | 体现“外部能力先注册、再被服务引用”的控制面语义 |
| `StorageS3` | `upm-resource` | `upm-resource/upm-resource-ms/src/main/java/io/syntropycloud/upm/resource/ms/domain/StorageS3DO.java` | 备份对象存储注册 | 被 `MysqlBackupStrategyDO`、`MysqlBackupFileDO` 引用 | 备份/证据包/快照资产中心的原型 |
| `Software` | `upm-resource` | `upm-resource/upm-resource-ms/src/main/java/io/syntropycloud/upm/resource/ms/service/impl/SoftwareServiceImpl.java` | 不是本地 DB 实体，而是基于 K8s `PodTemplate` 暴露的软件版本对象 | 受 `Cluster.adminNamespace` 管理 | 未来版本目录可做成独立 catalog，不一定绑定 K8s PodTemplate |
| `ServGroup` | `upm-service-common` | `upm-service/upm-service-common/upm-service-common-core/src/main/java/io/syntropycloud/upm/service/common/core/domain/ServGroupDO.java` | 一个逻辑服务组/部署聚合 | 包含多个 `Serv` | 是未来 assistant 最值得保留的服务聚合对象 |
| `Serv` | `upm-service-common` | `upm-service/upm-service-common/upm-service-common-core/src/main/java/io/syntropycloud/upm/service/common/core/domain/ServDO.java` | 服务组中的具体服务角色，如 mysql/proxysql | 从属 `ServGroup`，包含 `Unit` | 对应未来 `service component` 或 `role instance set` |
| `Unit` | `upm-service-common` | `upm-service/upm-service-common/upm-service-common-core/src/main/java/io/syntropycloud/upm/service/common/core/domain/UnitDO.java` | 最小执行单元，对应具体实例 | 从属 `Serv` | 未来行动应尽量落到 unit 级，便于局部修复/重建/备份 |
| `UnitSet` | K8s CRD 侧对象 | `upm-service/upm-service-common/upm-service-common-core/src/main/java/io/syntropycloud/upm/service/common/core/service/KubeResourceCoreService.java` | 不是本地 DO，而是通过 CRD 构造的运行时集合 | 由 `Serv` 映射生成 | 对未来 adapter 很关键，体现“控制面对象 != 执行面对象” |
| `OrderGroup` | `upm-service-common` | `upm-service/upm-service-common/upm-service-common-core/src/main/java/io/syntropycloud/upm/service/common/core/domain/OrderGroupDO.java` | 变更申请/work order 聚合 | 包含多个 `Order`，可绑定已有 `ServGroup` | 是未来审批、风控、执行前冻结的核心模型 |
| `Order` | `upm-service-common` | `upm-service/upm-service-common/upm-service-common-core/src/main/java/io/syntropycloud/upm/service/common/core/domain/OrderDO.java` | 服务级 desired state 及 pre-state 快照 | 从属 `OrderGroup` | 非常适合转译成 assistant 的标准动作参数集 |
| `Task` | `upm-service-common` | `upm-service/upm-service-common/upm-service-common-core/src/main/java/io/syntropycloud/upm/service/common/core/domain/TaskDO.java` | 异步执行运行时对象 | 包含 `Subtask` | 应保留为 Control Layer 的标准执行记录 |
| `Subtask` | `upm-service-common` | `upm-service/upm-service-common/upm-service-common-core/src/main/java/io/syntropycloud/upm/service/common/core/domain/SubtaskDO.java` | 可排序、可并发的执行步骤 | 从属 `Task` | 未来 action plan/execution plan 的天然载体 |
| `User` | `upm-user` | `upm-user/upm-user-ms/src/main/java/io/syntropycloud/upm/user/ms/domain/UserDO.java` | 用户主体 | 绑定 `Role` / `Group` | 未来 assistant 仍需主体、代理人和发起人分离 |
| `Role` | `upm-user` | `upm-user/upm-user-ms/src/main/java/io/syntropycloud/upm/user/ms/domain/RoleDO.java` | 角色与可见范围 | 绑定 `App` 权限 | 现有模型偏 UI/URL 权限，未来需增强为 action/resource policy |
| `Permission/App` | `upm-user` | `upm-user/upm-user-ms/src/main/java/io/syntropycloud/upm/user/ms/domain/AppDO.java` | 菜单 + API 权限树 | `Role -> App(url,method)` | 可复用命名和树结构，但不能直接作为 assistant 权限内核 |
| `OperateLog` | `upm-user` | `upm-user/upm-user-ms/src/main/java/io/syntropycloud/upm/user/ms/domain/OperateLogDO.java` | 操作审计日志 | 各服务 `OperateLogAspect` 汇总后写入 | 可直接借鉴字段，但仍缺 AI 场景所需证据和意图链 |
| `Event` | `upm-service-common` | `upm-service/upm-service-common/upm-service-common-core/src/main/java/io/syntropycloud/upm/service/common/core/domain/EventDO.java` | 服务对象事件 | 与 `ServGroup` 关联 | 更像通知/状态事件，不是完整审计账本 |
| `MysqlBackupStrategy` | `upm-service-mysql` | `upm-service/upm-service-mysql/upm-service-mysql-ms/src/main/java/io/syntropycloud/upm/service/mysql/ms/domain/MysqlBackupStrategyDO.java` | 周期备份策略 | 指向 `ServGroup` 和 `StorageS3` | 未来可以沉为标准计划任务对象 |
| `MysqlBackupFile` | `upm-service-mysql` | `upm-service/upm-service-mysql/upm-service-mysql-ms/src/main/java/io/syntropycloud/upm/service/mysql/ms/domain/MysqlBackupFileDO.java` | 备份产物元数据 | 指向 `Task` / `Unit` / `S3` | 很适合作为 evidence/artifact 元模型 |

### 2.3 领域对象上的关键判断

#### 代码事实

- `ServGroup / Serv / Unit` 与 `OrderGroup / Order` 分离得很清楚。
- 资源侧的 `Cluster / Zone / NodeGroup / Node / StorageClass / StorageS3` 形成了完整的资产目录。
- `Software` 更像执行面模板目录，而不是业务资源库。

#### 架构判断

- 未来 assistant 最值得保留的是“两套模型分离”：
  - `资产模型`：项目、集群、区域、节点、存储、软件目录
  - `变更模型`：意图、动作、工单、执行计划、任务、证据
- 这套分离比任何单个 controller 更重要。

## 3. 核心控制流逆向

下面只分析与未来 DBA Assistant 建模直接相关的链路。

### 3.1 创建 MySQL 数据库

#### 入口

- Controller: `MysqlDatabaseController.save(...)`
- 路径: `POST /serv_groups/{serv_group_id}/mysql/databases`
- 文件: `upm-service/upm-service-mysql/upm-service-mysql-ms/src/main/java/io/syntropycloud/upm/service/mysql/ms/controller/MysqlDatabaseController.java`

#### 调用链

1. `MysqlDatabaseController.save(...)`
2. `MysqlDatabaseServiceImpl.save(servGroupId, MysqlDatabaseForm)`
3. `MysqlDatabaseCheck.checkSave(...)`
4. `FeatureService.buildJdbcTemplate(servGroupDO)`  
   文件待补充细读，但从调用可确认是直接拿到 MySQL JDBC 连接
5. `JdbcTemplate.execute("CREATE DATABASE IF NOT EXISTS ...")`

#### 流转特点

- 同步执行
- 不落 `OrderGroup`
- 不落 `Task`
- 有 `@OperateLog(action = ADD)`，因此会进入 MySQL 模块 `OperateLogAspect`
- 最终执行入口：`DB-native`

#### 对未来 Assistant 的意义

- 这是最典型的 `控制面 -> DB-native adapter` 动作。
- 不需要为了这种动作引入复杂工单，除非触发风控条件。

### 3.2 创建 MySQL 用户

#### 入口

- Controller: `MysqlUserController.save(...)`
- 路径: `POST /serv_groups/{serv_group_id}/mysql/users`
- 文件: `upm-service/upm-service-mysql/upm-service-mysql-ms/src/main/java/io/syntropycloud/upm/service/mysql/ms/controller/MysqlUserController.java`

#### 调用链

1. `MysqlUserController.save(...)`
2. `MysqlUserServiceImpl.save(servGroupId, MysqlUserForm)`
3. `MysqlUserCheck.checkSave(...)`
4. `FeatureService.buildJdbcTemplate(servGroupDO)`
5. 执行 SQL：
   - `CREATE USER ...`
   - 对每个 grant 执行 `GRANT ...`
6. 异常时补偿：
   - `DROP USER IF EXISTS ...`

#### 流转特点

- 同步执行
- 不落 `Task` / `OrderGroup`
- 逻辑里已经做了局部补偿
- 最终执行入口：`DB-native`

#### 建模判断

- 对未来 assistant，`mysql.user.create` 和 `mysql.user.grant` 最好拆成两个标准动作。
- 现有代码把两者放在一个请求里，这对 UI 友好，但对 agent 动作字典不够细。

### 3.3 Grant 权限

#### 入口与现实情况

- 没有单独的 `grant` controller。
- 当前代码把 grant 合并在：
  - `MysqlUserServiceImpl.save(...)`
  - `MysqlUserServiceImpl.update(...)`

#### 调用链

1. `MysqlUserController.save(...)` 或 `MysqlUserController.update(...)`
2. `MysqlUserServiceImpl.save/update(...)`
3. 对 grant 增量执行：
   - `GRANT ...`
   - `REVOKE ...`

#### 结论

- `mysql.user.grant` 在现有系统中是“内嵌子动作”，不是显式一等动作。
- 这是未来 assistant 需要重构的地方：assistant 里应把“用户创建”和“授权变更”区分为独立 action。

### 3.4 创建 MySQL 服务

#### 入口

创建不是直接 `POST /serv_groups`，而是走工单：

1. `OrderGroupController.save(...)`  
   `POST /order_groups`
2. `OrderGroupController.approve(...)`  
   `PUT /order_groups/{order_group_id}/approval`
3. `OrderGroupController.execute(...)`  
   `PUT /order_groups/{order_group_id}/execute`

关键文件：

- `upm-service/upm-service-mysql/upm-service-mysql-ms/src/main/java/io/syntropycloud/upm/service/mysql/ms/controller/OrderGroupController.java`
- `upm-service/upm-service-mysql/upm-service-mysql-ms/src/main/java/io/syntropycloud/upm/service/mysql/ms/service/impl/OrderGroupServiceImpl.java`
- `upm-service/upm-service-common/upm-service-common-core/src/main/java/io/syntropycloud/upm/service/common/core/service/OrderGroupCoreService.java`

#### 调用链

1. `OrderGroupController.save(...)`
2. `OrderGroupServiceImpl.save(...)`
3. `BaseService.buildOrderGroupCoreForm(orderGroupForm)`  
   把 `mysql` / `proxysql` 表单转成标准 `OrderCoreForm`
4. `OrderGroupCoreService.save(...)`  
   落 `OrderGroupDO` 和 `OrderDO`
5. 审批时 `OrderGroupCoreService.approve(...)`  
   只是更新 `status + msg`
6. 执行时 `OrderGroupCoreService.execute(...)`
7. 根据 `orderGroupDO.type == NEW` 调用 `ServGroupCoreService.create(...)`
8. `ServGroupCoreService` 构造 `TaskDO + SubtaskDO`
9. `TaskSchedulerCoreService.saveTaskAndExecutor(taskDO)`
10. 运行时再进入 MySQL 模块 `ServGroupServiceImpl.executeSubtask(...)`
11. 继续分发到：
    - `SubtaskExecuteService.createMysqlReplication(...)`
    - `KubeResourceCoreService.createUnitSet(...)`
    - `KubeResourceCoreService.createSecret(...)`
    - 可选 `PROXYSQL_SYNC`

#### 子任务形态

MySQL 创建任务里常见子任务：

- `CREATE_SECRET`
- `CREATE` mysql unitset
- `CREATE_REPLICATION`
- 可选 `CREATE` proxysql unitset
- 可选 `PROXYSQL_SYNC`

#### 流转特点

- 明确异步
- 明确工单前置
- 落 `OrderGroup`
- 落 `Task` / `Subtask`
- 有操作日志
- 最终执行入口主要是 `K8s / CRD`，并伴随 MySQL 专属 CRD/同步对象

#### 建模价值

- 这是未来 assistant 最值得继承的控制流：  
  `意图 -> 变更单 -> 审批 -> 执行计划 -> 子任务 -> 适配器`

### 3.5 MySQL 扩缩容

#### 入口

当前代码不是一个统一 `scale` 动作，而是拆成三类：

- `PUT /serv_groups/{serv_group_id}/computer_scaling`
- `PUT /serv_groups/{serv_group_id}/storage_expansion`
- `PUT /serv_groups/{serv_group_id}/unit_expansion`

文件：

- `upm-service/upm-service-mysql/upm-service-mysql-ms/src/main/java/io/syntropycloud/upm/service/mysql/ms/controller/ServGroupController.java`

#### 调用链

1. `ServGroupController.computerScaling/storageExpansion/unitExpansion(...)`
2. MySQL `ServGroupServiceImpl` 进入 `ServGroupCoreService`
3. `ServGroupCoreService` 将变更固化为 `OrderGroupDO` / `OrderDO`
4. 生成异步 `Task`
5. 执行阶段落到 `KubeResourceCoreService`：
   - `updateUnitSetResource`
   - `updateUnitSetStorage`
   - `updateUnitSetUnits`
6. MySQL 特定副本关系更新再由 `SubtaskExecuteService.updateMysqlReplication(...)` 补齐

#### 流转特点

- 异步
- 有工单/任务
- 有审批入口
- 最终执行入口：`K8s/CRD + MySQL replication CRD`

#### 建模判断

- 未来 assistant 可定义统一动作 `mysql.service.scale`，但内部必须带 `dimension`：
  - `compute`
  - `storage`
  - `replica`
- 这比直接复制 UPM 的三个 HTTP 接口更适合 agent 编排。

### 3.6 发起备份

#### 当前系统里有两条备份路径

##### 路径 A：针对整个服务组

- 入口：`ServGroupController.backup(...)`
- 路径：`PUT /serv_groups/{serv_group_id}/backup`
- 文件：`upm-service/upm-service-mysql/upm-service-mysql-ms/src/main/java/io/syntropycloud/upm/service/mysql/ms/controller/ServGroupController.java`

调用链：

1. `ServGroupController.backup(...)`
2. `ServGroupServiceImpl.backup(...)`
3. `servGroupCheck.checkBackup(...)`
4. `buildBackupTask(servGroupDO, ..., MysqlReplication, MysqlBackupForm)`
5. 根据 replication role 选中目标 unit，给每个 unit 生成 `BACKUP` 子任务
6. `saveTaskAndExecutor(taskDO)`
7. `ServGroupServiceImpl.executeSubtask(...)`
8. `SubtaskExecuteService.backup(...)`

##### 路径 B：针对单个 unit

- 入口：`UnitController.backup(...)`
- 路径：`PUT /units/{unit_id}/backup`
- 文件：`upm-service/upm-service-mysql/upm-service-mysql-ms/src/main/java/io/syntropycloud/upm/service/mysql/ms/controller/UnitController.java`

调用链：

1. `UnitController.backup(...)`
2. `UnitServiceImpl.backup(...)`
3. `execute(id, BACKUP, paramJson)`
4. `buildBackupTask(unitDO, ...)`
5. 生成单个 `BACKUP` 子任务
6. `saveTaskAndExecutor(taskDO)`
7. `SubtaskExecuteService.backup(...)`

#### 最终执行入口

`SubtaskExecuteService.backup(...)` 做了完整闭环：

1. 校验 `StorageS3`
2. 校验 bucket
3. 构造 `GrpcCall`
4. 创建 `MysqlBackupFileDO`
5. 轮询 `GrpcCall` 状态
6. 成功后回填 S3 目录大小和备份状态

关键文件：

- `upm-service/upm-service-mysql/upm-service-mysql-ms/src/main/java/io/syntropycloud/upm/service/mysql/ms/service/impl/SubtaskExecuteService.java`
- `upm-service/upm-service-common/upm-service-common-core/src/main/java/io/syntropycloud/upm/service/common/core/service/KubeResourceCoreService.java`

#### 流转特点

- 异步
- 不走 `OrderGroup`
- 走 `Task` / `Subtask`
- 产出备份文件对象
- 最终执行入口：`GrpcCall CRD + S3`

#### 建模判断

- 未来 assistant 的 `mysql.backup.create` 应优先建模为：
  - 控制动作
  - 产物对象
  - 证据对象
- `MysqlBackupFileDO` 已经非常接近 future evidence/artifact registry。

### 3.7 查询任务状态

#### 入口

- `TaskController.list(...)` -> `GET /tasks`
- `TaskController.get(...)` -> `GET /tasks/{task_id}`
- `TaskController.cancel(...)` -> `PUT /tasks/{task_id}/cancel`

文件：

- `upm-service/upm-service-mysql/upm-service-mysql-ms/src/main/java/io/syntropycloud/upm/service/mysql/ms/controller/TaskController.java`
- `upm-service/upm-service-mysql/upm-service-mysql-ms/src/main/java/io/syntropycloud/upm/service/mysql/ms/service/impl/TaskServiceImpl.java`
- `upm-service/upm-service-common/upm-service-common-core/src/main/java/io/syntropycloud/upm/service/common/core/service/TaskCoreService.java`

#### 运行时模型

- `TaskDO` 记录动作级执行状态
- `SubtaskDO` 记录步骤级执行状态、超时、错误消息
- `TaskSchedulerCoreService` 根据 `priority` 分批并发执行 subtasks
- 每 10 秒写一次 task 心跳到 Redis

#### 建模判断

- 这是未来 Assistant Control Layer 的直接蓝本。
- 尤其适合用来承载：
  - 多 adapter 编排
  - 状态轮询
  - 子步骤并发
  - 中途取消

### 3.8 审批/执行工单

#### 代码事实

- `OrderGroupController.approve(...)` 只接收 `ApproveCoreForm`
- `OrderGroupCoreService.approve(...)` 只是更新 `status` 和 `msg`
- `OrderGroupCoreCheck.checkApprove(...)` 只校验：
  - 是否有权限
  - 当前状态是否可审批
  - 目标状态是否是 `APPROVED` / `REJECTED`

#### 结论

- 当前仓库里存在“审批状态位”，但没有发现完整的多级审批流、审批节点编排、审批策略引擎。
- 更像是：平台/UI/workflow 层发起审批，人审结果回写当前服务。
- 因此应明确标注：  
  `审批可能主要在 platform/UI/workflow 层，而非当前代码完整承载`

## 4. 审计与权限模型

### 4.1 权限校验发生在哪一层

#### 网关层

- `AuthorizationFilter`
- `GlobalRequestSignFilter`

作用：

- 校验签名
- 校验 token
- 向 auth 服务确认当前 path/method 是否允许
- 透传 `x-upm-user-id` 到下游

文件：

- `upm-gateway/upm-gateway-ms/src/main/java/io/syntropycloud/upm/gateway/ms/filter/AuthorizationFilter.java`
- `upm-gateway/upm-gateway-ms/src/main/java/io/syntropycloud/upm/gateway/ms/filter/GlobalRequestSignFilter.java`

#### Auth 层

- `AuthServiceImpl.check(AuthAppForm)`
- 从 Redis 取 token，判定是否登录
- 通过 `userClient.checkApp(userId, appForm)` 进行 URL/method 权限校验

文件：

- `upm-auth/upm-auth-ms/src/main/java/io/syntropycloud/upm/auth/ms/service/impl/AuthServiceImpl.java`

#### User 层

- `UserServiceImpl.checkApp(...)`
- 规则核心是：
  - 超级管理员 bypass
  - 白名单 URL bypass
  - 先查 role 与 url/method 精确匹配
  - 再做 path-template 模式匹配

文件：

- `upm-user/upm-user-ms/src/main/java/io/syntropycloud/upm/user/ms/service/impl/UserServiceImpl.java`

### 4.2 审计记录发生在哪一层

- 各模块通过 `@OperateLog` + `OperateLogAspect` 做 AOP 记录。
- 已看到的 aspect：
  - `upm-auth/.../OperateLogAspect.java`
  - `upm-user/.../OperateLogAspect.java`
  - `upm-resource/.../OperateLogAspect.java`
  - `upm-service-mysql/.../OperateLogAspect.java`
- 最终写入 `OperateLogController.save(...)` / `OperateLogServiceImpl.save(...)`。

### 4.3 审计可记录的字段

`OperateLogDO` 已有字段：

- `msName`
- `traceId`
- `parentSpanId`
- `spanId`
- `objType`
- `action`
- `objName`
- `description`
- `body`
- `uptimeMillisecond`
- `success`
- `errMsg`
- `clientIp`
- `unixTimestamp`
- `creator`

文件：

- `upm-user/upm-user-ms/src/main/java/io/syntropycloud/upm/user/ms/domain/OperateLogDO.java`

### 4.4 审计粒度判断

#### 代码事实

- 这是请求级 / 接口级审计。
- 它能很好记录“谁调用了哪个接口，参数是什么，成功还是失败”。
- 它不天然记录：
  - assistant 原始意图
  - 规划推理过程
  - 风险评估
  - 审批链
  - 执行证据集合
  - 下游多跳 adapter 的统一 execution trace

#### 架构判断

- 对未来 AI assistant 来说，这套审计“不够用，但很有价值”。
- 它适合成为 `audit event` 的一个来源，而不是完整账本。

### 4.5 权限模型是否足够支撑未来 AI assistant

#### 代码事实

- 当前权限对象本质是 `Role -> App(url, method)`。
- 当前鉴权主要基于 HTTP 路由与菜单树，而不是标准动作。

#### 不足

- 无法自然表达：
  - “允许创建数据库，但禁止 grant 超级权限”
  - “允许在 dev project 做 R1 操作，但 prod 只能 R0”
  - “允许调用动作，但必须经过审批”
  - “允许 assistant 提建议，但不允许 assistant 直接执行”

#### 结论

- 未来 assistant 必须把权限模型从 `URL 权限` 升级为 `Action + Resource + Risk + Approval policy`。

## 5. 统一返回、异常、OpenAPI、请求校验

### 5.1 统一返回

- 普遍使用 `Result<T>` 作为统一返回。
- 这对未来 Control Layer 很有参考价值：  
  成功/失败语义统一，但未来应补 `decisionId / taskId / evidenceId / approvalId` 等字段。

### 5.2 统一异常处理

- 各服务模块普遍有 `GlobalExceptionHandler`
- service-common 里：
  - `ServiceException` 被单独处理
  - 其他异常按角色决定暴露程度

文件：

- `upm-service/upm-service-common/upm-service-common-core/src/main/java/io/syntropycloud/upm/service/common/core/configuration/GlobalExceptionHandler.java`

### 5.3 输入标准化

- `StringHttpMessageConverterAdvice` 会自动 trim 所有字符串字段。
- 各业务操作在 service 层普遍有 `*Check` 校验类，返回 `ChkRs`。

文件：

- `upm-service/upm-service-common/upm-service-common-core/src/main/java/io/syntropycloud/upm/service/common/core/configuration/StringHttpMessageConverterAdvice.java`
- 例如：
  - `MysqlUserCheck`
  - `MysqlDatabaseCheck`
  - `ClusterCheck`
  - `OrderGroupCheck`

### 5.4 OpenAPI / Swagger

- 基于 `springdoc` 统一生成 OpenAPI。
- `SwaggerOpenApiConfiguration` 做了较多控制：
  - tag 排序
  - 参数排序
  - header 注入
  - 错误响应补齐
  - 统一描述签名要求

文件：

- `upm-common/upm-springdoc/src/main/java/io/syntropycloud/upm/doc/configuration/SwaggerOpenApiConfiguration.java`

### 5.5 请求签名

`SwaggerOpenApiConfiguration` 与 `GlobalRequestSignFilter` 都表明：

- 所有 API 调用要求 `x-upm-signature`
- 需要 `x-upm-timestamp`
- 签名格式是 `method + url + timestamp + body`

这说明当前平台把“请求级防篡改”看作控制面的一部分。

## 6. 下游执行能力逆向

### 6.1 Kubernetes / CRD

是当前系统最主要的执行面。

关键证据：

- `KubeResourceCoreService`
- `K8sStorageClassService`
- `K8sNodeService`
- `k8sUnitSetCrdService`
- `k8sGrpcCallCrdService`

结论：

- 大量动作最终不是 shell，而是通过 K8s API / CRD 落地。

### 6.2 GrpcCall

是非常关键的“受控远程执行适配器”原型。

关键方法：

- `KubeResourceCoreService.buildGrpcCall(...)`
- `KubeResourceCoreService.createGrpcCall(...)`
- `KubeResourceCoreService.pollingSubTaskGrpcCallStatus(...)`

用例：

- 动态参数更新
- clone data
- backup
- restore
- gtid purge

### 6.3 DB-native

目前在 MySQL 模块里明确存在：

- `MysqlDatabaseServiceImpl`
- `MysqlUserServiceImpl`

这是未来 assistant 必须保留的一类执行路径。

### 6.4 S3

资源注册和备份文件元数据高度依赖 S3：

- `StorageS3ServiceImpl`
- `MysqlBackupFileServiceImpl`
- `SubtaskExecuteService.backup(...)`

### 6.5 定时任务 / 异步任务

已发现：

- `TaskSchedulerCoreService` 线程池异步执行
- `MysqlBackupStrategyTask` 周期备份
- `MysqlBackupFileStatusUpdateTask`
- `MysqlBackupFileCleanTask`
- `ShedLockConfiguration` 防止定时任务并发竞争

### 6.6 未发现的执行方式

本次扫描未发现 Java 运行时直接调用：

- `shell`
- `ansible`
- `python`
- `ProcessBuilder`
- `Runtime.exec`

说明：

- 仓库中有部署脚本 `script/*.sh`，但那是容器启动/部署层，不是业务执行链。
- 因此可以初步判断：当前控制面更偏 `K8s API / CRD / DB-native / S3`，不是 `shell orchestration`。
- 若外部仍有 operator/agent 承接 shell，相关实现不在本仓库内，属于 `文档待补充/代码待补充`。

## 7. 关键结论

### 7.1 这个项目真正的控制面价值在哪里

最有价值的不是 Java 接口本身，而是以下五层语义已经成型：

1. `资源注册语义`  
   外部基础设施先被注册为内部受控对象，再被业务服务引用。
2. `服务聚合语义`  
   `ServGroup -> Serv -> Unit` 把“服务逻辑对象”和“运行时实例对象”分开了。
3. `变更语义`  
   `OrderGroup -> Order` 把“申请的目标状态”和“当前运行状态”分开了。
4. `执行语义`  
   `Task -> Subtask` 把复杂动作拆成可排序、可并发、可观察的执行步骤。
5. `适配器语义`  
   控制面不直接等于执行面，最终通过 `K8s / CRD / GrpcCall / DB-native / S3` 下发。

### 7.2 哪些是未来 assistant 最值得借鉴的控制面语义

- `资源先注册、动作再解析`
- `先形成标准动作，再路由到执行适配器`
- `高风险动作先固化成工单，再执行`
- `异步任务必须拆成子任务并保留状态`
- `操作日志、事件、任务记录要分层，不要混成一种表`

### 7.3 对 future DBA Assistant 的总判断

如果未来要基于 Deep Agent 做企业级 DBA Assistant，而不直接依赖 Java platform，那么 `upm-api-server` 最值得复用的不是代码，而是下面这套 blueprint：

```text
Intent
  -> Action
     -> Resource Resolution
        -> Risk / Approval
           -> Execution Plan
              -> Task / Subtask
                 -> Adapter (MCP or controlled executor)
                    -> Audit / Evidence / Artifact
```

而这正是本仓库已经隐含具备、但尚未抽象为独立 assistant control layer 的核心价值。
